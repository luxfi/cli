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
)

// Environment variable names for key loading
const (
	// EnvMnemonic contains a BIP39 mnemonic phrase
	EnvMnemonic = "LUX_MNEMONIC"

	// EnvPrivateKey contains a hex-encoded secp256k1 private key
	EnvPrivateKey = "LUX_PRIVATE_KEY"

	// EnvBLSKey contains a hex-encoded BLS private key
	EnvBLSKey = "LUX_BLS_KEY"

	// EnvKeyPassword for encrypted key files
	EnvKeyPassword = "LUX_KEY_PASSWORD"
)

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
	// Available if any key env vars are set
	return os.Getenv(EnvMnemonic) != "" ||
		os.Getenv(EnvPrivateKey) != "" ||
		os.Getenv(EnvBLSKey) != ""
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
	return nil, errors.New("cannot create keys in environment backend - set LUX_MNEMONIC or LUX_PRIVATE_KEY")
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
	// Priority 1: LUX_MNEMONIC
	if mnemonic := os.Getenv(EnvMnemonic); mnemonic != "" {
		if !ValidateMnemonic(mnemonic) {
			return nil, errors.New("invalid mnemonic in LUX_MNEMONIC")
		}
		return DeriveAllKeys(name, mnemonic)
	}

	// Priority 2: LUX_PRIVATE_KEY (hex-encoded EC key)
	if privKeyHex := os.Getenv(EnvPrivateKey); privKeyHex != "" {
		privKeyHex = strings.TrimPrefix(privKeyHex, "0x")
		privKeyBytes, err := hex.DecodeString(privKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid hex in LUX_PRIVATE_KEY: %w", err)
		}

		privKey, err := secp256k1.ToPrivateKey(privKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("invalid private key in LUX_PRIVATE_KEY: %w", err)
		}

		// Create minimal key set with just EC key
		ks := &HDKeySet{
			Name:         name,
			ECPrivateKey: privKeyBytes,
			ECPublicKey:  privKey.PublicKey().Bytes(),
			ECAddress:    deriveECAddress(privKey.PublicKey().Bytes()),
		}

		// Also load BLS key if provided
		if blsHex := os.Getenv(EnvBLSKey); blsHex != "" {
			blsHex = strings.TrimPrefix(blsHex, "0x")
			blsBytes, err := hex.DecodeString(blsHex)
			if err == nil {
				ks.BLSPrivateKey = blsBytes
				ks.BLSPublicKey, ks.BLSPoP, _ = deriveBLSPublicKey(blsBytes)
			}
		}

		return ks, nil
	}

	return nil, errors.New("no key found in environment (set LUX_MNEMONIC or LUX_PRIVATE_KEY)")
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
	if os.Getenv(EnvMnemonic) != "" || os.Getenv(EnvPrivateKey) != "" {
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

func init() {
	RegisterBackend(NewEnvBackend())
}
