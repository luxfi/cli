// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package key provides a pluggable key storage backend system supporting:
// - Software encrypted storage (AES-256-GCM + Argon2id)
// - macOS Keychain with TouchID/Biometrics
// - Linux Secret Service (GNOME Keyring, KWallet)
// - Hardware security modules (Zymbit, Yubikey)
// - Remote signing via WalletConnect/QR codes
// - Ledger hardware wallet (optional)

package key

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// BackendType identifies the key storage backend
type BackendType string

const (
	// BackendSoftware is the default encrypted file storage
	BackendSoftware BackendType = "software"

	// BackendKeychain uses macOS Keychain with optional TouchID
	BackendKeychain BackendType = "keychain"

	// BackendSecretService uses Linux Secret Service API (GNOME Keyring, KWallet)
	BackendSecretService BackendType = "secret-service"

	// BackendYubikey uses Yubikey for key storage/signing
	BackendYubikey BackendType = "yubikey"

	// BackendZymbit uses Zymbit HSM (Raspberry Pi hardware security)
	BackendZymbit BackendType = "zymbit"

	// BackendWalletConnect uses mobile wallet for remote signing
	BackendWalletConnect BackendType = "walletconnect"

	// BackendLedger uses Ledger hardware wallet (optional)
	BackendLedger BackendType = "ledger"

	// BackendEnv loads keys from environment variables
	BackendEnv BackendType = "env"
)

var (
	ErrBackendNotFound     = errors.New("key backend not found")
	ErrBackendNotSupported = errors.New("key backend not supported on this platform")
	ErrBackendUnavailable  = errors.New("key backend unavailable (check hardware/service)")
	ErrSigningCancelled    = errors.New("signing cancelled by user")
	ErrAuthFailed          = errors.New("authentication failed")
	ErrKeyLocked           = errors.New("key is locked, use 'lux key unlock' first")
	ErrKeyNotFound         = errors.New("key not found")
	ErrInvalidPassword     = errors.New("invalid password")
	ErrKeyExists           = errors.New("key already exists")
	ErrNoPassword          = errors.New("password required")
)

// KeyInfo represents information about a stored key
type KeyInfo struct {
	Name      string
	Address   string
	NodeID    string
	Encrypted bool
	Locked    bool
	CreatedAt time.Time
}

// SignRequest represents a transaction signing request
type SignRequest struct {
	Type        string // "transaction", "message", "auth"
	ChainID     uint64
	Description string
	Data        []byte   // Raw data to sign
	DataHash    [32]byte // Hash of data (for display)
}

// SignResponse contains the signature result
type SignResponse struct {
	Signature []byte
	PublicKey []byte
	Address   string
}

// KeyBackend defines the interface for all key storage backends
type KeyBackend interface {
	// Type returns the backend type identifier
	Type() BackendType

	// Name returns a human-readable name
	Name() string

	// Available checks if this backend is available on the current system
	Available() bool

	// RequiresPassword returns true if password is needed
	RequiresPassword() bool

	// RequiresHardware returns true if hardware device is needed
	RequiresHardware() bool

	// SupportsRemoteSigning returns true if signing is done externally
	SupportsRemoteSigning() bool

	// Initialize sets up the backend (creates directories, connects to services, etc.)
	Initialize(ctx context.Context) error

	// Close cleans up resources
	Close() error

	// CreateKey creates a new key set with the given name
	CreateKey(ctx context.Context, name string, opts CreateKeyOptions) (*HDKeySet, error)

	// LoadKey loads a key set by name
	LoadKey(ctx context.Context, name, password string) (*HDKeySet, error)

	// SaveKey saves a key set
	SaveKey(ctx context.Context, keySet *HDKeySet, password string) error

	// DeleteKey removes a key
	DeleteKey(ctx context.Context, name string) error

	// ListKeys returns all available keys
	ListKeys(ctx context.Context) ([]KeyInfo, error)

	// Lock locks a key (clears from memory)
	Lock(ctx context.Context, name string) error

	// Unlock unlocks a key for use
	Unlock(ctx context.Context, name, password string) error

	// IsLocked checks if a key is locked
	IsLocked(name string) bool

	// Sign signs data with the specified key
	Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error)
}

// CreateKeyOptions contains options for key creation
type CreateKeyOptions struct {
	// Mnemonic is an optional existing mnemonic phrase
	Mnemonic string

	// Password for encryption (software backend)
	Password string

	// UseBiometrics enables TouchID/FaceID on macOS
	UseBiometrics bool

	// YubikeySlot specifies the PIV slot for Yubikey
	YubikeySlot int

	// ImportOnly indicates we're importing, not generating
	ImportOnly bool
}

// backendRegistry holds all registered backends
var (
	backendMu      sync.RWMutex
	backends       = make(map[BackendType]KeyBackend)
	defaultBackend BackendType
	activeBackends = make(map[BackendType]KeyBackend)
)

// RegisterBackend registers a key backend
func RegisterBackend(b KeyBackend) {
	backendMu.Lock()
	defer backendMu.Unlock()
	backends[b.Type()] = b
}

// GetBackend returns a backend by type
func GetBackend(t BackendType) (KeyBackend, error) {
	backendMu.RLock()
	defer backendMu.RUnlock()

	b, ok := backends[t]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrBackendNotFound, t)
	}

	if !b.Available() {
		return nil, fmt.Errorf("%w: %s", ErrBackendNotSupported, t)
	}

	return b, nil
}

// GetDefaultBackend returns the default backend for the current platform
func GetDefaultBackend() (KeyBackend, error) {
	backendMu.RLock()
	defer backendMu.RUnlock()

	if defaultBackend != "" {
		if b, ok := backends[defaultBackend]; ok && b.Available() {
			return b, nil
		}
	}

	// Platform-specific defaults
	switch runtime.GOOS {
	case "darwin":
		// Prefer Keychain on macOS
		if b, ok := backends[BackendKeychain]; ok && b.Available() {
			return b, nil
		}
	case "linux":
		// Prefer Secret Service on Linux
		if b, ok := backends[BackendSecretService]; ok && b.Available() {
			return b, nil
		}
	}

	// Fall back to software backend
	if b, ok := backends[BackendSoftware]; ok {
		return b, nil
	}

	return nil, ErrBackendNotFound
}

// SetDefaultBackend sets the default backend type
func SetDefaultBackend(t BackendType) error {
	backendMu.Lock()
	defer backendMu.Unlock()

	if _, ok := backends[t]; !ok {
		return fmt.Errorf("%w: %s", ErrBackendNotFound, t)
	}

	defaultBackend = t
	return nil
}

// ListAvailableBackends returns all available backends
func ListAvailableBackends() []KeyBackend {
	backendMu.RLock()
	defer backendMu.RUnlock()

	var available []KeyBackend
	for _, b := range backends {
		if b.Available() {
			available = append(available, b)
		}
	}
	return available
}

// BackendConfig holds configuration for backend initialization
type BackendConfig struct {
	// DataDir is the base directory for key storage
	DataDir string

	// WalletConnectProjectID for WalletConnect backend
	WalletConnectProjectID string

	// ZymbitDevicePath for Zymbit HSM
	ZymbitDevicePath string

	// YubikeyPIN for Yubikey operations
	YubikeyPIN string
}

// InitializeBackends initializes all available backends
func InitializeBackends(ctx context.Context, config BackendConfig) error {
	backendMu.Lock()
	defer backendMu.Unlock()

	for t, b := range backends {
		if b.Available() {
			if err := b.Initialize(ctx); err != nil {
				// Log warning but don't fail - some backends may be optional
				fmt.Printf("Warning: failed to initialize %s backend: %v\n", t, err)
				continue
			}
			activeBackends[t] = b
		}
	}

	return nil
}

// CloseBackends closes all active backends
func CloseBackends() {
	backendMu.Lock()
	defer backendMu.Unlock()

	for _, b := range activeBackends {
		_ = b.Close()
	}
	activeBackends = make(map[BackendType]KeyBackend)
}

// SessionTimeout is the default session timeout for unlocked keys
const SessionTimeout = 15 * time.Minute

// LockKey locks a key using the default backend
func LockKey(name string) error {
	backend, err := GetDefaultBackend()
	if err != nil {
		return err
	}
	return backend.Lock(context.Background(), name)
}

// LockAllKeys locks all keys across all active backends
func LockAllKeys() {
	backendMu.RLock()
	defer backendMu.RUnlock()

	for _, b := range activeBackends {
		keys, err := b.ListKeys(context.Background())
		if err != nil {
			continue
		}
		for _, k := range keys {
			_ = b.Lock(context.Background(), k.Name)
		}
	}
}

// UnlockKey unlocks a key using the default backend
func UnlockKey(name, password string) error {
	backend, err := GetDefaultBackend()
	if err != nil {
		return err
	}
	return backend.Unlock(context.Background(), name, password)
}

// IsKeyLocked checks if a key is locked using the default backend
func IsKeyLocked(name string) bool {
	backend, err := GetDefaultBackend()
	if err != nil {
		return true
	}
	return backend.IsLocked(name)
}

// GetPasswordFromEnv returns the password from the LUX_KEY_PASSWORD environment variable
func GetPasswordFromEnv() string {
	return os.Getenv(EnvKeyPassword)
}
