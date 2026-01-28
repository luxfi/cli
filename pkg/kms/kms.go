// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"sync"
	"time"
)

// KeyType represents the type of cryptographic key.
type KeyType string

const (
	KeyTypeAES256    KeyType = "aes-256-gcm"
	KeyTypeRSA3072   KeyType = "rsa-3072"
	KeyTypeRSA4096   KeyType = "rsa-4096"
	KeyTypeECDSAP256 KeyType = "ecdsa-p256"
	KeyTypeECDSAP384 KeyType = "ecdsa-p384"
	KeyTypeEdDSA     KeyType = "ed25519"
)

// KeyUsage represents what a key can be used for.
type KeyUsage string

const (
	KeyUsageEncryptDecrypt KeyUsage = "encrypt-decrypt"
	KeyUsageSignVerify     KeyUsage = "sign-verify"
	KeyUsageMPC            KeyUsage = "mpc"
)

// KeyStatus represents the current state of a key.
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusInactive KeyStatus = "inactive"
	KeyStatusDeleted  KeyStatus = "deleted"
	KeyStatusPending  KeyStatus = "pending" // For MPC key generation
)

// Key represents a cryptographic key in the KMS.
type Key struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        KeyType           `json:"type"`
	Usage       KeyUsage          `json:"usage"`
	Status      KeyStatus         `json:"status"`
	Version     int               `json:"version"`
	OrgID       string            `json:"orgId,omitempty"`
	ProjectID   string            `json:"projectId,omitempty"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
	ExpiresAt   *time.Time        `json:"expiresAt,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	// For MPC keys
	Threshold    int      `json:"threshold,omitempty"`
	TotalShares  int      `json:"totalShares,omitempty"`
	ShareHolders []string `json:"shareHolders,omitempty"`
}

// KeyMaterial holds the encrypted key material.
type KeyMaterial struct {
	KeyID            string    `json:"keyId"`
	Version          int       `json:"version"`
	EncryptedKey     []byte    `json:"encryptedKey"`     // Encrypted with root key
	EncryptedPrivate []byte    `json:"encryptedPrivate"` // For asymmetric keys
	PublicKey        []byte    `json:"publicKey"`        // Public key (if asymmetric)
	Nonce            []byte    `json:"nonce"`
	Created          time.Time `json:"created"`
}

// Secret represents a stored secret.
type Secret struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     int               `json:"version"`
	KeyID       string            `json:"keyId"`       // KMS key used for encryption
	Environment string            `json:"environment"` // dev, staging, prod
	Path        string            `json:"path"`        // Folder path
	Value       []byte            `json:"value"`       // Encrypted value
	Nonce       []byte            `json:"nonce"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	OrgID       string            `json:"orgId,omitempty"`
	ProjectID   string            `json:"projectId,omitempty"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
}

// KMS provides key management and encryption services.
type KMS struct {
	store      StorageBackend
	rootKey    []byte // 32-byte root encryption key
	rootCipher cipher.AEAD
	mu         sync.RWMutex
}

// Config holds KMS configuration.
type Config struct {
	Store       StorageBackend
	RootKey     []byte // Must be 32 bytes for AES-256
	DataDir     string // Only used if Store is nil (creates BadgerStore)
	InMemory    bool
	Compression bool
}

// New creates a new KMS instance.
func New(cfg *Config) (*KMS, error) {
	if len(cfg.RootKey) != 32 {
		return nil, fmt.Errorf("root key must be 32 bytes")
	}

	store := cfg.Store
	if store == nil {
		badgerCfg := &BadgerConfig{
			Dir:         cfg.DataDir,
			InMemory:    cfg.InMemory,
			SyncWrites:  true,
			Compression: cfg.Compression,
		}
		var err error
		store, err = NewBadgerStore(badgerCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create store: %w", err)
		}
	}

	// Create root cipher
	block, err := aes.NewCipher(cfg.RootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create root cipher: %w", err)
	}

	rootCipher, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &KMS{
		store:      store,
		rootKey:    cfg.RootKey,
		rootCipher: rootCipher,
	}, nil
}

// Key prefix constants
const (
	keyPrefix         = "kms/key/"
	keyMaterialPrefix = "kms/material/"
	secretPrefix      = "kms/secret/"
)

// GenerateKey generates a new cryptographic key.
func (k *KMS) GenerateKey(ctx context.Context, name string, keyType KeyType, usage KeyUsage, opts *KeyOptions) (*Key, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	keyID := generateID(16)
	now := time.Now()

	key := &Key{
		ID:      keyID,
		Name:    name,
		Type:    keyType,
		Usage:   usage,
		Status:  KeyStatusActive,
		Version: 1,
		Created: now,
		Updated: now,
	}

	if opts != nil {
		key.Description = opts.Description
		key.OrgID = opts.OrgID
		key.ProjectID = opts.ProjectID
		key.Metadata = opts.Metadata
		if opts.ExpiresIn > 0 {
			exp := now.Add(opts.ExpiresIn)
			key.ExpiresAt = &exp
		}
	}

	// Generate key material
	material, err := k.generateKeyMaterial(keyID, keyType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key material: %w", err)
	}

	// Save key
	if err := SetJSON(ctx, k.store, keyPrefix+keyID, key); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	// Save encrypted key material
	if err := SetJSON(ctx, k.store, keyMaterialPrefix+keyID+"/1", material); err != nil {
		return nil, fmt.Errorf("failed to save key material: %w", err)
	}

	return key, nil
}

// KeyOptions holds optional parameters for key generation.
type KeyOptions struct {
	Description string
	OrgID       string
	ProjectID   string
	Metadata    map[string]string
	ExpiresIn   time.Duration
}

// generateKeyMaterial creates and encrypts key material.
func (k *KMS) generateKeyMaterial(keyID string, keyType KeyType) (*KeyMaterial, error) {
	material := &KeyMaterial{
		KeyID:   keyID,
		Version: 1,
		Created: time.Now(),
	}

	var keyBytes []byte
	var privateBytes []byte
	var publicBytes []byte

	switch keyType {
	case KeyTypeAES256:
		keyBytes = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
			return nil, err
		}

	case KeyTypeRSA3072:
		key, err := rsa.GenerateKey(rand.Reader, 3072)
		if err != nil {
			return nil, err
		}
		privateBytes = x509.MarshalPKCS1PrivateKey(key)
		publicBytes, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return nil, err
		}

	case KeyTypeRSA4096:
		key, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, err
		}
		privateBytes = x509.MarshalPKCS1PrivateKey(key)
		publicBytes, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return nil, err
		}

	case KeyTypeECDSAP256:
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		privateBytes, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		publicBytes, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return nil, err
		}

	case KeyTypeECDSAP384:
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return nil, err
		}
		privateBytes, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		publicBytes, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return nil, err
		}

	case KeyTypeEdDSA:
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		privateBytes = priv
		publicBytes = pub

	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	// Generate nonce for encryption
	nonce := make([]byte, k.rootCipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	material.Nonce = nonce

	// Encrypt key material with root key
	if len(keyBytes) > 0 {
		material.EncryptedKey = k.rootCipher.Seal(nil, nonce, keyBytes, nil)
	}
	if len(privateBytes) > 0 {
		material.EncryptedPrivate = k.rootCipher.Seal(nil, nonce, privateBytes, nil)
	}
	material.PublicKey = publicBytes

	return material, nil
}

// GetKey retrieves a key by ID.
func (k *KMS) GetKey(ctx context.Context, keyID string) (*Key, error) {
	return GetJSON[Key](ctx, k.store, keyPrefix+keyID)
}

// GetKeyByName retrieves a key by name.
func (k *KMS) GetKeyByName(ctx context.Context, name string) (*Key, error) {
	keys, err := k.ListKeys(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if key.Name == name {
			return key, nil
		}
	}
	return nil, ErrKeyNotFound
}

// ListKeys lists all keys, optionally filtered by prefix.
func (k *KMS) ListKeys(ctx context.Context, prefix string) ([]*Key, error) {
	var keys []*Key
	err := k.store.Scan(ctx, keyPrefix+prefix, func(key string, value []byte) error {
		var kmsKey Key
		if err := json.Unmarshal(value, &kmsKey); err != nil {
			return nil // Skip invalid entries
		}
		if kmsKey.Status != KeyStatusDeleted {
			keys = append(keys, &kmsKey)
		}
		return nil
	})
	return keys, err
}

// DeleteKey soft-deletes a key.
func (k *KMS) DeleteKey(ctx context.Context, keyID string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	key, err := k.GetKey(ctx, keyID)
	if err != nil {
		return err
	}

	key.Status = KeyStatusDeleted
	key.Updated = time.Now()

	return SetJSON(ctx, k.store, keyPrefix+keyID, key)
}

// Encrypt encrypts data using the specified key.
func (k *KMS) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	key, err := k.GetKey(ctx, keyID)
	if err != nil {
		return nil, err
	}

	if key.Usage != KeyUsageEncryptDecrypt {
		return nil, fmt.Errorf("key %s cannot be used for encryption", keyID)
	}

	if key.Status != KeyStatusActive {
		return nil, fmt.Errorf("key %s is not active", keyID)
	}

	// Get key material
	material, err := k.getKeyMaterial(ctx, keyID, key.Version)
	if err != nil {
		return nil, err
	}

	// Decrypt the key material
	decryptedKey, err := k.rootCipher.Open(nil, material.Nonce, material.EncryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key material: %w", err)
	}

	// Create cipher from decrypted key
	block, err := aes.NewCipher(decryptedKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce for this encryption
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Prepend key version for decryption
	result := &EncryptedData{
		KeyID:      keyID,
		KeyVersion: key.Version,
		Data:       ciphertext,
	}

	return json.Marshal(result)
}

// EncryptedData represents encrypted data with metadata.
type EncryptedData struct {
	KeyID      string `json:"keyId"`
	KeyVersion int    `json:"keyVersion"`
	Data       []byte `json:"data"`
}

// Decrypt decrypts data.
func (k *KMS) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	var encrypted EncryptedData
	if err := json.Unmarshal(ciphertext, &encrypted); err != nil {
		return nil, fmt.Errorf("invalid ciphertext format: %w", err)
	}

	// Get key material
	material, err := k.getKeyMaterial(ctx, encrypted.KeyID, encrypted.KeyVersion)
	if err != nil {
		return nil, err
	}

	// Decrypt the key material
	decryptedKey, err := k.rootCipher.Open(nil, material.Nonce, material.EncryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key material: %w", err)
	}

	// Create cipher from decrypted key
	block, err := aes.NewCipher(decryptedKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(encrypted.Data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, data := encrypted.Data[:nonceSize], encrypted.Data[nonceSize:]

	// Decrypt
	return gcm.Open(nil, nonce, data, nil)
}

// getKeyMaterial retrieves and returns key material.
func (k *KMS) getKeyMaterial(ctx context.Context, keyID string, version int) (*KeyMaterial, error) {
	key := fmt.Sprintf("%s%s/%d", keyMaterialPrefix, keyID, version)
	return GetJSON[KeyMaterial](ctx, k.store, key)
}

// GetPublicKey returns the public key for an asymmetric key.
func (k *KMS) GetPublicKey(ctx context.Context, keyID string) ([]byte, error) {
	key, err := k.GetKey(ctx, keyID)
	if err != nil {
		return nil, err
	}

	material, err := k.getKeyMaterial(ctx, keyID, key.Version)
	if err != nil {
		return nil, err
	}

	if len(material.PublicKey) == 0 {
		return nil, fmt.Errorf("key %s is not an asymmetric key", keyID)
	}

	// Return PEM-encoded public key
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: material.PublicKey,
	}), nil
}

// Sign signs data using an asymmetric key.
func (k *KMS) Sign(ctx context.Context, keyID string, data []byte) ([]byte, error) {
	key, err := k.GetKey(ctx, keyID)
	if err != nil {
		return nil, err
	}

	if key.Usage != KeyUsageSignVerify {
		return nil, fmt.Errorf("key %s cannot be used for signing", keyID)
	}

	material, err := k.getKeyMaterial(ctx, keyID, key.Version)
	if err != nil {
		return nil, err
	}

	// Decrypt private key
	privateBytes, err := k.rootCipher.Open(nil, material.Nonce, material.EncryptedPrivate, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	hash := sha256.Sum256(data)

	switch key.Type {
	case KeyTypeRSA3072, KeyTypeRSA4096:
		privateKey, err := x509.ParsePKCS1PrivateKey(privateBytes)
		if err != nil {
			return nil, err
		}
		return rsa.SignPKCS1v15(rand.Reader, privateKey, 0, hash[:])

	case KeyTypeECDSAP256, KeyTypeECDSAP384:
		privateKey, err := x509.ParseECPrivateKey(privateBytes)
		if err != nil {
			return nil, err
		}
		return ecdsa.SignASN1(rand.Reader, privateKey, hash[:])

	case KeyTypeEdDSA:
		return ed25519.Sign(privateBytes, data), nil

	default:
		return nil, fmt.Errorf("unsupported key type for signing: %s", key.Type)
	}
}

// Verify verifies a signature.
func (k *KMS) Verify(ctx context.Context, keyID string, data, signature []byte) (bool, error) {
	key, err := k.GetKey(ctx, keyID)
	if err != nil {
		return false, err
	}

	material, err := k.getKeyMaterial(ctx, keyID, key.Version)
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(data)

	switch key.Type {
	case KeyTypeRSA3072, KeyTypeRSA4096:
		pub, err := x509.ParsePKIXPublicKey(material.PublicKey)
		if err != nil {
			return false, err
		}
		rsaPub := pub.(*rsa.PublicKey)
		err = rsa.VerifyPKCS1v15(rsaPub, 0, hash[:], signature)
		return err == nil, nil

	case KeyTypeECDSAP256, KeyTypeECDSAP384:
		pub, err := x509.ParsePKIXPublicKey(material.PublicKey)
		if err != nil {
			return false, err
		}
		ecdsaPub := pub.(*ecdsa.PublicKey)
		return ecdsa.VerifyASN1(ecdsaPub, hash[:], signature), nil

	case KeyTypeEdDSA:
		return ed25519.Verify(material.PublicKey, data, signature), nil

	default:
		return false, fmt.Errorf("unsupported key type for verification: %s", key.Type)
	}
}

// Secret Management

// CreateSecret creates a new secret.
func (k *KMS) CreateSecret(ctx context.Context, name string, value []byte, opts *SecretOptions) (*Secret, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	secretID := generateID(16)
	now := time.Now()

	// Get or create encryption key
	keyID := ""
	if opts != nil && opts.KeyID != "" {
		keyID = opts.KeyID
	} else {
		// Use default project key or create one
		key, err := k.GetKeyByName(ctx, "default")
		if err == ErrKeyNotFound {
			key, err = k.GenerateKey(ctx, "default", KeyTypeAES256, KeyUsageEncryptDecrypt, nil)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get encryption key: %w", err)
		}
		keyID = key.ID
	}

	// Encrypt the secret value
	encrypted, err := k.Encrypt(ctx, keyID, value)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secret := &Secret{
		ID:      secretID,
		Name:    name,
		Version: 1,
		KeyID:   keyID,
		Value:   encrypted,
		Created: now,
		Updated: now,
	}

	if opts != nil {
		secret.Environment = opts.Environment
		secret.Path = opts.Path
		secret.Tags = opts.Tags
		secret.Metadata = opts.Metadata
		secret.OrgID = opts.OrgID
		secret.ProjectID = opts.ProjectID
	}

	if err := SetJSON(ctx, k.store, secretPrefix+secretID, secret); err != nil {
		return nil, fmt.Errorf("failed to save secret: %w", err)
	}

	return secret, nil
}

// SecretOptions holds options for secret creation.
type SecretOptions struct {
	KeyID       string
	Environment string
	Path        string
	Tags        []string
	Metadata    map[string]string
	OrgID       string
	ProjectID   string
}

// GetSecret retrieves a secret by ID.
func (k *KMS) GetSecret(ctx context.Context, secretID string) (*Secret, error) {
	return GetJSON[Secret](ctx, k.store, secretPrefix+secretID)
}

// GetSecretValue retrieves and decrypts a secret value.
func (k *KMS) GetSecretValue(ctx context.Context, secretID string) ([]byte, error) {
	secret, err := k.GetSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	return k.Decrypt(ctx, secret.Value)
}

// ListSecrets lists secrets, optionally filtered by environment or path.
func (k *KMS) ListSecrets(ctx context.Context, env, path string) ([]*Secret, error) {
	var secrets []*Secret
	err := k.store.Scan(ctx, secretPrefix, func(key string, value []byte) error {
		var secret Secret
		if err := json.Unmarshal(value, &secret); err != nil {
			return nil
		}
		if (env == "" || secret.Environment == env) && (path == "" || secret.Path == path) {
			secrets = append(secrets, &secret)
		}
		return nil
	})
	return secrets, err
}

// UpdateSecret updates a secret's value.
func (k *KMS) UpdateSecret(ctx context.Context, secretID string, newValue []byte) (*Secret, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	secret, err := k.GetSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	// Encrypt the new value
	encrypted, err := k.Encrypt(ctx, secret.KeyID, newValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secret.Value = encrypted
	secret.Version++
	secret.Updated = time.Now()

	if err := SetJSON(ctx, k.store, secretPrefix+secretID, secret); err != nil {
		return nil, fmt.Errorf("failed to save secret: %w", err)
	}

	return secret, nil
}

// DeleteSecret deletes a secret.
func (k *KMS) DeleteSecret(ctx context.Context, secretID string) error {
	return k.store.Delete(ctx, secretPrefix+secretID)
}

// Close closes the KMS and underlying storage.
func (k *KMS) Close() error {
	return k.store.Close()
}

// Helper functions

func generateID(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// EncodeBase64 encodes bytes to base64.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 to bytes.
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
