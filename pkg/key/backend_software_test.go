// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

// Test mnemonic for reproducible tests
const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// deriveKey is a test helper that derives a key using Argon2id for testing purposes
func deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
}

func newTestSoftwareBackend(t *testing.T) *SoftwareBackend {
	t.Helper()
	b := NewSoftwareBackend()
	b.dataDir = t.TempDir()
	return b
}

func TestSoftwareBackend_Properties(t *testing.T) {
	b := NewSoftwareBackend()

	assert.Equal(t, BackendSoftware, b.Type())
	assert.Equal(t, "Encrypted File Storage", b.Name())
	assert.True(t, b.Available())
	assert.True(t, b.RequiresPassword())
	assert.False(t, b.RequiresHardware())
	assert.False(t, b.SupportsRemoteSigning())
}

func TestSoftwareBackend_Initialize(t *testing.T) {
	t.Run("creates data directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		dataDir := filepath.Join(tmpDir, "keys")

		b := NewSoftwareBackend()
		b.dataDir = dataDir

		ctx := context.Background()
		err := b.Initialize(ctx)
		require.NoError(t, err)

		info, err := os.Stat(dataDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	})

	t.Run("uses default directory when not set", func(t *testing.T) {
		// This test would use real home directory, skip in CI
		if os.Getenv("CI") != "" {
			t.Skip("skipping in CI")
		}

		b := NewSoftwareBackend()
		ctx := context.Background()
		err := b.Initialize(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, b.dataDir)
	})
}

func TestSoftwareBackend_CreateKey(t *testing.T) {
	t.Run("creates new key with generated mnemonic", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		keySet, err := b.CreateKey(ctx, "testkey", CreateKeyOptions{
			Password: "testpassword123",
		})
		require.NoError(t, err)
		require.NotNil(t, keySet)

		assert.Equal(t, "testkey", keySet.Name)
		assert.NotEmpty(t, keySet.ECPrivateKey)
		assert.NotEmpty(t, keySet.ECPublicKey)
		assert.NotEmpty(t, keySet.ECAddress)
		assert.NotEmpty(t, keySet.BLSPrivateKey)
		assert.NotEmpty(t, keySet.BLSPublicKey)
		assert.NotEmpty(t, keySet.NodeID)
	})

	t.Run("creates key from provided mnemonic", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		keySet, err := b.CreateKey(ctx, "imported", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "testpassword123",
		})
		require.NoError(t, err)
		require.NotNil(t, keySet)
		assert.Equal(t, "imported", keySet.Name)
	})

	t.Run("fails with invalid mnemonic", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "badkey", CreateKeyOptions{
			Mnemonic: "invalid mnemonic phrase that is not valid",
			Password: "testpassword123",
		})
		require.Error(t, err)
	})

	t.Run("fails if key already exists", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "duplicate", CreateKeyOptions{
			Password: "testpassword123",
		})
		require.NoError(t, err)

		_, err = b.CreateKey(ctx, "duplicate", CreateKeyOptions{
			Password: "testpassword123",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyExists)
	})

	t.Run("fails without password", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "nopass", CreateKeyOptions{
			Password: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoPassword)
	})
}

func TestSoftwareBackend_LoadKey(t *testing.T) {
	t.Run("loads existing key with correct password", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		original, err := b.CreateKey(ctx, "loadtest", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "correctpassword",
		})
		require.NoError(t, err)

		// Clear session to force password-based load
		_ = b.Lock(ctx, "loadtest")

		loaded, err := b.LoadKey(ctx, "loadtest", "correctpassword")
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, original.Name, loaded.Name)
		assert.Equal(t, original.ECAddress, loaded.ECAddress)
		assert.Equal(t, original.ECPrivateKey, loaded.ECPrivateKey)
	})

	t.Run("uses session when available", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "sessiontest", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "testpassword",
		})
		require.NoError(t, err)

		// First load with password creates session
		_, err = b.LoadKey(ctx, "sessiontest", "testpassword")
		require.NoError(t, err)

		// Second load without password should use session
		loaded, err := b.LoadKey(ctx, "sessiontest", "")
		require.NoError(t, err)
		require.NotNil(t, loaded)
	})

	t.Run("fails for non-existent key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.LoadKey(ctx, "nonexistent", "password")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("returns ErrKeyLocked without password or session", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "locked", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Clear session
		_ = b.Lock(ctx, "locked")

		// Clear env var
		_ = os.Unsetenv(EnvKeyPassword)

		_, err = b.LoadKey(ctx, "locked", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyLocked)
	})
}

func TestSoftwareBackend_SaveKey(t *testing.T) {
	t.Run("saves key set successfully", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		keySet, err := DeriveAllKeys("savetest", testMnemonic)
		require.NoError(t, err)

		err = b.SaveKey(ctx, keySet, "password123")
		require.NoError(t, err)

		// Verify files exist
		keyDir := filepath.Join(b.dataDir, "savetest")
		_, err = os.Stat(filepath.Join(keyDir, "keystore.enc"))
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(keyDir, "info.json"))
		require.NoError(t, err)
	})

	t.Run("fails without password", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		keySet, err := DeriveAllKeys("nopass", testMnemonic)
		require.NoError(t, err)

		err = b.SaveKey(ctx, keySet, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoPassword)
	})

	t.Run("overwrites existing key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		keySet1, err := DeriveAllKeys("overwrite", testMnemonic)
		require.NoError(t, err)
		err = b.SaveKey(ctx, keySet1, "password1")
		require.NoError(t, err)

		// Save again with different password
		keySet2, err := DeriveAllKeys("overwrite", testMnemonic)
		require.NoError(t, err)
		err = b.SaveKey(ctx, keySet2, "password2")
		require.NoError(t, err)

		// Should load with new password
		_ = b.Lock(ctx, "overwrite")
		loaded, err := b.LoadKey(ctx, "overwrite", "password2")
		require.NoError(t, err)
		assert.Equal(t, keySet2.ECAddress, loaded.ECAddress)
	})
}

func TestSoftwareBackend_DeleteKey(t *testing.T) {
	t.Run("deletes existing key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "todelete", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		err = b.DeleteKey(ctx, "todelete")
		require.NoError(t, err)

		// Verify directory is removed
		keyDir := filepath.Join(b.dataDir, "todelete")
		_, err = os.Stat(keyDir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("clears session on delete", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "sessiondel", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "sessiondel", "testpassword")
		require.NoError(t, err)
		assert.False(t, b.IsLocked("sessiondel"))

		err = b.DeleteKey(ctx, "sessiondel")
		require.NoError(t, err)

		assert.True(t, b.IsLocked("sessiondel"))
	})

	t.Run("succeeds for non-existent key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		// Should not error
		err := b.DeleteKey(ctx, "nonexistent")
		require.NoError(t, err)
	})
}

func TestSoftwareBackend_ListKeys(t *testing.T) {
	t.Run("lists all keys", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		// Create multiple keys
		for _, name := range []string{"key1", "key2", "key3"} {
			_, err := b.CreateKey(ctx, name, CreateKeyOptions{
				Password: "testpassword",
			})
			require.NoError(t, err)
		}

		keys, err := b.ListKeys(ctx)
		require.NoError(t, err)
		assert.Len(t, keys, 3)

		names := make(map[string]bool)
		for _, k := range keys {
			names[k.Name] = true
			assert.True(t, k.Encrypted)
		}
		assert.True(t, names["key1"])
		assert.True(t, names["key2"])
		assert.True(t, names["key3"])
	})

	t.Run("returns empty list for empty directory", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		keys, err := b.ListKeys(ctx)
		require.NoError(t, err)
		assert.Empty(t, keys)
	})

	t.Run("shows lock status", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "lockstatus", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Initially locked (no session)
		_ = b.Lock(ctx, "lockstatus")
		keys, err := b.ListKeys(ctx)
		require.NoError(t, err)
		assert.True(t, keys[0].Locked)

		// Unlock
		_, err = b.LoadKey(ctx, "lockstatus", "testpassword")
		require.NoError(t, err)

		keys, err = b.ListKeys(ctx)
		require.NoError(t, err)
		assert.False(t, keys[0].Locked)
	})
}

func TestSoftwareBackend_LockUnlock(t *testing.T) {
	t.Run("lock clears session", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "locktest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "locktest", "testpassword")
		require.NoError(t, err)
		assert.False(t, b.IsLocked("locktest"))

		// Lock
		err = b.Lock(ctx, "locktest")
		require.NoError(t, err)
		assert.True(t, b.IsLocked("locktest"))
	})

	t.Run("unlock creates session", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "unlocktest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "unlocktest")

		assert.True(t, b.IsLocked("unlocktest"))

		err = b.Unlock(ctx, "unlocktest", "testpassword")
		require.NoError(t, err)
		assert.False(t, b.IsLocked("unlocktest"))
	})

	t.Run("unlock with wrong password fails", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "wrongpass", CreateKeyOptions{
			Password: "correctpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "wrongpass")

		err = b.Unlock(ctx, "wrongpass", "wrongpassword")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPassword)
		assert.True(t, b.IsLocked("wrongpass"))
	})
}

func TestSoftwareBackend_Sign(t *testing.T) {
	t.Run("signs data with unlocked key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "signtest", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to unlock
		_, err = b.LoadKey(ctx, "signtest", "testpassword")
		require.NoError(t, err)

		data := []byte("test data to sign")
		hash := sha256.Sum256(data)

		resp, err := b.Sign(ctx, "signtest", SignRequest{
			Type:     "message",
			Data:     data,
			DataHash: hash,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.NotEmpty(t, resp.Signature)
		assert.NotEmpty(t, resp.PublicKey)
		assert.NotEmpty(t, resp.Address)
	})

	t.Run("fails when key is locked", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "lockedsign", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "lockedsign")

		_, err = b.Sign(ctx, "lockedsign", SignRequest{
			DataHash: sha256.Sum256([]byte("test")),
		})
		require.Error(t, err)
	})
}

func TestSoftwareBackend_InvalidPassword(t *testing.T) {
	t.Run("wrong password returns ErrInvalidPassword", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "passtest", CreateKeyOptions{
			Password: "correctpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "passtest")

		_, err = b.LoadKey(ctx, "passtest", "wrongpassword")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPassword)
	})

	t.Run("empty password on locked key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "emptypass", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "emptypass")

		// Clear env var
		_ = os.Unsetenv(EnvKeyPassword)

		_, err = b.LoadKey(ctx, "emptypass", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyLocked)
	})
}

func TestSoftwareBackend_SessionExpiry(t *testing.T) {
	t.Run("session expires after timeout", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "expirytest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "expirytest", "testpassword")
		require.NoError(t, err)

		// Manually expire the session
		b.sessionMu.Lock()
		if s, ok := b.sessions["expirytest"]; ok {
			s.expiresAt = time.Now().Add(-1 * time.Hour)
		}
		b.sessionMu.Unlock()

		// Should be locked now
		assert.True(t, b.IsLocked("expirytest"))
	})

	t.Run("session extends on access", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "extendtest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Load to create session
		_, err = b.LoadKey(ctx, "extendtest", "testpassword")
		require.NoError(t, err)

		// Get initial expiry
		b.sessionMu.RLock()
		initialExpiry := b.sessions["extendtest"].expiresAt
		b.sessionMu.RUnlock()

		// Wait a bit then access
		time.Sleep(10 * time.Millisecond)
		_ = b.getSession("extendtest")

		// Expiry should be extended
		b.sessionMu.RLock()
		newExpiry := b.sessions["extendtest"].expiresAt
		b.sessionMu.RUnlock()

		assert.True(t, newExpiry.After(initialExpiry))
	})
}

func TestSoftwareBackend_Close(t *testing.T) {
	t.Run("close zeroes session keys", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "closetest", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		_, err = b.LoadKey(ctx, "closetest", "testpassword")
		require.NoError(t, err)

		// Get reference to session key
		b.sessionMu.RLock()
		keyRef := b.sessions["closetest"].key
		keyLen := len(keyRef)
		b.sessionMu.RUnlock()

		err = b.Close()
		require.NoError(t, err)

		// Key should be zeroed
		allZero := true
		for i := 0; i < keyLen; i++ {
			if keyRef[i] != 0 {
				allZero = false
				break
			}
		}
		assert.True(t, allZero, "session key should be zeroed after close")

		// Sessions map should be empty
		b.sessionMu.RLock()
		assert.Empty(t, b.sessions)
		b.sessionMu.RUnlock()
	})
}

func TestSoftwareBackend_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent loads are safe", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "concurrent", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "testpassword",
		})
		require.NoError(t, err)

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := b.LoadKey(ctx, "concurrent", "testpassword")
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("concurrent load error: %v", err)
		}
	})

	t.Run("concurrent lock/unlock is safe", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "lockrace", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				_ = b.Lock(ctx, "lockrace")
			}()
			go func() {
				defer wg.Done()
				_ = b.Unlock(ctx, "lockrace", "testpassword")
			}()
		}
		wg.Wait()
		// Should not panic
	})
}

func TestEncryptDecrypt(t *testing.T) {
	t.Run("encrypt and decrypt roundtrip", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		plaintext := []byte("secret data to encrypt")

		nonce, ciphertext, err := encryptAESGCM(key, plaintext)
		require.NoError(t, err)
		require.NotEmpty(t, nonce)
		require.NotEmpty(t, ciphertext)
		assert.NotEqual(t, plaintext, ciphertext)

		decrypted, err := decryptAESGCM(key, nonce, ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("different nonce produces different ciphertext", func(t *testing.T) {
		key := make([]byte, 32)
		plaintext := []byte("test data")

		nonce1, ciphertext1, err := encryptAESGCM(key, plaintext)
		require.NoError(t, err)

		nonce2, ciphertext2, err := encryptAESGCM(key, plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, nonce1, nonce2)
		assert.NotEqual(t, ciphertext1, ciphertext2)
	})

	t.Run("wrong key fails decryption", func(t *testing.T) {
		key1 := make([]byte, 32)
		key2 := make([]byte, 32)
		key2[0] = 1

		plaintext := []byte("test data")

		nonce, ciphertext, err := encryptAESGCM(key1, plaintext)
		require.NoError(t, err)

		_, err = decryptAESGCM(key2, nonce, ciphertext)
		require.Error(t, err)
	})

	t.Run("tampered ciphertext fails", func(t *testing.T) {
		key := make([]byte, 32)
		plaintext := []byte("test data")

		nonce, ciphertext, err := encryptAESGCM(key, plaintext)
		require.NoError(t, err)

		// Tamper with ciphertext
		ciphertext[0] ^= 0xFF

		_, err = decryptAESGCM(key, nonce, ciphertext)
		require.Error(t, err)
	})

	t.Run("empty plaintext", func(t *testing.T) {
		key := make([]byte, 32)
		plaintext := []byte{}

		nonce, ciphertext, err := encryptAESGCM(key, plaintext)
		require.NoError(t, err)

		decrypted, err := decryptAESGCM(key, nonce, ciphertext)
		require.NoError(t, err)
		// Empty plaintext decrypts to nil or empty slice - check length
		assert.Len(t, decrypted, 0)
	})
}

func TestArgon2KeyDerivation(t *testing.T) {
	t.Run("derives consistent key", func(t *testing.T) {
		password := []byte("testpassword")
		salt := []byte("testsalt12345678testsalt12345678")

		key1 := deriveKey(password, salt)
		key2 := deriveKey(password, salt)

		assert.Equal(t, key1, key2)
		assert.Len(t, key1, 32)
	})

	t.Run("different password produces different key", func(t *testing.T) {
		salt := []byte("testsalt12345678testsalt12345678")

		key1 := deriveKey([]byte("password1"), salt)
		key2 := deriveKey([]byte("password2"), salt)

		assert.NotEqual(t, key1, key2)
	})

	t.Run("different salt produces different key", func(t *testing.T) {
		password := []byte("testpassword")

		key1 := deriveKey(password, []byte("salt1234567890123456789012345678"))
		key2 := deriveKey(password, []byte("salt8765432109876543210987654321"))

		assert.NotEqual(t, key1, key2)
	})

	t.Run("empty password works", func(t *testing.T) {
		salt := []byte("testsalt12345678testsalt12345678")
		key := deriveKey([]byte{}, salt)
		assert.Len(t, key, 32)
	})

	t.Run("same password via backend roundtrip", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		// Create key with specific password
		original, err := b.CreateKey(ctx, "argon2test", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "consistentpassword",
		})
		require.NoError(t, err)

		// Lock to clear session
		_ = b.Lock(ctx, "argon2test")

		// Load with same password should work (proves consistent derivation)
		loaded, err := b.LoadKey(ctx, "argon2test", "consistentpassword")
		require.NoError(t, err)
		assert.Equal(t, original.ECPrivateKey, loaded.ECPrivateKey)
	})

	t.Run("different passwords via backend", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "diffpass", CreateKeyOptions{
			Password: "password1",
		})
		require.NoError(t, err)

		_ = b.Lock(ctx, "diffpass")

		// Different password should fail (proves different key derivation)
		_, err = b.LoadKey(ctx, "diffpass", "password2")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPassword)
	})
}

func TestSoftwareBackend_GetKeyChecksum(t *testing.T) {
	t.Run("returns checksum for unlocked key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "checksumtest", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "testpassword",
		})
		require.NoError(t, err)

		_, err = b.LoadKey(ctx, "checksumtest", "testpassword")
		require.NoError(t, err)

		checksum, err := b.GetKeyChecksum("checksumtest")
		require.NoError(t, err)
		assert.Len(t, checksum, 16) // 8 bytes = 16 hex chars
	})

	t.Run("fails for locked key", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "lockedchecksum", CreateKeyOptions{
			Password: "testpassword",
		})
		require.NoError(t, err)
		_ = b.Lock(ctx, "lockedchecksum")

		_, err = b.GetKeyChecksum("lockedchecksum")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyLocked)
	})

	t.Run("same key produces same checksum", func(t *testing.T) {
		b := newTestSoftwareBackend(t)
		ctx := context.Background()

		_, err := b.CreateKey(ctx, "samechecksum", CreateKeyOptions{
			Mnemonic: testMnemonic,
			Password: "testpassword",
		})
		require.NoError(t, err)

		_, err = b.LoadKey(ctx, "samechecksum", "testpassword")
		require.NoError(t, err)

		checksum1, err := b.GetKeyChecksum("samechecksum")
		require.NoError(t, err)

		checksum2, err := b.GetKeyChecksum("samechecksum")
		require.NoError(t, err)

		assert.Equal(t, checksum1, checksum2)
	})
}

func TestSerializeParseKeySet(t *testing.T) {
	t.Run("serialize and parse roundtrip", func(t *testing.T) {
		original, err := DeriveAllKeys("roundtrip", testMnemonic)
		require.NoError(t, err)

		serialized, err := serializeKeySet(original)
		require.NoError(t, err)
		require.NotEmpty(t, serialized)

		parsed, err := parseKeySetJSON(serialized)
		require.NoError(t, err)

		assert.Equal(t, original.Name, parsed.Name)
		assert.Equal(t, original.ECPrivateKey, parsed.ECPrivateKey)
		assert.Equal(t, original.ECPublicKey, parsed.ECPublicKey)
		assert.Equal(t, original.ECAddress, parsed.ECAddress)
		assert.Equal(t, original.BLSPrivateKey, parsed.BLSPrivateKey)
		assert.Equal(t, original.BLSPublicKey, parsed.BLSPublicKey)
		assert.Equal(t, original.RingtailPrivateKey, parsed.RingtailPrivateKey)
		assert.Equal(t, original.MLDSAPrivateKey, parsed.MLDSAPrivateKey)
		assert.Equal(t, original.NodeID, parsed.NodeID)
	})
}
