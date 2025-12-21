// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionTimeout(t *testing.T) {
	assert.Equal(t, 15*time.Minute, SessionTimeout)
}

func TestGetPasswordFromEnv(t *testing.T) {
	t.Run("returns empty when not set", func(t *testing.T) {
		os.Unsetenv(EnvKeyPassword)
		assert.Empty(t, GetPasswordFromEnv())
	})

	t.Run("returns value when set", func(t *testing.T) {
		os.Setenv(EnvKeyPassword, "testpassword")
		defer os.Unsetenv(EnvKeyPassword)

		assert.Equal(t, "testpassword", GetPasswordFromEnv())
	})
}

func TestIsKeyLocked(t *testing.T) {
	// Without a backend, keys should be considered locked
	t.Run("returns true when no backend available", func(t *testing.T) {
		// Clear backends for this test
		backendMu.Lock()
		oldBackends := backends
		backends = make(map[BackendType]KeyBackend)
		backendMu.Unlock()

		defer func() {
			backendMu.Lock()
			backends = oldBackends
			backendMu.Unlock()
		}()

		assert.True(t, IsKeyLocked("nonexistent"))
	})
}
