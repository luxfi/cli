// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBackend implements KeyBackend for testing
type mockBackend struct {
	backendType BackendType
	name        string
	available   bool
}

func (m *mockBackend) Type() BackendType                    { return m.backendType }
func (m *mockBackend) Name() string                         { return m.name }
func (m *mockBackend) Available() bool                      { return m.available }
func (m *mockBackend) RequiresPassword() bool               { return true }
func (m *mockBackend) RequiresHardware() bool               { return false }
func (m *mockBackend) SupportsRemoteSigning() bool          { return false }
func (m *mockBackend) Initialize(ctx context.Context) error { return nil }
func (m *mockBackend) Close() error                         { return nil }
func (m *mockBackend) CreateKey(ctx context.Context, name string, opts CreateKeyOptions) (*HDKeySet, error) {
	return nil, nil
}
func (m *mockBackend) LoadKey(ctx context.Context, name, password string) (*HDKeySet, error) {
	return nil, nil
}
func (m *mockBackend) SaveKey(ctx context.Context, keySet *HDKeySet, password string) error {
	return nil
}
func (m *mockBackend) DeleteKey(ctx context.Context, name string) error { return nil }
func (m *mockBackend) ListKeys(ctx context.Context) ([]KeyInfo, error)  { return nil, nil }
func (m *mockBackend) Lock(ctx context.Context, name string) error      { return nil }
func (m *mockBackend) Unlock(ctx context.Context, name, password string) error {
	return nil
}
func (m *mockBackend) IsLocked(name string) bool { return true }
func (m *mockBackend) Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error) {
	return nil, nil
}

func resetBackendRegistry() {
	backendMu.Lock()
	defer backendMu.Unlock()
	backends = make(map[BackendType]KeyBackend)
	defaultBackend = ""
	activeBackends = make(map[BackendType]KeyBackend)
}

func TestBackendRegistry(t *testing.T) {
	t.Run("register and get backend", func(t *testing.T) {
		resetBackendRegistry()

		mock := &mockBackend{
			backendType: BackendType("test"),
			name:        "Test Backend",
			available:   true,
		}

		RegisterBackend(mock)

		got, err := GetBackend(BackendType("test"))
		require.NoError(t, err)
		assert.Equal(t, mock, got)
		assert.Equal(t, BackendType("test"), got.Type())
		assert.Equal(t, "Test Backend", got.Name())
	})

	t.Run("get non-existent backend", func(t *testing.T) {
		resetBackendRegistry()

		_, err := GetBackend(BackendType("nonexistent"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBackendNotFound)
	})

	t.Run("get unavailable backend", func(t *testing.T) {
		resetBackendRegistry()

		mock := &mockBackend{
			backendType: BackendType("unavailable"),
			name:        "Unavailable Backend",
			available:   false,
		}

		RegisterBackend(mock)

		_, err := GetBackend(BackendType("unavailable"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBackendNotSupported)
	})

	t.Run("register overwrites existing", func(t *testing.T) {
		resetBackendRegistry()

		mock1 := &mockBackend{
			backendType: BackendType("test"),
			name:        "First Backend",
			available:   true,
		}
		mock2 := &mockBackend{
			backendType: BackendType("test"),
			name:        "Second Backend",
			available:   true,
		}

		RegisterBackend(mock1)
		RegisterBackend(mock2)

		got, err := GetBackend(BackendType("test"))
		require.NoError(t, err)
		assert.Equal(t, "Second Backend", got.Name())
	})

	t.Run("concurrent registration is safe", func(t *testing.T) {
		resetBackendRegistry()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				mock := &mockBackend{
					backendType: BackendType("concurrent"),
					name:        "Concurrent Backend",
					available:   true,
				}
				RegisterBackend(mock)
			}(i)
		}
		wg.Wait()

		// Should have exactly one registration (last one wins)
		got, err := GetBackend(BackendType("concurrent"))
		require.NoError(t, err)
		assert.NotNil(t, got)
	})
}

func TestListAvailableBackends(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		resetBackendRegistry()

		available := ListAvailableBackends()
		assert.Empty(t, available)
	})

	t.Run("mixed availability", func(t *testing.T) {
		resetBackendRegistry()

		available1 := &mockBackend{
			backendType: BackendType("available1"),
			name:        "Available 1",
			available:   true,
		}
		available2 := &mockBackend{
			backendType: BackendType("available2"),
			name:        "Available 2",
			available:   true,
		}
		unavailable := &mockBackend{
			backendType: BackendType("unavailable"),
			name:        "Unavailable",
			available:   false,
		}

		RegisterBackend(available1)
		RegisterBackend(available2)
		RegisterBackend(unavailable)

		list := ListAvailableBackends()
		assert.Len(t, list, 2)

		// Check both available backends are in list
		names := make(map[string]bool)
		for _, b := range list {
			names[b.Name()] = true
		}
		assert.True(t, names["Available 1"])
		assert.True(t, names["Available 2"])
		assert.False(t, names["Unavailable"])
	})

	t.Run("all unavailable", func(t *testing.T) {
		resetBackendRegistry()

		for i := 0; i < 3; i++ {
			RegisterBackend(&mockBackend{
				backendType: BackendType("unavailable"),
				available:   false,
			})
		}

		available := ListAvailableBackends()
		assert.Empty(t, available)
	})
}

func TestGetDefaultBackend(t *testing.T) {
	t.Run("no backends registered", func(t *testing.T) {
		resetBackendRegistry()

		_, err := GetDefaultBackend()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBackendNotFound)
	})

	t.Run("software backend as fallback", func(t *testing.T) {
		resetBackendRegistry()

		software := &mockBackend{
			backendType: BackendSoftware,
			name:        "Software Backend",
			available:   true,
		}
		RegisterBackend(software)

		got, err := GetDefaultBackend()
		require.NoError(t, err)
		assert.Equal(t, BackendSoftware, got.Type())
	})

	t.Run("explicit default overrides platform", func(t *testing.T) {
		resetBackendRegistry()

		software := &mockBackend{
			backendType: BackendSoftware,
			name:        "Software Backend",
			available:   true,
		}
		keychain := &mockBackend{
			backendType: BackendKeychain,
			name:        "Keychain Backend",
			available:   true,
		}
		yubikey := &mockBackend{
			backendType: BackendYubikey,
			name:        "Yubikey Backend",
			available:   true,
		}

		RegisterBackend(software)
		RegisterBackend(keychain)
		RegisterBackend(yubikey)

		// Set explicit default
		err := SetDefaultBackend(BackendYubikey)
		require.NoError(t, err)

		got, err := GetDefaultBackend()
		require.NoError(t, err)
		assert.Equal(t, BackendYubikey, got.Type())
	})

	t.Run("set default for non-existent backend fails", func(t *testing.T) {
		resetBackendRegistry()

		err := SetDefaultBackend(BackendType("nonexistent"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBackendNotFound)
	})

	t.Run("darwin prefers keychain", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("darwin-specific test")
		}

		resetBackendRegistry()

		software := &mockBackend{
			backendType: BackendSoftware,
			name:        "Software Backend",
			available:   true,
		}
		keychain := &mockBackend{
			backendType: BackendKeychain,
			name:        "Keychain Backend",
			available:   true,
		}

		RegisterBackend(software)
		RegisterBackend(keychain)

		got, err := GetDefaultBackend()
		require.NoError(t, err)
		assert.Equal(t, BackendKeychain, got.Type())
	})

	t.Run("darwin falls back to software when keychain unavailable", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("darwin-specific test")
		}

		resetBackendRegistry()

		software := &mockBackend{
			backendType: BackendSoftware,
			name:        "Software Backend",
			available:   true,
		}
		keychain := &mockBackend{
			backendType: BackendKeychain,
			name:        "Keychain Backend",
			available:   false, // unavailable
		}

		RegisterBackend(software)
		RegisterBackend(keychain)

		got, err := GetDefaultBackend()
		require.NoError(t, err)
		assert.Equal(t, BackendSoftware, got.Type())
	})

	t.Run("linux prefers secret service", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("linux-specific test")
		}

		resetBackendRegistry()

		software := &mockBackend{
			backendType: BackendSoftware,
			name:        "Software Backend",
			available:   true,
		}
		secretService := &mockBackend{
			backendType: BackendSecretService,
			name:        "Secret Service Backend",
			available:   true,
		}

		RegisterBackend(software)
		RegisterBackend(secretService)

		got, err := GetDefaultBackend()
		require.NoError(t, err)
		assert.Equal(t, BackendSecretService, got.Type())
	})
}

func TestInitializeBackends(t *testing.T) {
	t.Run("initializes available backends", func(t *testing.T) {
		resetBackendRegistry()

		mock := &mockBackend{
			backendType: BackendSoftware,
			name:        "Software",
			available:   true,
		}
		RegisterBackend(mock)

		ctx := context.Background()
		err := InitializeBackends(ctx, BackendConfig{})
		require.NoError(t, err)

		// Check activeBackends was populated
		backendMu.RLock()
		_, exists := activeBackends[BackendSoftware]
		backendMu.RUnlock()
		assert.True(t, exists)
	})

	t.Run("skips unavailable backends", func(t *testing.T) {
		resetBackendRegistry()

		mock := &mockBackend{
			backendType: BackendYubikey,
			name:        "Yubikey",
			available:   false,
		}
		RegisterBackend(mock)

		ctx := context.Background()
		err := InitializeBackends(ctx, BackendConfig{})
		require.NoError(t, err)

		backendMu.RLock()
		_, exists := activeBackends[BackendYubikey]
		backendMu.RUnlock()
		assert.False(t, exists)
	})
}

func TestCloseBackends(t *testing.T) {
	t.Run("clears active backends", func(t *testing.T) {
		resetBackendRegistry()

		mock := &mockBackend{
			backendType: BackendSoftware,
			name:        "Software",
			available:   true,
		}
		RegisterBackend(mock)

		ctx := context.Background()
		_ = InitializeBackends(ctx, BackendConfig{})

		CloseBackends()

		backendMu.RLock()
		count := len(activeBackends)
		backendMu.RUnlock()
		assert.Equal(t, 0, count)
	})
}

func TestBackendTypes(t *testing.T) {
	// Verify backend type constants are distinct
	types := []BackendType{
		BackendSoftware,
		BackendKeychain,
		BackendSecretService,
		BackendYubikey,
		BackendZymbit,
		BackendWalletConnect,
		BackendLedger,
		BackendEnv,
	}

	seen := make(map[BackendType]bool)
	for _, bt := range types {
		assert.False(t, seen[bt], "duplicate backend type: %s", bt)
		seen[bt] = true
		assert.NotEmpty(t, string(bt))
	}
}

func TestSignRequest(t *testing.T) {
	t.Run("sign request fields", func(t *testing.T) {
		req := SignRequest{
			Type:        "transaction",
			ChainID:     1,
			Description: "Test transaction",
			Data:        []byte("test data"),
			DataHash:    [32]byte{1, 2, 3},
		}

		assert.Equal(t, "transaction", req.Type)
		assert.Equal(t, uint64(1), req.ChainID)
		assert.Equal(t, "Test transaction", req.Description)
		assert.Equal(t, []byte("test data"), req.Data)
		assert.Equal(t, byte(1), req.DataHash[0])
	})
}

func TestSignResponse(t *testing.T) {
	t.Run("sign response fields", func(t *testing.T) {
		resp := SignResponse{
			Signature: []byte{1, 2, 3, 4},
			PublicKey: []byte{5, 6, 7, 8},
			Address:   "0x1234567890abcdef",
		}

		assert.Equal(t, []byte{1, 2, 3, 4}, resp.Signature)
		assert.Equal(t, []byte{5, 6, 7, 8}, resp.PublicKey)
		assert.Equal(t, "0x1234567890abcdef", resp.Address)
	})
}

func TestCreateKeyOptions(t *testing.T) {
	t.Run("create key options fields", func(t *testing.T) {
		opts := CreateKeyOptions{
			Mnemonic:      "test mnemonic phrase",
			Password:      "testpassword",
			UseBiometrics: true,
			YubikeySlot:   9,
			ImportOnly:    true,
		}

		assert.Equal(t, "test mnemonic phrase", opts.Mnemonic)
		assert.Equal(t, "testpassword", opts.Password)
		assert.True(t, opts.UseBiometrics)
		assert.Equal(t, 9, opts.YubikeySlot)
		assert.True(t, opts.ImportOnly)
	})
}

func TestKeyInfo(t *testing.T) {
	t.Run("key info fields", func(t *testing.T) {
		info := KeyInfo{
			Name:      "testkey",
			Address:   "0x123",
			NodeID:    "NodeID-abc",
			Encrypted: true,
			Locked:    false,
		}

		assert.Equal(t, "testkey", info.Name)
		assert.Equal(t, "0x123", info.Address)
		assert.Equal(t, "NodeID-abc", info.NodeID)
		assert.True(t, info.Encrypted)
		assert.False(t, info.Locked)
	})
}

func TestBackendConfig(t *testing.T) {
	t.Run("backend config fields", func(t *testing.T) {
		cfg := BackendConfig{
			DataDir:                "/tmp/keys",
			WalletConnectProjectID: "project123",
			ZymbitDevicePath:       "/dev/zymbit",
			YubikeyPIN:             "123456",
		}

		assert.Equal(t, "/tmp/keys", cfg.DataDir)
		assert.Equal(t, "project123", cfg.WalletConnectProjectID)
		assert.Equal(t, "/dev/zymbit", cfg.ZymbitDevicePath)
		assert.Equal(t, "123456", cfg.YubikeyPIN)
	})
}
