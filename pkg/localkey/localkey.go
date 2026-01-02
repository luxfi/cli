// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package localkey provides functions to load keys from ~/.lux/keys at runtime.
// Keys are NEVER embedded in code - they must exist on disk or be generated.
package localkey

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/luxfi/crypto/secp256k1"
)

var (
	// Cached keys to avoid re-reading from disk on every call
	keyCache     *secp256k1.PrivateKey
	keyCacheMu   sync.RWMutex
	keyCacheInit bool

	ErrNoKeysFound = errors.New("no keys found in ~/.lux/keys - please generate keys first")
)

// GetLocalKey returns the first key from ~/.lux/keys for local network operations.
// Keys are loaded from disk at runtime, never embedded in code.
// The key is cached after first load.
func GetLocalKey() (*secp256k1.PrivateKey, error) {
	keyCacheMu.RLock()
	if keyCacheInit && keyCache != nil {
		keyCacheMu.RUnlock()
		return keyCache, nil
	}
	keyCacheMu.RUnlock()

	keyCacheMu.Lock()
	defer keyCacheMu.Unlock()

	// Double-check after acquiring write lock
	if keyCacheInit && keyCache != nil {
		return keyCache, nil
	}

	key, err := loadFirstKey()
	if err != nil {
		return nil, err
	}

	keyCache = key
	keyCacheInit = true
	return key, nil
}

// loadFirstKey loads the first key from ~/.lux/keys directory
func loadFirstKey() (*secp256k1.PrivateKey, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	keysDir := filepath.Join(homeDir, ".lux", "keys")

	// Look for validator_XXX.pk files
	entries, err := os.ReadDir(keysDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: directory %s does not exist", ErrNoKeysFound, keysDir)
		}
		return nil, fmt.Errorf("failed to read keys directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".pk" {
			continue
		}

		keyPath := filepath.Join(keysDir, name)
		keyHex, err := os.ReadFile(keyPath) //nolint:gosec // G304: Reading from keys directory
		if err != nil {
			continue // Try next key
		}

		// Trim whitespace/newlines
		keyHexStr := string(keyHex)
		keyHexStr = keyHexStr[:len(keyHexStr)-1] // Remove trailing newline if present
		if len(keyHexStr) > 64 {
			keyHexStr = keyHexStr[:64]
		}

		keyBytes, err := hex.DecodeString(keyHexStr)
		if err != nil {
			continue // Try next key
		}

		privKey, err := secp256k1.ToPrivateKey(keyBytes)
		if err != nil {
			continue // Try next key
		}

		return privKey, nil
	}

	return nil, ErrNoKeysFound
}

// MustGetLocalKey returns the local key or panics if not found.
// Use this only in contexts where key availability is guaranteed.
func MustGetLocalKey() *secp256k1.PrivateKey {
	key, err := GetLocalKey()
	if err != nil {
		panic(fmt.Sprintf("failed to get local key: %v", err))
	}
	return key
}

// ClearCache clears the cached key, forcing a reload on next GetLocalKey call.
func ClearCache() {
	keyCacheMu.Lock()
	defer keyCacheMu.Unlock()
	keyCache = nil
	keyCacheInit = false
}
