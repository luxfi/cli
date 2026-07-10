// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/light"
)

// Environment variable names for key loading. Brand prefix dropped — these
// are the canonical (unprefixed) names every tool reads.
const (
	// EnvMnemonic contains a BIP39 mnemonic phrase.
	EnvMnemonic = "MNEMONIC"

	// EnvPrivateKey contains a hex-encoded secp256k1 private key.
	EnvPrivateKey = "PRIVATE_KEY"

	// EnvBLSKey contains a hex-encoded BLS private key.
	EnvBLSKey = "BLS_KEY"

	// EnvKeyPassword for encrypted key files.
	EnvKeyPassword = "KEY_PASSWORD"

	// EnvKeySessionTimeout configures the session timeout duration.
	// Format: Go duration string (e.g., "30s", "5m", "1h").
	// Default: 30s (30 seconds of inactivity before auto-lock).
	EnvKeySessionTimeout = "KEY_SESSION_TIMEOUT"

	// EnvKeyIndex selects the BIP-44 address index for mnemonic derivation.
	// Path: m/44'/9000'/0'/0/{index} for P/X-Chain.
	// Default: "auto" — scans indices 0-99 to find the first funded account.
	// Set to a specific number (e.g., "1") to use that index directly.
	// MNEMONIC_ACCOUNT takes priority for backward compatibility with other tools.
	EnvKeyIndex = "KEY_INDEX"

	// EnvLightMnemonic is the well-known dev/local mnemonic for local development.
	// This mnemonic is PUBLIC and safe to commit — it is NOT used for production.
	EnvLightMnemonic = "LIGHT_MNEMONIC"

	// LightMnemonic is the default mnemonic for local development networks.
	// Intentionally public. Single source of truth: github.com/luxfi/light.Mnemonic
	// — this is a backward-compat re-export; do not re-declare the literal here.
	LightMnemonic = light.Mnemonic
)

// getEnv returns the value of the named environment variable.
func getEnv(name string) string {
	return os.Getenv(name)
}

// getKeyIndex returns the configured key index from MNEMONIC_ACCOUNT or KEY_INDEX.
// MNEMONIC_ACCOUNT (industry-standard name) takes priority.
func getKeyIndex() string {
	if v := os.Getenv("MNEMONIC_ACCOUNT"); v != "" {
		return v
	}
	return os.Getenv(EnvKeyIndex)
}

// EnvBackend loads keys from environment variables
// This is useful for CI/CD, containers, and automation
type EnvBackend struct {
	// Cache loaded keys in memory (they're already in env anyway)
	keys map[string]*HDKeySet
}

// NewEnvBackend creates an environment variable backend
func NewEnvBackend() *EnvBackend {
	return &EnvBackend{
		keys: make(map[string]*HDKeySet),
	}
}

func (*EnvBackend) Type() BackendType {
	return BackendEnv
}

func (*EnvBackend) Name() string {
	return "Environment Variables"
}

func (*EnvBackend) Available() bool {
	return getEnv(EnvMnemonic) != "" ||
		getEnv(EnvPrivateKey) != "" ||
		getEnv(EnvBLSKey) != ""
}

func (*EnvBackend) RequiresPassword() bool {
	return false
}

func (*EnvBackend) RequiresHardware() bool {
	return false
}

func (*EnvBackend) SupportsRemoteSigning() bool {
	return false
}

func (*EnvBackend) Initialize(_ context.Context) error {
	return nil
}

func (b *EnvBackend) Close() error {
	// Zero out cached keys
	for name, ks := range b.keys {
		if ks != nil {
			for i := range ks.ECPrivateKey {
				ks.ECPrivateKey[i] = 0
			}
			for i := range ks.BLSPrivateKey {
				ks.BLSPrivateKey[i] = 0
			}
		}
		delete(b.keys, name)
	}
	return nil
}

func (*EnvBackend) CreateKey(_ context.Context, _ string, _ CreateKeyOptions) (*HDKeySet, error) {
	return nil, errors.New("cannot create keys in environment backend - set MNEMONIC or PRIVATE_KEY")
}

func (b *EnvBackend) LoadKey(ctx context.Context, name, password string) (*HDKeySet, error) {
	// Check cache first
	if ks, ok := b.keys[name]; ok {
		return ks, nil
	}

	// Try to load from environment
	ks, err := b.loadFromEnv(name)
	if err != nil {
		return nil, err
	}

	// Cache
	b.keys[name] = ks
	return ks, nil
}

func (*EnvBackend) loadFromEnv(name string) (*HDKeySet, error) {
	// Priority 1: MNEMONIC
	if mnemonic := getEnv(EnvMnemonic); mnemonic != "" {
		if !ValidateMnemonic(mnemonic) {
			return nil, errors.New("invalid mnemonic in MNEMONIC")
		}
		return DeriveAllKeys(name, mnemonic)
	}

	// Priority 2: PRIVATE_KEY (hex-encoded EC key)
	if privKeyHex := getEnv(EnvPrivateKey); privKeyHex != "" {
		privKeyHex = strings.TrimPrefix(privKeyHex, "0x")
		privKeyBytes, err := hex.DecodeString(privKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid hex in PRIVATE_KEY: %w", err)
		}

		privKey, err := secp256k1.ToPrivateKey(privKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("invalid private key in PRIVATE_KEY: %w", err)
		}

		// Create minimal key set with just EC key
		ks := &HDKeySet{
			Name:         name,
			ECPrivateKey: privKeyBytes,
			ECPublicKey:  privKey.PublicKey().Bytes(),
			ECAddress:    deriveECAddress(privKey.PublicKey().Bytes()),
		}

		// Also load BLS key if provided
		if blsHex := getEnv(EnvBLSKey); blsHex != "" {
			blsHex = strings.TrimPrefix(blsHex, "0x")
			blsBytes, err := hex.DecodeString(blsHex)
			if err == nil {
				ks.BLSPrivateKey = blsBytes
				ks.BLSPublicKey, ks.BLSPoP, _ = deriveBLSPublicKey(blsBytes)
			}
		}

		return ks, nil
	}

	return nil, errors.New("no key found in environment (set MNEMONIC or PRIVATE_KEY)")
}

func (*EnvBackend) SaveKey(_ context.Context, _ *HDKeySet, _ string) error {
	return errors.New("cannot save keys to environment backend")
}

func (b *EnvBackend) DeleteKey(ctx context.Context, name string) error {
	if ks, ok := b.keys[name]; ok {
		for i := range ks.ECPrivateKey {
			ks.ECPrivateKey[i] = 0
		}
		delete(b.keys, name)
	}
	return nil
}

func (b *EnvBackend) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	var keys []KeyInfo

	// Check if env vars are set
	if getEnv(EnvMnemonic) != "" || getEnv(EnvPrivateKey) != "" {
		// Load the key to get address info
		ks, err := b.LoadKey(ctx, "env", "")
		if err == nil {
			keys = append(keys, KeyInfo{
				Name:      "env",
				Address:   ks.ECAddress,
				NodeID:    ks.NodeID,
				Encrypted: false,
				Locked:    false,
			})
		}
	}

	return keys, nil
}

func (*EnvBackend) Lock(_ context.Context, _ string) error {
	// Cannot lock env keys - they're always available via env
	return nil
}

func (b *EnvBackend) Unlock(ctx context.Context, name, password string) error {
	// Env keys don't need unlocking
	_, err := b.LoadKey(ctx, name, "")
	return err
}

func (*EnvBackend) IsLocked(_ string) bool {
	return false // Env keys are never locked
}

func (b *EnvBackend) Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error) {
	keySet, err := b.LoadKey(ctx, name, "")
	if err != nil {
		return nil, err
	}

	if len(keySet.ECPrivateKey) == 0 {
		return nil, errors.New("no EC private key available")
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

func (b *EnvBackend) GetKeyChecksum(name string) (string, error) {
	ks, err := b.LoadKey(context.Background(), name, "")
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(ks.ECPrivateKey)
	h.Write(ks.BLSPrivateKey)
	return hex.EncodeToString(h.Sum(nil)[:8]), nil
}

// GetLightMnemonic returns the light mnemonic from LIGHT_MNEMONIC env var,
// or the default "light light light...energy" if not set.
// This is the well-known dev mnemonic for local networks.
func GetLightMnemonic() string {
	if v := os.Getenv(EnvLightMnemonic); v != "" {
		return v
	}
	return LightMnemonic
}

func init() {
	RegisterBackend(NewEnvBackend())
}
