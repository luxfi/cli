// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build darwin

package key

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/crypto/secp256k1"
)

// KeychainBackend uses macOS Keychain with optional TouchID/biometrics
type KeychainBackend struct {
	dataDir   string
	sessions  map[string]*keySession
	sessionMu sync.RWMutex
}

const (
	keychainService = "io.lux.cli"
	keychainAccess  = "Lux CLI Key Management"
)

// NewKeychainBackend creates a macOS Keychain backend
func NewKeychainBackend() *KeychainBackend {
	return &KeychainBackend{
		sessions: make(map[string]*keySession),
	}
}

func (*KeychainBackend) Type() BackendType {
	return BackendKeychain
}

func (*KeychainBackend) Name() string {
	return "macOS Keychain (TouchID)"
}

func (*KeychainBackend) Available() bool {
	// Check if security command is available
	_, err := exec.LookPath("security")
	return err == nil
}

func (*KeychainBackend) RequiresPassword() bool {
	return false // Uses biometrics or keychain password
}

func (*KeychainBackend) RequiresHardware() bool {
	return false
}

func (*KeychainBackend) SupportsRemoteSigning() bool {
	return false
}

func (b *KeychainBackend) Initialize(ctx context.Context) error {
	if b.dataDir == "" {
		keysDir, err := GetKeysDir()
		if err != nil {
			return err
		}
		b.dataDir = keysDir
	}
	return os.MkdirAll(b.dataDir, 0o700)
}

func (b *KeychainBackend) Close() error {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	for _, s := range b.sessions {
		for i := range s.key {
			s.key[i] = 0
		}
	}
	b.sessions = make(map[string]*keySession)
	return nil
}

func (b *KeychainBackend) CreateKey(ctx context.Context, name string, opts CreateKeyOptions) (*HDKeySet, error) {
	keyDir := filepath.Join(b.dataDir, name)

	if _, err := os.Stat(keyDir); err == nil {
		return nil, ErrKeyExists
	}

	// Generate mnemonic
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

	// Derive keys
	keySet, err := DeriveAllKeys(name, mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to derive keys: %w", err)
	}

	// Save to keychain
	if err := b.SaveKey(ctx, keySet, ""); err != nil {
		return nil, err
	}

	return keySet, nil
}

func (b *KeychainBackend) LoadKey(ctx context.Context, name, password string) (*HDKeySet, error) {
	// Check session cache
	b.sessionMu.RLock()
	if s, ok := b.sessions[name]; ok && time.Now().Before(s.expiresAt) {
		b.sessionMu.RUnlock()
		return b.loadFromSession(name, s.key)
	}
	b.sessionMu.RUnlock()

	// Read from keychain (will prompt for TouchID or password)
	data, err := b.readFromKeychain(name)
	if err != nil {
		if strings.Contains(err.Error(), "could not be found") {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("keychain read failed: %w", err)
	}

	keySet, err := parseKeySetJSON(data)
	if err != nil {
		return nil, err
	}

	// Cache in session
	b.sessionMu.Lock()
	b.sessions[name] = &keySession{
		name:       name,
		key:        data,
		unlockedAt: time.Now(),
		expiresAt:  time.Now().Add(sessionTimeout),
	}
	b.sessionMu.Unlock()

	return keySet, nil
}

func (b *KeychainBackend) SaveKey(ctx context.Context, keySet *HDKeySet, password string) error {
	keyDir := filepath.Join(b.dataDir, keySet.Name)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Serialize key set
	data, err := serializeKeySet(keySet)
	if err != nil {
		return fmt.Errorf("failed to serialize keys: %w", err)
	}

	// Store in keychain with biometric protection
	if err := b.writeToKeychain(keySet.Name, data); err != nil {
		return fmt.Errorf("keychain write failed: %w", err)
	}

	// Write public info
	pubInfo := map[string]interface{}{
		"name":       keySet.Name,
		"ec_address": keySet.ECAddress,
		"node_id":    keySet.NodeID,
		"created_at": time.Now().Format(time.RFC3339),
		"backend":    string(BackendKeychain),
	}
	pubData, _ := json.MarshalIndent(pubInfo, "", "  ")
	_ = os.WriteFile(filepath.Join(keyDir, "info.json"), pubData, 0o644) //nolint:gosec // G306: Public info file needs to be readable

	return nil
}

func (b *KeychainBackend) DeleteKey(ctx context.Context, name string) error {
	// Remove from keychain
	if err := b.deleteFromKeychain(name); err != nil {
		// Ignore if not found
		if !strings.Contains(err.Error(), "could not be found") {
			return err
		}
	}

	// Remove session
	b.sessionMu.Lock()
	if s, ok := b.sessions[name]; ok {
		for i := range s.key {
			s.key[i] = 0
		}
		delete(b.sessions, name)
	}
	b.sessionMu.Unlock()

	// Remove local files
	keyDir := filepath.Join(b.dataDir, name)
	return os.RemoveAll(keyDir)
}

func (b *KeychainBackend) ListKeys(ctx context.Context) ([]KeyInfo, error) {
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

		// Check if stored in keychain
		pubPath := filepath.Join(keyDir, "info.json")
		if data, err := os.ReadFile(pubPath); err == nil { //nolint:gosec // G304: Reading from user's key directory
			var pubInfo struct {
				ECAddress string `json:"ec_address"`
				NodeID    string `json:"node_id"`
				CreatedAt string `json:"created_at"`
				Backend   string `json:"backend"`
			}
			if json.Unmarshal(data, &pubInfo) == nil {
				if pubInfo.Backend != string(BackendKeychain) {
					continue // Not a keychain key
				}
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

func (b *KeychainBackend) Lock(ctx context.Context, name string) error {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	if s, ok := b.sessions[name]; ok {
		for i := range s.key {
			s.key[i] = 0
		}
		delete(b.sessions, name)
	}
	return nil
}

func (b *KeychainBackend) Unlock(ctx context.Context, name, password string) error {
	_, err := b.LoadKey(ctx, name, password)
	return err
}

func (b *KeychainBackend) IsLocked(name string) bool {
	b.sessionMu.RLock()
	defer b.sessionMu.RUnlock()

	s, ok := b.sessions[name]
	if !ok {
		return true
	}
	return time.Now().After(s.expiresAt)
}

func (b *KeychainBackend) Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error) {
	keySet, err := b.LoadKey(ctx, name, "")
	if err != nil {
		return nil, err
	}

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

// Keychain operations using security command

func (b *KeychainBackend) writeToKeychain(name string, data []byte) error {
	account := fmt.Sprintf("lux-key-%s", name)

	// Delete existing item if present
	_ = b.deleteFromKeychain(name)

	// Add new item with access control for biometrics
	// -T "" allows access without app confirmation
	// -w stores the data as password
	cmd := exec.Command("security", "add-generic-password", //nolint:gosec // G204: Intentional keychain command
		"-a", account,
		"-s", keychainService,
		"-l", fmt.Sprintf("%s: %s", keychainAccess, name),
		"-w", hex.EncodeToString(data),
		"-T", "", // Allow access without confirmation
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security add-generic-password failed: %s: %w", string(output), err)
	}

	return nil
}

func (*KeychainBackend) readFromKeychain(name string) ([]byte, error) {
	account := fmt.Sprintf("lux-key-%s", name)

	cmd := exec.Command("security", "find-generic-password", //nolint:gosec // G204: Intentional keychain command
		"-a", account,
		"-s", keychainService,
		"-w", // Output password only
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("security find-generic-password failed: %w", err)
	}

	// Decode hex
	hexData := strings.TrimSpace(string(output))
	return hex.DecodeString(hexData)
}

func (*KeychainBackend) deleteFromKeychain(name string) error {
	account := fmt.Sprintf("lux-key-%s", name)

	cmd := exec.Command("security", "delete-generic-password", //nolint:gosec // G204: Intentional keychain command
		"-a", account,
		"-s", keychainService,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security delete-generic-password failed: %s: %w", string(output), err)
	}

	return nil
}

func (*KeychainBackend) loadFromSession(_ string, data []byte) (*HDKeySet, error) {
	return parseKeySetJSON(append([]byte{}, data...))
}

func (b *KeychainBackend) GetKeyChecksum(name string) (string, error) {
	ks, err := b.LoadKey(context.Background(), name, "")
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(ks.ECPrivateKey)
	h.Write(ks.BLSPrivateKey)
	return hex.EncodeToString(h.Sum(nil)[:8]), nil
}

func init() {
	RegisterBackend(NewKeychainBackend())
}
