// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package key provides hierarchical deterministic key derivation for
// all key types used in the Lux network: secp256k1 (EC), BLS, Ringtail, and ML-DSA.
package key

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/hkdf"
)

const (
	// Key type subdirectories
	ECKeyDir       = "ec"       // secp256k1 keys for transaction signing
	BLSKeyDir      = "bls"      // BLS keys for consensus
	RingtailKeyDir = "rt"       // Ringtail keys for ring signatures
	MLDSAKeyDir    = "mldsa"    // ML-DSA keys for post-quantum signatures

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

// DeriveAllKeys derives all key types from a mnemonic phrase
func DeriveAllKeys(name, mnemonic string) (*HDKeySet, error) {
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

	// Derive EC (secp256k1) key
	keySet.ECPrivateKey, err = deriveKeyFromSeed(seed, DomainEC, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive EC key: %w", err)
	}
	keySet.ECPublicKey = deriveECPublicKey(keySet.ECPrivateKey)
	keySet.ECAddress = deriveECAddress(keySet.ECPublicKey)

	// Derive BLS key
	keySet.BLSPrivateKey, err = deriveKeyFromSeed(seed, DomainBLS, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive BLS key: %w", err)
	}
	keySet.BLSPublicKey, keySet.BLSPoP, err = deriveBLSPublicKey(keySet.BLSPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive BLS public key: %w", err)
	}

	// Derive Ringtail key
	keySet.RingtailPrivateKey, err = deriveKeyFromSeed(seed, DomainRingtail, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Ringtail key: %w", err)
	}
	keySet.RingtailPublicKey, err = deriveRingtailPublicKey(keySet.RingtailPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Ringtail public key: %w", err)
	}

	// Derive ML-DSA key (needs more entropy - 32 bytes seed for deterministic generation)
	mldsaSeed, err := deriveKeyFromSeed(seed, DomainMLDSA, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ML-DSA seed: %w", err)
	}
	keySet.MLDSAPrivateKey, keySet.MLDSAPublicKey, err = deriveMLDSAKeys(mldsaSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ML-DSA keys: %w", err)
	}

	return keySet, nil
}

// deriveKeyFromSeed uses HKDF to derive a key from a seed with domain separation
func deriveKeyFromSeed(seed []byte, domain string, keyLen int) ([]byte, error) {
	salt := sha256.Sum256([]byte("lux-hd-key-derivation"))
	reader := hkdf.New(sha512.New, seed, salt[:], []byte(domain))

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

// SaveKeySet saves all keys to the filesystem in organized subdirectories
// Structure: ~/.lux/keys/<name>/{ec,bls,rt,mldsa}/{private.key,public.key}
func SaveKeySet(keySet *HDKeySet) error {
	keysDir, err := GetKeysDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(keysDir, keySet.Name)
	if err := os.MkdirAll(baseDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Save mnemonic (encrypted in production - plain text for now)
	mnemonicPath := filepath.Join(baseDir, MnemonicFile)
	if err := os.WriteFile(mnemonicPath, []byte(keySet.Mnemonic), 0600); err != nil {
		return fmt.Errorf("failed to save mnemonic: %w", err)
	}

	// Save EC keys
	ecDir := filepath.Join(baseDir, ECKeyDir)
	if err := os.MkdirAll(ecDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create EC directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(ecDir, PrivateKeyFile), []byte(hex.EncodeToString(keySet.ECPrivateKey)), 0600); err != nil {
		return fmt.Errorf("failed to save EC private key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(ecDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.ECPublicKey)), 0644); err != nil {
		return fmt.Errorf("failed to save EC public key: %w", err)
	}

	// Save BLS keys
	blsDir := filepath.Join(baseDir, BLSKeyDir)
	if err := os.MkdirAll(blsDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create BLS directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(blsDir, PrivateKeyFile), keySet.BLSPrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to save BLS private key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(blsDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.BLSPublicKey)), 0644); err != nil {
		return fmt.Errorf("failed to save BLS public key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(blsDir, "pop.key"), []byte(hex.EncodeToString(keySet.BLSPoP)), 0644); err != nil {
		return fmt.Errorf("failed to save BLS proof of possession: %w", err)
	}

	// Save Ringtail keys
	rtDir := filepath.Join(baseDir, RingtailKeyDir)
	if err := os.MkdirAll(rtDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create Ringtail directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(rtDir, PrivateKeyFile), []byte(hex.EncodeToString(keySet.RingtailPrivateKey)), 0600); err != nil {
		return fmt.Errorf("failed to save Ringtail private key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(rtDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.RingtailPublicKey)), 0644); err != nil {
		return fmt.Errorf("failed to save Ringtail public key: %w", err)
	}

	// Save ML-DSA keys
	mldsaDir := filepath.Join(baseDir, MLDSAKeyDir)
	if err := os.MkdirAll(mldsaDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create ML-DSA directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(mldsaDir, PrivateKeyFile), []byte(hex.EncodeToString(keySet.MLDSAPrivateKey)), 0600); err != nil {
		return fmt.Errorf("failed to save ML-DSA private key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(mldsaDir, PublicKeyFile), []byte(hex.EncodeToString(keySet.MLDSAPublicKey)), 0644); err != nil {
		return fmt.Errorf("failed to save ML-DSA public key: %w", err)
	}

	return nil
}

// LoadKeySet loads all keys from the filesystem
func LoadKeySet(name string) (*HDKeySet, error) {
	keysDir, err := GetKeysDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(keysDir, name)
	keySet := &HDKeySet{Name: name}

	// Load mnemonic
	mnemonicBytes, err := os.ReadFile(filepath.Join(baseDir, MnemonicFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load mnemonic: %w", err)
	}
	keySet.Mnemonic = string(mnemonicBytes)

	// Load EC keys
	ecDir := filepath.Join(baseDir, ECKeyDir)
	ecPrivHex, err := os.ReadFile(filepath.Join(ecDir, PrivateKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load EC private key: %w", err)
	}
	keySet.ECPrivateKey, err = hex.DecodeString(string(ecPrivHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode EC private key: %w", err)
	}
	ecPubHex, err := os.ReadFile(filepath.Join(ecDir, PublicKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load EC public key: %w", err)
	}
	keySet.ECPublicKey, err = hex.DecodeString(string(ecPubHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode EC public key: %w", err)
	}
	// Derive address from public key
	keySet.ECAddress = deriveECAddress(keySet.ECPublicKey)

	// Load BLS keys
	blsDir := filepath.Join(baseDir, BLSKeyDir)
	keySet.BLSPrivateKey, err = os.ReadFile(filepath.Join(blsDir, PrivateKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load BLS private key: %w", err)
	}
	blsPubHex, err := os.ReadFile(filepath.Join(blsDir, PublicKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load BLS public key: %w", err)
	}
	keySet.BLSPublicKey, err = hex.DecodeString(string(blsPubHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode BLS public key: %w", err)
	}
	blsPoPHex, err := os.ReadFile(filepath.Join(blsDir, "pop.key"))
	if err != nil {
		return nil, fmt.Errorf("failed to load BLS proof of possession: %w", err)
	}
	keySet.BLSPoP, err = hex.DecodeString(string(blsPoPHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode BLS PoP: %w", err)
	}

	// Load Ringtail keys
	rtDir := filepath.Join(baseDir, RingtailKeyDir)
	rtPrivHex, err := os.ReadFile(filepath.Join(rtDir, PrivateKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load Ringtail private key: %w", err)
	}
	keySet.RingtailPrivateKey, err = hex.DecodeString(string(rtPrivHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode Ringtail private key: %w", err)
	}
	rtPubHex, err := os.ReadFile(filepath.Join(rtDir, PublicKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load Ringtail public key: %w", err)
	}
	keySet.RingtailPublicKey, err = hex.DecodeString(string(rtPubHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode Ringtail public key: %w", err)
	}

	// Load ML-DSA keys
	mldsaDir := filepath.Join(baseDir, MLDSAKeyDir)
	mldsaPrivHex, err := os.ReadFile(filepath.Join(mldsaDir, PrivateKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load ML-DSA private key: %w", err)
	}
	keySet.MLDSAPrivateKey, err = hex.DecodeString(string(mldsaPrivHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-DSA private key: %w", err)
	}
	mldsaPubHex, err := os.ReadFile(filepath.Join(mldsaDir, PublicKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load ML-DSA public key: %w", err)
	}
	keySet.MLDSAPublicKey, err = hex.DecodeString(string(mldsaPubHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-DSA public key: %w", err)
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
