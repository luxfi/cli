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

// Test mnemonic for reproducible tests
const sessionTestMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestSessionCache30Seconds(t *testing.T) {
	t.Run("default timeout is 30 seconds", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, DefaultSessionTimeout)
	})

	t.Run("session timeout resets on access", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "slidetest", CreateKeyOptions{
			Mnemonic: sessionTestMnemonic,
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "slidetest", "testpassword")
		require.NoError(t, err)

		// Get initial expiry
		b.sessionMu.Lock()
		initialExpiry := b.sessions["slidetest"].expiresAt
		b.sessionMu.Unlock()

		// Wait a bit
		time.Sleep(50 * time.Millisecond)

		// Access the session (should extend)
		session := b.getSession("slidetest")
		require.NotNil(t, session)

		// Get new expiry
		b.sessionMu.Lock()
		newExpiry := b.sessions["slidetest"].expiresAt
		b.sessionMu.Unlock()

		// Should have been extended
		assert.True(t, newExpiry.After(initialExpiry), "session expiry should extend on access")
	})

	t.Run("session expires after inactivity", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		// Set very short timeout for testing
		b.sessionTimeout = 100 * time.Millisecond
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "expiretest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "expiretest", "testpassword")
		require.NoError(t, err)
		assert.False(t, b.IsLocked("expiretest"))

		// Wait for session to expire
		time.Sleep(150 * time.Millisecond)

		// Should be locked now
		assert.True(t, b.IsLocked("expiretest"))
	})

	t.Run("expired session is cleared securely", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		// Set very short timeout for testing
		b.sessionTimeout = 50 * time.Millisecond
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "secureclear", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "secureclear", "testpassword")
		require.NoError(t, err)

		// Get reference to session key
		b.sessionMu.Lock()
		keyRef := b.sessions["secureclear"].key
		keyLen := len(keyRef)
		b.sessionMu.Unlock()

		// Wait for session to expire
		time.Sleep(100 * time.Millisecond)

		// Access to trigger cleanup
		_ = b.getSession("secureclear")

		// Key should be zeroed
		allZero := true
		for i := 0; i < keyLen; i++ {
			if keyRef[i] != 0 {
				allZero = false
				break
			}
		}
		assert.True(t, allZero, "session key should be zeroed after expiry")
	})
}

func TestSessionCacheMemoryOnly(t *testing.T) {
	t.Run("session is memory only, not persisted", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "memonly", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "memonly", "testpassword")
		require.NoError(t, err)
		assert.False(t, b.IsLocked("memonly"))

		// Create new backend instance pointing to same directory
		b2 := NewSoftwareBackend()
		b2.dataDir = b.dataDir

		// New instance should not have the session
		assert.True(t, b2.IsLocked("memonly"))
	})
}

func TestSessionCacheConfigurable(t *testing.T) {
	t.Run("timeout configurable via SetSessionTimeout", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()

		// Change timeout
		b.SetSessionTimeout(5 * time.Minute)

		b.sessionMu.Lock()
		assert.Equal(t, 5*time.Minute, b.sessionTimeout)
		b.sessionMu.Unlock()
	})

	t.Run("timeout configurable via environment", func(t *testing.T) {
		originalValue := os.Getenv(EnvKeySessionTimeout)
		defer func() {
			if originalValue != "" {
				_ = os.Setenv(EnvKeySessionTimeout, originalValue)
			} else {
				_ = os.Unsetenv(EnvKeySessionTimeout)
			}
		}()

		_ = os.Setenv(EnvKeySessionTimeout, "2m")

		timeout := GetSessionTimeout()
		assert.Equal(t, 2*time.Minute, timeout)
	})

	t.Run("invalid env falls back to default", func(t *testing.T) {
		originalValue := os.Getenv(EnvKeySessionTimeout)
		defer func() {
			if originalValue != "" {
				_ = os.Setenv(EnvKeySessionTimeout, originalValue)
			} else {
				_ = os.Unsetenv(EnvKeySessionTimeout)
			}
		}()

		_ = os.Setenv(EnvKeySessionTimeout, "not-a-duration")

		timeout := GetSessionTimeout()
		assert.Equal(t, DefaultSessionTimeout, timeout)
	})
}

func TestMlockSupport(t *testing.T) {
	t.Run("mlock function exists", func(t *testing.T) {
		// Just verify the function can be called without panic
		err := mlock([]byte("test"))
		// May succeed or fail depending on platform/permissions
		_ = err

		err = munlock([]byte("test"))
		_ = err
	})

	t.Run("mlockSupported returns boolean", func(t *testing.T) {
		supported := mlockSupported()
		// On Unix systems this should be true, on others false
		assert.IsType(t, true, supported)
	})

	t.Run("session tracks mlock status", func(t *testing.T) {
		b := NewSoftwareBackend()
		b.dataDir = t.TempDir()
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "mlocktest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "mlocktest", "testpassword")
		require.NoError(t, err)

		// Check that session has mlocked field
		b.sessionMu.Lock()
		session := b.sessions["mlocktest"]
		b.sessionMu.Unlock()

		// On Unix systems with sufficient permissions, should be mlocked
		// Otherwise false - we just verify the field exists
		assert.IsType(t, true, session.mlocked)
	})
}

func TestClearSessionFunction(t *testing.T) {
	t.Run("clearSession zeros key bytes", func(t *testing.T) {
		key := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		s := &keySession{
			name:    "test",
			key:     key,
			mlocked: false,
		}

		clearSession(s)

		// All bytes should be zero
		for i, b := range key {
			assert.Equal(t, byte(0), b, "byte %d should be zero", i)
		}
	})

	t.Run("clearSession handles nil", func(t *testing.T) {
		// Should not panic
		clearSession(nil)
	})
}
