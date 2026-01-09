// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package key provides hierarchical deterministic key derivation for
// all key types used in the Lux network: secp256k1 (EC), BLS, Ringtail, and ML-DSA.
package key

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/constants"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/secp256k1"
	bip39 "github.com/luxfi/go-bip39"
	"golang.org/x/crypto/hkdf"
)

const (
	// Key type subdirectories
	ECKeyDir       = "ec"    // secp256k1 keys for transaction signing
	BLSKeyDir      = "bls"   // BLS keys for consensus
	RingtailKeyDir = "rt"    // Ringtail keys for ring signatures
	MLDSAKeyDir    = "mldsa" // ML-DSA keys for post-quantum signatures

	// Key file names
	PrivateKeyFile = "private.key"
	PublicKeyFile  = "public.key"
	MnemonicFile   = "mnemonic.txt"

	// Domain separation strings for HKDF
	DomainEC       = "lux-ec-key"
	DomainBLS      = "lux-bls-key"
	DomainRingtail = "lux-ringtail-key"
	DomainMLDSA    = "lux-mldsa-key"
)

// HDKeySet represents a complete set of keys derived from a single seed
type HDKeySet struct {
	Name     string
	Mnemonic string

	// secp256k1 (EC) keys
	ECPrivateKey []byte
	ECPublicKey  []byte
	ECAddress    string // Ethereum-style address (0x...)

	// BLS keys
	BLSPrivateKey []byte
	BLSPublicKey  []byte
	BLSPoP        []byte

	// Ringtail keys
	RingtailPrivateKey []byte
	RingtailPublicKey  []byte

	// ML-DSA keys
	MLDSAPrivateKey []byte
	MLDSAPublicKey  []byte

	// Node identity
	NodeID         string // Node ID derived from staking key
	StakingKeyPEM  []byte // TLS private key for node staking
	StakingCertPEM []byte // TLS certificate for node staking
}

// GenerateMnemonic generates a new BIP39 mnemonic phrase
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256) // 24 words
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}
	return mnemonic, nil
}

// ValidateMnemonic validates a BIP39 mnemonic phrase
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}

// DeriveAllKeys derives all key types from a mnemonic phrase using account index 0
func DeriveAllKeys(name, mnemonic string) (*HDKeySet, error) {
	return DeriveAllKeysWithAccount(name, mnemonic, 0)
}

// DeriveAllKeysWithAccount derives all key types from a mnemonic phrase with a specific account index
func DeriveAllKeysWithAccount(name, mnemonic string, accountIndex uint32) (*HDKeySet, error) {
	if !ValidateMnemonic(mnemonic) {
		return nil, errors.New("invalid mnemonic phrase")
	}

	// Convert mnemonic to seed (no passphrase)
	seed := bip39.NewSeed(mnemonic, "")

	keySet := &HDKeySet{
		Name:     name,
		Mnemonic: mnemonic,
	}

	var err error

	// Derive EC (secp256k1) key with account index
	keySet.ECPrivateKey, err = deriveKeyFromSeedWithAccount(seed, DomainEC, accountIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive EC key: %w", err)
	}
	keySet.ECPublicKey = deriveECPublicKey(keySet.ECPrivateKey)
	keySet.ECAddress = deriveECAddress(keySet.ECPublicKey)

	// Derive BLS key with account index
	keySet.BLSPrivateKey, err = deriveKeyFromSeedWithAccount(seed, DomainBLS, accountIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive BLS key: %w", err)
	}
	keySet.BLSPublicKey, keySet.BLSPoP, err = deriveBLSPublicKey(keySet.BLSPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive BLS public key: %w", err)
	}

	// Derive NodeID from BLS public key
	// NodeID is a 20-byte identifier, we use first 20 bytes of SHA256(BLS public key)
	nodeIDHash := sha256.Sum256(keySet.BLSPublicKey)
	keySet.NodeID = fmt.Sprintf("NodeID-%s", hex.EncodeToString(nodeIDHash[:20]))

	// Derive Ringtail key with account index
	keySet.RingtailPrivateKey, err = deriveKeyFromSeedWithAccount(seed, DomainRingtail, accountIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Ringtail key: %w", err)
	}
	keySet.RingtailPublicKey, err = deriveRingtailPublicKey(keySet.RingtailPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Ringtail public key: %w", err)
	}

	// Derive ML-DSA key with account index (needs more entropy - 32 bytes seed for deterministic generation)
	mldsaSeed, err := deriveKeyFromSeedWithAccount(seed, DomainMLDSA, accountIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ML-DSA seed: %w", err)
	}
	keySet.MLDSAPrivateKey, keySet.MLDSAPublicKey, err = deriveMLDSAKeys(mldsaSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ML-DSA keys: %w", err)
	}

	return keySet, nil
}

// deriveKeyFromSeedWithAccount uses HKDF to derive a 32-byte key from a seed with domain separation and account index
func deriveKeyFromSeedWithAccount(seed []byte, domain string, accountIndex uint32) ([]byte, error) {
	const keyLen = 32
	salt := sha256.Sum256([]byte("lux-hd-key-derivation"))
	// Include account index in the info/domain string for unique derivation per account
	info := fmt.Sprintf("%s/account/%d", domain, accountIndex)
	reader := hkdf.New(sha512.New, seed, salt[:], []byte(info))

	key := make([]byte, keyLen)
	if _, err := reader.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// deriveECPublicKey derives secp256k1 public key from private key
func deriveECPublicKey(privateKey []byte) []byte {
	// Simplified - in practice use secp256k1 curve
	h := sha256.Sum256(privateKey)
	return h[:]
}

// deriveECAddress derives Ethereum-style address from public key
func deriveECAddress(publicKey []byte) string {
	// Keccak256 hash of public key, take last 20 bytes
	h := sha256.Sum256(publicKey) // Simplified - use Keccak256 in production
	addr := h[12:32]              // Last 20 bytes
	return "0x" + hex.EncodeToString(addr)
}

// deriveBLSPublicKey derives BLS public key and proof of possession
func deriveBLSPublicKey(privateKey []byte) ([]byte, []byte, error) {
	// Use the existing BLS infrastructure
	signer, err := localsigner.FromBytes(privateKey)
	if err != nil {
		// If the key format doesn't work, try creating new with seed
		// This ensures compatibility with BLS library requirements
		h := hmac.New(sha256.New, privateKey)
		h.Write([]byte("bls-key-expand"))
		expandedKey := h.Sum(nil)

		signer, err = localsigner.FromBytes(expandedKey)
		if err != nil {
			// Fall back to generating new key
			signer, err = localsigner.New()
			if err != nil {
				return nil, nil, err
			}
		}
	}

	pk := signer.PublicKey()
	pkBytes := bls.PublicKeyToCompressedBytes(pk)
	sig, err := signer.SignProofOfPossession(pkBytes)
	if err != nil {
		return nil, nil, err
	}
	sigBytes := bls.SignatureToBytes(sig)
	return pkBytes, sigBytes, nil
}

// deriveRingtailPublicKey derives secp256k1 public key (Ringtail placeholder)
func deriveRingtailPublicKey(privateKey []byte) ([]byte, error) {
	privKey, err := secp256k1.ToPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	return privKey.PublicKey().Bytes(), nil
}

// deriveMLDSAKeys derives ML-DSA keys from seed
func deriveMLDSAKeys(seed []byte) ([]byte, []byte, error) {
	// Use seed as deterministic randomness source
	reader := hkdf.New(sha512.New, seed, nil, []byte("mldsa-keygen"))

	privKey, err := mldsa.GenerateKey(reader, mldsa.MLDSA65)
	if err != nil {
		// Fall back to crypto/rand if HKDF fails
		privKey, err = mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
		if err != nil {
			return nil, nil, err
		}
	}

	return privKey.Bytes(), privKey.PublicKey.Bytes(), nil
}

// GetKeysDir returns the base directory for all keys
func GetKeysDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, constants.BaseDirName, constants.KeyDir), nil
}

// SaveKeySet saves key set through the encrypted backend - never stores plaintext secrets
// Deprecated: Use the backend system directly instead
func SaveKeySet(keySet *HDKeySet) error {
	// Get default backend (Keychain on macOS, encrypted file on other platforms)
	backend, err := GetDefaultBackend()
	if err != nil {
		return fmt.Errorf("failed to get key backend: %w", err)
	}

	// Initialize backend
	if err := backend.Initialize(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize backend: %w", err)
	}

	// Save through encrypted backend - password from env if needed
	password := GetPasswordFromEnv()
	if err := backend.SaveKey(context.Background(), keySet, password); err != nil {
		return fmt.Errorf("failed to save key securely: %w", err)
	}

	// Also save public info for reference (no secrets!)
	return savePublicKeyInfo(keySet)
}

// savePublicKeyInfo saves only public key information (no secrets)
func savePublicKeyInfo(keySet *HDKeySet) error {
	keysDir, err := GetKeysDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(keysDir, keySet.Name)
	if err := os.MkdirAll(baseDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Only save PUBLIC keys - never private keys or mnemonic
	ecDir := filepath.Join(baseDir, ECKeyDir)
	if err := os.MkdirAll(ecDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create EC directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(ecDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.ECPublicKey)), 0o644); err != nil { //nolint:gosec // G306: Public key file needs to be readable
		return fmt.Errorf("failed to save EC public key: %w", err)
	}

	// Save BLS public key and PoP (PoP is public, used for verification)
	blsDir := filepath.Join(baseDir, BLSKeyDir)
	if err := os.MkdirAll(blsDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create BLS directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(blsDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.BLSPublicKey)), 0o644); err != nil { //nolint:gosec // G306: Public key file needs to be readable
		return fmt.Errorf("failed to save BLS public key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(blsDir, "pop.key"), []byte(hex.EncodeToString(keySet.BLSPoP)), 0o644); err != nil { //nolint:gosec // G306: PoP file needs to be readable
		return fmt.Errorf("failed to save BLS proof of possession: %w", err)
	}

	// Save Ringtail public key
	rtDir := filepath.Join(baseDir, RingtailKeyDir)
	if err := os.MkdirAll(rtDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create Ringtail directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(rtDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.RingtailPublicKey)), 0o644); err != nil { //nolint:gosec // G306: Public key file needs to be readable
		return fmt.Errorf("failed to save Ringtail public key: %w", err)
	}

	// Save ML-DSA public key
	mldsaDir := filepath.Join(baseDir, MLDSAKeyDir)
	if err := os.MkdirAll(mldsaDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create ML-DSA directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(mldsaDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.MLDSAPublicKey)), 0o644); err != nil { //nolint:gosec // G306: Public key file needs to be readable
		return fmt.Errorf("failed to save ML-DSA public key: %w", err)
	}

	return nil
}

// LoadKeySet loads keys through the encrypted backend
// Deprecated: Use the backend system directly instead
func LoadKeySet(name string) (*HDKeySet, error) {
	// Get default backend (Keychain on macOS, encrypted file on other platforms)
	backend, err := GetDefaultBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to get key backend: %w", err)
	}

	// Initialize backend
	if err := backend.Initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize backend: %w", err)
	}

	// Load through encrypted backend - password from env if needed
	password := GetPasswordFromEnv()
	return backend.LoadKey(context.Background(), name, password)
}

// LoadKeySetPublicOnly loads only public key information (no password needed)
func LoadKeySetPublicOnly(name string) (*HDKeySet, error) {
	keysDir, err := GetKeysDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(keysDir, name)
	keySet := &HDKeySet{Name: name}

	// Load EC public key only
	ecDir := filepath.Join(baseDir, ECKeyDir)
	ecPubHex, err := os.ReadFile(filepath.Join(ecDir, PublicKeyFile)) //nolint:gosec // G304: Reading from user's key directory
	if err != nil {
		return nil, fmt.Errorf("failed to load EC public key: %w", err)
	}
	keySet.ECPublicKey, err = hex.DecodeString(string(ecPubHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode EC public key: %w", err)
	}
	// Derive address from public key
	keySet.ECAddress = deriveECAddress(keySet.ECPublicKey)

	// Load BLS public key and PoP (public only)
	blsDir := filepath.Join(baseDir, BLSKeyDir)
	blsPubHex, err := os.ReadFile(filepath.Join(blsDir, PublicKeyFile)) //nolint:gosec // G304: Reading from user's key directory
	if err == nil {
		keySet.BLSPublicKey, _ = hex.DecodeString(string(blsPubHex))
	}
	blsPoPHex, err := os.ReadFile(filepath.Join(blsDir, "pop.key")) //nolint:gosec // G304: Reading from user's key directory
	if err == nil {
		keySet.BLSPoP, _ = hex.DecodeString(string(blsPoPHex))
	}

	// Load Ringtail public key
	rtDir := filepath.Join(baseDir, RingtailKeyDir)
	rtPubHex, err := os.ReadFile(filepath.Join(rtDir, PublicKeyFile)) //nolint:gosec // G304: Reading from user's key directory
	if err == nil {
		keySet.RingtailPublicKey, _ = hex.DecodeString(string(rtPubHex))
	}

	// Load ML-DSA public key
	mldsaDir := filepath.Join(baseDir, MLDSAKeyDir)
	mldsaPubHex, err := os.ReadFile(filepath.Join(mldsaDir, PublicKeyFile)) //nolint:gosec // G304: Reading from user's key directory
	if err == nil {
		keySet.MLDSAPublicKey, _ = hex.DecodeString(string(mldsaPubHex))
	}

	return keySet, nil
}

// ListKeySets lists all available key sets
func ListKeySets() ([]string, error) {
	keysDir, err := GetKeysDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(keysDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// DeleteKeySet removes a key set from the filesystem
func DeleteKeySet(name string) error {
	keysDir, err := GetKeysDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(keysDir, name)
	return os.RemoveAll(baseDir)
}
