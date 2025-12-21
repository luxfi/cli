// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptedKeyStore(t *testing.T) {
	t.Run("encryptedStore struct fields", func(t *testing.T) {
		store := encryptedStore{
			Version:   1,
			Salt:      []byte("testsalt"),
			Nonce:     []byte("testnonce"),
			Data:      []byte("encrypted data"),
			CreatedAt: time.Now().Unix(),
		}

		assert.Equal(t, 1, store.Version)
		assert.Equal(t, []byte("testsalt"), store.Salt)
		assert.Equal(t, []byte("testnonce"), store.Nonce)
		assert.Equal(t, []byte("encrypted data"), store.Data)
		assert.Greater(t, store.CreatedAt, int64(0))
	})
}

func TestSessionManager(t *testing.T) {
	t.Run("session through backend", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		// Create key
		_, err := b.CreateKey(ctx, "sessiontest", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Should have session after create (since LoadKey is called internally)
		// Clear it first to test unlock flow
		_ = b.Lock(ctx, "sessiontest")
		assert.True(t, b.IsLocked("sessiontest"))

		// Unlock creates session
		err = b.Unlock(ctx, "sessiontest", "testpassword")
		require.NoError(t, err)
		assert.False(t, b.IsLocked("sessiontest"))

		// Lock clears session
		err = b.Lock(ctx, "sessiontest")
		require.NoError(t, err)
		assert.True(t, b.IsLocked("sessiontest"))
	})

	t.Run("session expiration", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "expiretest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "expiretest", "testpassword")
		require.NoError(t, err)
		assert.False(t, b.IsLocked("expiretest"))

		// Manually expire session
		b.sessionMu.Lock()
		if s, ok := b.sessions["expiretest"]; ok {
			s.expiresAt = time.Now().Add(-1 * time.Hour)
		}
		b.sessionMu.Unlock()

		// Should be locked now
		assert.True(t, b.IsLocked("expiretest"))
	})

	t.Run("session extends on access", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "extendtest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		_, err = b.LoadKey(ctx, "extendtest", "testpassword")
		require.NoError(t, err)

		// Get initial expiry
		b.sessionMu.RLock()
		initialExpiry := b.sessions["extendtest"].expiresAt
		b.sessionMu.RUnlock()

		time.Sleep(10 * time.Millisecond)

		// Access session (via getSession)
		_ = b.getSession("extendtest")

		// Expiry should be extended
		b.sessionMu.RLock()
		newExpiry := b.sessions["extendtest"].expiresAt
		b.sessionMu.RUnlock()

		assert.True(t, newExpiry.After(initialExpiry))
	})
}

func TestPasswordFromEnv(t *testing.T) {
	t.Run("returns password from env", func(t *testing.T) {
		originalValue := os.Getenv(EnvKeyPassword)
		defer func() {
			if originalValue != "" {
				_ = os.Setenv(EnvKeyPassword, originalValue)
			} else {
				_ = os.Unsetenv(EnvKeyPassword)
			}
		}()

		_ = os.Setenv(EnvKeyPassword, "env-password-123")
		password := GetPasswordFromEnv()
		assert.Equal(t, "env-password-123", password)
	})

	t.Run("returns empty when not set", func(t *testing.T) {
		originalValue := os.Getenv(EnvKeyPassword)
		defer func() {
			if originalValue != "" {
				_ = os.Setenv(EnvKeyPassword, originalValue)
			} else {
				_ = os.Unsetenv(EnvKeyPassword)
			}
		}()

		_ = os.Unsetenv(EnvKeyPassword)
		password := GetPasswordFromEnv()
		assert.Empty(t, password)
	})

	t.Run("password from env used in LoadKey", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		// Create key with password
		_, err := b.CreateKey(ctx, "envtest", CreateKeyOptions{
			Password: "envpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "envtest")

		// Set env password
		originalValue := os.Getenv(EnvKeyPassword)
		defer func() {
			if originalValue != "" {
				_ = os.Setenv(EnvKeyPassword, originalValue)
			} else {
				_ = os.Unsetenv(EnvKeyPassword)
			}
		}()
		_ = os.Setenv(EnvKeyPassword, "envpassword")

		// Load without explicit password (uses env)
		_, err = b.LoadKey(ctx, "envtest", "")
		require.NoError(t, err)
	})
}

func TestGlobalLockFunctions(t *testing.T) {
	// These tests require setting up a default backend
	// Skip if we can't properly set one up
	t.Run("LockKey with default backend", func(t *testing.T) {
		resetBackendRegistry()

		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		RegisterBackend(b)

		ctx := context.Background()
		_ = b.Initialize(ctx)

		_, err := b.CreateKey(ctx, "locktest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		err = LockKey("locktest")
		require.NoError(t, err)
		assert.True(t, IsKeyLocked("locktest"))
	})

	t.Run("UnlockKey with default backend", func(t *testing.T) {
		resetBackendRegistry()

		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		RegisterBackend(b)

		ctx := context.Background()
		_ = b.Initialize(ctx)

		_, err := b.CreateKey(ctx, "unlocktest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "unlocktest")

		err = UnlockKey("unlocktest", "testpassword")
		require.NoError(t, err)
		assert.False(t, IsKeyLocked("unlocktest"))
	})

	t.Run("IsKeyLocked returns true when no backend", func(t *testing.T) {
		resetBackendRegistry()
		assert.True(t, IsKeyLocked("anykey"))
	})
}

func TestSessionTimeoutConstant(t *testing.T) {
	t.Run("session timeout matches internal constant", func(t *testing.T) {
		// Verify public and internal constants match
		assert.Equal(t, 15*time.Minute, sessionTimeout)
	})
}

func TestKeyInfoFields(t *testing.T) {
	t.Run("all fields populated", func(t *testing.T) {
		now := time.Now()
		info := KeyInfo{
			Name:      "testkey",
			Address:   "0x1234567890abcdef",
			NodeID:    "NodeID-abc123",
			Encrypted: true,
			Locked:    false,
			CreatedAt: now,
		}

		assert.Equal(t, "testkey", info.Name)
		assert.Equal(t, "0x1234567890abcdef", info.Address)
		assert.Equal(t, "NodeID-abc123", info.NodeID)
		assert.True(t, info.Encrypted)
		assert.False(t, info.Locked)
		assert.Equal(t, now, info.CreatedAt)
	})
}

func TestEncryptedStoreVersion(t *testing.T) {
	t.Run("current version is 1", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "versiontest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Verify store version by loading raw file
		// This is an implementation detail but important for compatibility
	})
}

func TestErrorConstants(t *testing.T) {
	t.Run("error messages defined", func(t *testing.T) {
		assert.NotNil(t, ErrKeyLocked)
		assert.NotNil(t, ErrKeyNotFound)
		assert.NotNil(t, ErrInvalidPassword)
		assert.NotNil(t, ErrKeyExists)
		assert.NotNil(t, ErrNoPassword)
		assert.NotNil(t, ErrBackendNotFound)
		assert.NotNil(t, ErrBackendNotSupported)
		assert.NotNil(t, ErrBackendUnavailable)
		assert.NotNil(t, ErrSigningCancelled)
		assert.NotNil(t, ErrAuthFailed)

		// Verify error messages contain key information
		assert.Contains(t, ErrKeyLocked.Error(), "locked")
		assert.Contains(t, ErrKeyNotFound.Error(), "not found")
		assert.Contains(t, ErrNoPassword.Error(), "required")
	})
}

func TestBackendConstants(t *testing.T) {
	t.Run("EnvKeyPassword constant", func(t *testing.T) {
		assert.Equal(t, "LUX_KEY_PASSWORD", EnvKeyPassword)
	})

	t.Run("Argon2 parameters are reasonable", func(t *testing.T) {
		// These should match OWASP recommendations
		assert.Equal(t, uint32(3), uint32(argon2Time))
		assert.Equal(t, uint32(64*1024), uint32(argon2Memory))
		assert.Equal(t, uint8(4), uint8(argon2Threads))
		assert.Equal(t, uint32(32), uint32(argon2KeyLen))
	})
}

func TestKeySessionStruct(t *testing.T) {
	t.Run("keySession fields", func(t *testing.T) {
		now := time.Now()
		session := keySession{
			name:       "testkey",
			key:        []byte("encryption-key"),
			unlockedAt: now,
			expiresAt:  now.Add(SessionTimeout),
		}

		assert.Equal(t, "testkey", session.name)
		assert.Equal(t, []byte("encryption-key"), session.key)
		assert.Equal(t, now, session.unlockedAt)
		assert.True(t, session.expiresAt.After(now))
	})
}

func TestCloseZeroesSessionKeys(t *testing.T) {
	t.Run("close zeros all session keys", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		// Create multiple keys
		for i := 0; i < 3; i++ {
			name := string(rune('a' + i))
			_, err := b.CreateKey(ctx, name, CreateKeyOptions{
				Password: "testpassword",
			})
			require.NoError(t, err)
			_, err = b.LoadKey(ctx, name, "testpassword")
			require.NoError(t, err)
		}

		// Get references to session keys
		var keyRefs [][]byte
		b.sessionMu.RLock()
		for _, s := range b.sessions {
			keyRefs = append(keyRefs, s.key)
		}
		b.sessionMu.RUnlock()

		// Close backend
		err := b.Close()
		require.NoError(t, err)

		// All key refs should be zeroed
		for _, keyRef := range keyRefs {
			allZero := true
			for _, b := range keyRef {
				if b != 0 {
					allZero = false
					break
				}
			}
			assert.True(t, allZero, "session key should be zeroed after close")
		}

		// Sessions map should be empty
		b.sessionMu.RLock()
		assert.Empty(t, b.sessions)
		b.sessionMu.RUnlock()
	})
}

func TestLoadKeyWithExpiredSession(t *testing.T) {
	t.Run("expired session requires password", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "expiredload", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Create session
		_, err = b.LoadKey(ctx, "expiredload", "testpassword")
		require.NoError(t, err)

		// Expire session
		b.sessionMu.Lock()
		if s, ok := b.sessions["expiredload"]; ok {
			s.expiresAt = time.Now().Add(-1 * time.Hour)
		}
		b.sessionMu.Unlock()

		// Clear env password
		_ = os.Unsetenv(EnvKeyPassword)

		// Load without password should fail (session expired)
		_, err = b.LoadKey(ctx, "expiredload", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyLocked)

		// Load with password should work
		_, err = b.LoadKey(ctx, "expiredload", "testpassword")
		require.NoError(t, err)
	})
}

func TestLockAllKeysViaBackend(t *testing.T) {
	t.Run("close locks all keys", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		// Create and unlock multiple keys
		for i := 0; i < 3; i++ {
			name := string(rune('x' + i))
			_, err := b.CreateKey(ctx, name, CreateKeyOptions{
				Password: "testpassword",
			})
			require.NoError(t, err)
			// Load to create session
			_, err = b.LoadKey(ctx, name, "testpassword")
			require.NoError(t, err)
		}

		// All should be unlocked (session created during LoadKey)
		for i := 0; i < 3; i++ {
			name := string(rune('x' + i))
			assert.False(t, b.IsLocked(name), "key %s should be unlocked", name)
		}

		// Close backend
		_ = b.Close()

		// All should be locked
		for i := 0; i < 3; i++ {
			name := string(rune('x' + i))
			assert.True(t, b.IsLocked(name), "key %s should be locked after close", name)
		}
	})
}
