// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/luxfi/crypto/secp256k1"
	"golang.org/x/crypto/argon2"
)

// SoftwareBackend implements encrypted file-based key storage
type SoftwareBackend struct {
	dataDir        string
	sessions       map[string]*keySession
	sessionMu      sync.RWMutex
	sessionTimeout time.Duration // Configurable session timeout
}

type keySession struct {
	name       string
	key        []byte
	unlockedAt time.Time
	expiresAt  time.Time
	mlocked    bool // Whether the key memory is locked
}

// Argon2id parameters (OWASP recommended for password hashing)
const (
	argon2Time    = 3         // iterations
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32

	// DefaultSessionTimeout is the inactivity timeout for unlocked keys.
	// After this duration without access, the key is automatically locked.
	// Can be overridden via LUX_KEY_SESSION_TIMEOUT environment variable.
	DefaultSessionTimeout = 30 * time.Second
)

// NewSoftwareBackend creates a new software-based key backend
func NewSoftwareBackend() *SoftwareBackend {
	return &SoftwareBackend{
		sessions:       make(map[string]*keySession),
		sessionTimeout: GetSessionTimeout(),
	}
}

// GetSessionTimeout returns the configured session timeout.
// Checks LUX_KEY_SESSION_TIMEOUT environment variable first,
// otherwise returns DefaultSessionTimeout (30 seconds).
func GetSessionTimeout() time.Duration {
	if envTimeout := os.Getenv(EnvKeySessionTimeout); envTimeout != "" {
		if d, err := time.ParseDuration(envTimeout); err == nil && d > 0 {
			return d
		}
	}
	return DefaultSessionTimeout
}

// SetSessionTimeout sets the session timeout for this backend.
// The timeout resets on each key access (sliding window).
func (b *SoftwareBackend) SetSessionTimeout(d time.Duration) {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()
	if d > 0 {
		b.sessionTimeout = d
	}
}

func (*SoftwareBackend) Type() BackendType {
	return BackendSoftware
}

func (*SoftwareBackend) Name() string {
	return "Encrypted File Storage"
}

func (*SoftwareBackend) Available() bool {
	return true // Always available
}

func (*SoftwareBackend) RequiresPassword() bool {
	return true
}

func (*SoftwareBackend) RequiresHardware() bool {
	return false
}

func (*SoftwareBackend) SupportsRemoteSigning() bool {
	return false
}

func (b *SoftwareBackend) Initialize(ctx context.Context) error {
	if b.dataDir == "" {
		keysDir, err := GetKeysDir()
		if err != nil {
			return err
		}
		b.dataDir = keysDir
	}
	return os.MkdirAll(b.dataDir, 0o700)
}

func (b *SoftwareBackend) Close() error {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	// Securely clear all session keys
	for _, s := range b.sessions {
		clearSession(s)
	}
	b.sessions = make(map[string]*keySession)
	return nil
}

func (b *SoftwareBackend) CreateKey(ctx context.Context, name string, opts CreateKeyOptions) (*HDKeySet, error) {
	keyDir := filepath.Join(b.dataDir, name)

	// Check if key already exists
	if _, err := os.Stat(keyDir); err == nil {
		return nil, ErrKeyExists
	}

	// Generate or import mnemonic
	var mnemonic string
	if opts.Mnemonic != "" {
		if !ValidateMnemonic(opts.Mnemonic) {
			return nil, errors.New("invalid mnemonic phrase")
		}
		mnemonic = opts.Mnemonic
	} else {
		var err error
		mnemonic, err = GenerateMnemonic()
		if err != nil {
			return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
		}
	}

	// Derive all keys from mnemonic
	keySet, err := DeriveAllKeys(name, mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to derive keys: %w", err)
	}

	// Save encrypted
	if err := b.SaveKey(ctx, keySet, opts.Password); err != nil {
		return nil, err
	}

	return keySet, nil
}

func (b *SoftwareBackend) LoadKey(ctx context.Context, name, password string) (*HDKeySet, error) {
	// Check for active session
	if session := b.getSession(name); session != nil {
		return b.loadWithKey(name, session.key)
	}

	// Need password
	if password == "" {
		password = os.Getenv(EnvKeyPassword)
		if password == "" {
			return nil, ErrKeyLocked
		}
	}

	keyDir := filepath.Join(b.dataDir, name)
	encPath := filepath.Join(keyDir, "keystore.enc")

	data, err := os.ReadFile(encPath) //nolint:gosec // G304: Reading from user's key directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to read keystore: %w", err)
	}

	var store encryptedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse keystore: %w", err)
	}

	// Derive encryption key
	encKey := argon2.IDKey([]byte(password), store.Salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Decrypt
	plaintext, err := decryptAESGCM(encKey, store.Nonce, store.Data)
	if err != nil {
		// Zero key on failure
		for i := range encKey {
			encKey[i] = 0
		}
		return nil, ErrInvalidPassword
	}

	// Store session
	b.setSession(name, encKey)

	return parseKeySetJSON(plaintext)
}

func (b *SoftwareBackend) SaveKey(ctx context.Context, keySet *HDKeySet, password string) error {
	if password == "" {
		return ErrNoPassword
	}

	keyDir := filepath.Join(b.dataDir, keySet.Name)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Generate salt
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key
	encKey := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	defer func() {
		for i := range encKey {
			encKey[i] = 0
		}
	}()

	// Serialize key set
	plaintext, err := serializeKeySet(keySet)
	if err != nil {
		return fmt.Errorf("failed to serialize keys: %w", err)
	}
	defer func() {
		for i := range plaintext {
			plaintext[i] = 0
		}
	}()

	// Encrypt
	nonce, ciphertext, err := encryptAESGCM(encKey, plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	store := encryptedStore{
		Version:   1,
		Salt:      salt,
		Nonce:     nonce,
		Data:      ciphertext,
		CreatedAt: time.Now().Unix(),
	}

	storeData, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	// Write encrypted keystore
	encPath := filepath.Join(keyDir, "keystore.enc")
	if err := os.WriteFile(encPath, storeData, 0o600); err != nil {
		return fmt.Errorf("failed to write keystore: %w", err)
	}

	// Write public info (viewable without unlock)
	pubInfo := map[string]interface{}{
		"name":       keySet.Name,
		"ec_address": keySet.ECAddress,
		"node_id":    keySet.NodeID,
		"created_at": time.Now().Format(time.RFC3339),
		"backend":    string(BackendSoftware),
	}
	pubData, _ := json.MarshalIndent(pubInfo, "", "  ")
	_ = os.WriteFile(filepath.Join(keyDir, "info.json"), pubData, 0o644) //nolint:gosec // G306: Public info file needs to be readable

	return nil
}

func (b *SoftwareBackend) DeleteKey(ctx context.Context, name string) error {
	keyDir := filepath.Join(b.dataDir, name)

	// Remove session
	b.sessionMu.Lock()
	if s, ok := b.sessions[name]; ok {
		clearSession(s)
		delete(b.sessions, name)
	}
	b.sessionMu.Unlock()

	return os.RemoveAll(keyDir)
}

func (b *SoftwareBackend) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	entries, err := os.ReadDir(b.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []KeyInfo{}, nil
		}
		return nil, err
	}

	keys := make([]KeyInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		keyDir := filepath.Join(b.dataDir, name)

		info := KeyInfo{
			Name:      name,
			Encrypted: true,
			Locked:    b.IsLocked(name),
		}

		// Read public info
		pubPath := filepath.Join(keyDir, "info.json")
		if data, err := os.ReadFile(pubPath); err == nil { //nolint:gosec // G304: Reading from user's key directory
			var pubInfo struct {
				ECAddress string `json:"ec_address"`
				NodeID    string `json:"node_id"`
				CreatedAt string `json:"created_at"`
			}
			if json.Unmarshal(data, &pubInfo) == nil {
				info.Address = pubInfo.ECAddress
				info.NodeID = pubInfo.NodeID
				if t, err := time.Parse(time.RFC3339, pubInfo.CreatedAt); err == nil {
					info.CreatedAt = t
				}
			}
		}

		keys = append(keys, info)
	}

	return keys, nil
}

func (b *SoftwareBackend) Lock(ctx context.Context, name string) error {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	if s, ok := b.sessions[name]; ok {
		clearSession(s)
		delete(b.sessions, name)
	}
	return nil
}

func (b *SoftwareBackend) Unlock(ctx context.Context, name, password string) error {
	_, err := b.LoadKey(ctx, name, password)
	return err
}

func (b *SoftwareBackend) IsLocked(name string) bool {
	return b.getSession(name) == nil
}

func (b *SoftwareBackend) Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error) {
	keySet, err := b.LoadKey(ctx, name, "")
	if err != nil {
		return nil, err
	}

	// Sign based on key type needed
	privKey, err := secp256k1.ToPrivateKey(keySet.ECPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	sig, err := privKey.Sign(request.DataHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	return &SignResponse{
		Signature: sig,
		PublicKey: privKey.PublicKey().Bytes(),
		Address:   keySet.ECAddress,
	}, nil
}

// Helper methods

func (b *SoftwareBackend) getSession(name string) *keySession {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	s, ok := b.sessions[name]
	if !ok {
		return nil
	}

	if time.Now().After(s.expiresAt) {
		// Session expired - clear it securely
		clearSession(s)
		delete(b.sessions, name)
		return nil
	}

	// Extend on access (sliding window)
	s.expiresAt = time.Now().Add(b.sessionTimeout)
	return s
}

func (b *SoftwareBackend) setSession(name string, key []byte) {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	// Clear existing session if present
	if existing, ok := b.sessions[name]; ok {
		clearSession(existing)
	}

	// Attempt to lock the key memory to prevent swapping
	mlocked := false
	if err := mlock(key); err == nil {
		mlocked = true
	}

	b.sessions[name] = &keySession{
		name:       name,
		key:        key,
		unlockedAt: time.Now(),
		expiresAt:  time.Now().Add(b.sessionTimeout),
		mlocked:    mlocked,
	}
}

// clearSession securely clears a session, zeroing the key and unlocking memory.
func clearSession(s *keySession) {
	if s == nil {
		return
	}
	// Unlock memory before zeroing
	if s.mlocked {
		_ = munlock(s.key)
	}
	// Zero out the key
	for i := range s.key {
		s.key[i] = 0
	}
}

func (b *SoftwareBackend) loadWithKey(name string, encKey []byte) (*HDKeySet, error) {
	keyDir := filepath.Join(b.dataDir, name)
	encPath := filepath.Join(keyDir, "keystore.enc")

	data, err := os.ReadFile(encPath) //nolint:gosec // G304: Reading from user's key directory
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore: %w", err)
	}

	var store encryptedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse keystore: %w", err)
	}

	plaintext, err := decryptAESGCM(encKey, store.Nonce, store.Data)
	if err != nil {
		return nil, ErrInvalidPassword
	}

	return parseKeySetJSON(plaintext)
}

// Encryption helpers

type encryptedStore struct {
	Version   int    `json:"version"`
	Salt      []byte `json:"salt"`
	Nonce     []byte `json:"nonce"`
	Data      []byte `json:"data"`
	CreatedAt int64  `json:"created_at"`
}

func encryptAESGCM(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

func decryptAESGCM(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

func serializeKeySet(ks *HDKeySet) ([]byte, error) {
	data := struct {
		Name               string `json:"name"`
		ECPrivateKey       string `json:"ec_private_key"`
		ECPublicKey        string `json:"ec_public_key"`
		ECAddress          string `json:"ec_address"`
		BLSPrivateKey      string `json:"bls_private_key"`
		BLSPublicKey       string `json:"bls_public_key"`
		BLSPoP             string `json:"bls_pop"`
		RingtailPrivateKey string `json:"ringtail_private_key"`
		RingtailPublicKey  string `json:"ringtail_public_key"`
		MLDSAPrivateKey    string `json:"mldsa_private_key"`
		MLDSAPublicKey     string `json:"mldsa_public_key"`
		StakingKeyPEM      string `json:"staking_key_pem"`
		StakingCertPEM     string `json:"staking_cert_pem"`
		NodeID             string `json:"node_id"`
	}{
		Name:               ks.Name,
		ECPrivateKey:       hex.EncodeToString(ks.ECPrivateKey),
		ECPublicKey:        hex.EncodeToString(ks.ECPublicKey),
		ECAddress:          ks.ECAddress,
		BLSPrivateKey:      hex.EncodeToString(ks.BLSPrivateKey),
		BLSPublicKey:       hex.EncodeToString(ks.BLSPublicKey),
		BLSPoP:             hex.EncodeToString(ks.BLSPoP),
		RingtailPrivateKey: hex.EncodeToString(ks.RingtailPrivateKey),
		RingtailPublicKey:  hex.EncodeToString(ks.RingtailPublicKey),
		MLDSAPrivateKey:    hex.EncodeToString(ks.MLDSAPrivateKey),
		MLDSAPublicKey:     hex.EncodeToString(ks.MLDSAPublicKey),
		StakingKeyPEM:      string(ks.StakingKeyPEM),
		StakingCertPEM:     string(ks.StakingCertPEM),
		NodeID:             ks.NodeID,
	}
	return json.Marshal(data)
}

func parseKeySetJSON(data []byte) (*HDKeySet, error) {
	defer func() {
		for i := range data {
			data[i] = 0
		}
	}()

	var raw struct {
		Name               string `json:"name"`
		ECPrivateKey       string `json:"ec_private_key"`
		ECPublicKey        string `json:"ec_public_key"`
		ECAddress          string `json:"ec_address"`
		BLSPrivateKey      string `json:"bls_private_key"`
		BLSPublicKey       string `json:"bls_public_key"`
		BLSPoP             string `json:"bls_pop"`
		RingtailPrivateKey string `json:"ringtail_private_key"`
		RingtailPublicKey  string `json:"ringtail_public_key"`
		MLDSAPrivateKey    string `json:"mldsa_private_key"`
		MLDSAPublicKey     string `json:"mldsa_public_key"`
		StakingKeyPEM      string `json:"staking_key_pem"`
		StakingCertPEM     string `json:"staking_cert_pem"`
		NodeID             string `json:"node_id"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	ks := &HDKeySet{
		Name:           raw.Name,
		ECAddress:      raw.ECAddress,
		StakingKeyPEM:  []byte(raw.StakingKeyPEM),
		StakingCertPEM: []byte(raw.StakingCertPEM),
		NodeID:         raw.NodeID,
	}

	var err error
	ks.ECPrivateKey, err = hex.DecodeString(raw.ECPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode ec private key: %w", err)
	}
	ks.ECPublicKey, err = hex.DecodeString(raw.ECPublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode ec public key: %w", err)
	}
	ks.BLSPrivateKey, err = hex.DecodeString(raw.BLSPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode bls private key: %w", err)
	}
	ks.BLSPublicKey, err = hex.DecodeString(raw.BLSPublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode bls public key: %w", err)
	}
	ks.BLSPoP, err = hex.DecodeString(raw.BLSPoP)
	if err != nil {
		return nil, fmt.Errorf("decode bls pop: %w", err)
	}
	ks.RingtailPrivateKey, err = hex.DecodeString(raw.RingtailPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode ringtail private key: %w", err)
	}
	ks.RingtailPublicKey, err = hex.DecodeString(raw.RingtailPublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode ringtail public key: %w", err)
	}
	ks.MLDSAPrivateKey, err = hex.DecodeString(raw.MLDSAPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode mldsa private key: %w", err)
	}
	ks.MLDSAPublicKey, err = hex.DecodeString(raw.MLDSAPublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode mldsa public key: %w", err)
	}

	return ks, nil
}

// GetKeyChecksum returns a checksum for key verification
func (b *SoftwareBackend) GetKeyChecksum(name string) (string, error) {
	session := b.getSession(name)
	if session == nil {
		return "", ErrKeyLocked
	}

	ks, err := b.loadWithKey(name, session.key)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(ks.ECPrivateKey)
	h.Write(ks.BLSPrivateKey)
	return hex.EncodeToString(h.Sum(nil)[:8]), nil
}

func init() {
	RegisterBackend(NewSoftwareBackend())
}
