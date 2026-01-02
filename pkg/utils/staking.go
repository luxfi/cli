// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/secp256k1"
	evmclient "github.com/luxfi/evm/plugin/evm/client"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/staking"
	"github.com/luxfi/node/vms/platformvm"
)

func NewBlsSecretKeyBytes() ([]byte, error) {
	blsSignerKey, err := localsigner.New()
	if err != nil {
		return nil, err
	}
	return blsSignerKey.ToBytes(), nil
}

func ToNodeID(certBytes []byte) (ids.NodeID, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return ids.EmptyNodeID, fmt.Errorf("failed to decode certificate")
	}
	cert, err := staking.ParseCertificate(block.Bytes)
	if err != nil {
		return ids.EmptyNodeID, err
	}
	idsCert := &ids.Certificate{
		Raw:       cert.Raw,
		PublicKey: cert.PublicKey,
	}
	return ids.NodeIDFromCert(idsCert), nil
}

func ToBLSPoP(keyBytes []byte) (
	[]byte, // bls public key
	[]byte, // bls proof of possession
	error,
) {
	localSigner, err := localsigner.FromBytes(keyBytes)
	if err != nil {
		return nil, nil, err
	}
	// LocalSigner has the secret key as a private field, but we can get the public key
	// and sign a proof of possession directly
	pk := localSigner.PublicKey()
	pkBytes := bls.PublicKeyToCompressedBytes(pk)
	sig, err := localSigner.SignProofOfPossession(pkBytes)
	if err != nil {
		return nil, nil, err
	}
	sigBytes := bls.SignatureToBytes(sig)
	return pkBytes, sigBytes, nil
}

// GetNodeParams returns node id, bls public key and bls proof of possession
func GetNodeParams(nodeDir string) (
	ids.NodeID,
	[]byte, // bls public key
	[]byte, // bls proof of possession
	error,
) {
	certBytes, err := os.ReadFile(filepath.Join(nodeDir, constants.StakerCertFileName))
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	nodeID, err := ToNodeID(certBytes)
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	blsKeyBytes, err := os.ReadFile(filepath.Join(nodeDir, constants.BLSKeyFileName))
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	blsPub, blsPoP, err := ToBLSPoP(blsKeyBytes)
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	return nodeID, blsPub, blsPoP, nil
}

func GetRemainingValidationTime(networkEndpoint string, nodeID ids.NodeID, subnetID ids.ID, startTime time.Time) (time.Duration, error) {
	ctx, cancel := GetAPIContext()
	defer cancel()
	platformCli := platformvm.NewClient(networkEndpoint)
	vs, err := platformCli.GetCurrentValidators(ctx, subnetID, nil)
	cancel()
	if err != nil {
		return 0, err
	}
	for _, v := range vs {
		if v.NodeID == nodeID {
			return time.Unix(int64(v.EndTime), 0).Sub(startTime), nil
		}
	}
	return 0, errors.New("nodeID not found in validator set: " + nodeID.String())
}

// GetL1ValidatorUptimeSeconds returns the uptime of the L1 validator
func GetL1ValidatorUptimeSeconds(rpcURL string, nodeID ids.NodeID) (uint64, error) {
	ctx, cancel := GetAPIContext()
	defer cancel()
	networkEndpoint, blockchainID, err := SplitRPCURI(rpcURL)
	if err != nil {
		return 0, err
	}
	evmCli := evmclient.NewClient(networkEndpoint, blockchainID)
	validators, err := evmCli.GetCurrentValidators(ctx, []ids.NodeID{nodeID})
	if err != nil {
		return 0, err
	}
	if len(validators) > 0 {
		deductibleSeconds := uint64(constants.ValidatorUptimeDeductible.Seconds())
		if validators[0].UptimeSeconds > deductibleSeconds {
			return validators[0].UptimeSeconds - deductibleSeconds, nil
		}
		return 0, nil
	}

	return 0, errors.New("nodeID not found in validator set: " + nodeID.String())
}

// NewRingtailKeyBytes generates a new secp256k1 private key and returns it as bytes
// Note: "Ringtail" is a placeholder name - we use standard secp256k1 for now
func NewRingtailKeyBytes() ([]byte, error) {
	privKey, err := secp256k1.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secp256k1 key: %w", err)
	}
	return privKey.Bytes(), nil
}

// ToRingtailPublicKey converts secp256k1 private key bytes to public key bytes
func ToRingtailPublicKey(keyBytes []byte) ([]byte, error) {
	privKey, err := secp256k1.ToPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secp256k1 private key: %w", err)
	}
	return privKey.PublicKey().Bytes(), nil
}

// NewMLDSAKeyBytes generates a new ML-DSA private key and returns it as bytes
// Uses MLDSA65 (192-bit security, NIST Level 3) as the default
func NewMLDSAKeyBytes() ([]byte, error) {
	privKey, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ML-DSA key: %w", err)
	}
	return privKey.Bytes(), nil
}

// ToMLDSAPublicKey converts ML-DSA private key bytes to public key bytes
func ToMLDSAPublicKey(keyBytes []byte) ([]byte, error) {
	privKey, err := mldsa.PrivateKeyFromBytes(mldsa.MLDSA65, keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ML-DSA private key: %w", err)
	}
	return privKey.PublicKey.Bytes(), nil
}

// QuantumKeys holds all quantum-safe keys for a validator
type QuantumKeys struct {
	BLSSecretKey      []byte
	BLSPublicKey      []byte
	BLSPoP            []byte
	RingtailSecretKey []byte
	RingtailPublicKey []byte
	MLDSASecretKey    []byte
	MLDSAPublicKey    []byte
}

// GenerateAllQuantumKeys generates BLS, Ringtail, and ML-DSA keys for a validator
func GenerateAllQuantumKeys() (*QuantumKeys, error) {
	keys := &QuantumKeys{}
	var err error

	// Generate BLS key
	keys.BLSSecretKey, err = NewBlsSecretKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("BLS key generation failed: %w", err)
	}
	keys.BLSPublicKey, keys.BLSPoP, err = ToBLSPoP(keys.BLSSecretKey)
	if err != nil {
		return nil, fmt.Errorf("BLS public key derivation failed: %w", err)
	}

	// Generate Ringtail key
	keys.RingtailSecretKey, err = NewRingtailKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("Ringtail key generation failed: %w", err)
	}
	keys.RingtailPublicKey, err = ToRingtailPublicKey(keys.RingtailSecretKey)
	if err != nil {
		return nil, fmt.Errorf("Ringtail public key derivation failed: %w", err)
	}

	// Generate ML-DSA key
	keys.MLDSASecretKey, err = NewMLDSAKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("ML-DSA key generation failed: %w", err)
	}
	keys.MLDSAPublicKey, err = ToMLDSAPublicKey(keys.MLDSASecretKey)
	if err != nil {
		return nil, fmt.Errorf("ML-DSA public key derivation failed: %w", err)
	}

	return keys, nil
}

// SaveQuantumKeys saves all quantum keys to the specified directory
func SaveQuantumKeys(nodeDir string, keys *QuantumKeys) error {
	// Save BLS key
	blsPath := filepath.Join(nodeDir, constants.BLSKeyFileName)
	if err := os.WriteFile(blsPath, keys.BLSSecretKey, 0o600); err != nil {
		return fmt.Errorf("failed to save BLS key: %w", err)
	}

	// Save Ringtail key (hex encoded)
	ringtailPath := filepath.Join(nodeDir, constants.RingtailKeyFileName)
	ringtailHex := hex.EncodeToString(keys.RingtailSecretKey)
	if err := os.WriteFile(ringtailPath, []byte(ringtailHex), 0o600); err != nil {
		return fmt.Errorf("failed to save Ringtail key: %w", err)
	}

	// Save ML-DSA key (hex encoded)
	mldsaPath := filepath.Join(nodeDir, constants.MLDSAKeyFileName)
	mldsaHex := hex.EncodeToString(keys.MLDSASecretKey)
	if err := os.WriteFile(mldsaPath, []byte(mldsaHex), 0o600); err != nil {
		return fmt.Errorf("failed to save ML-DSA key: %w", err)
	}

	return nil
}

// LoadQuantumKeys loads all quantum keys from the specified directory
func LoadQuantumKeys(nodeDir string) (*QuantumKeys, error) {
	keys := &QuantumKeys{}
	var err error

	// Load BLS key
	blsPath := filepath.Join(nodeDir, constants.BLSKeyFileName)
	keys.BLSSecretKey, err = os.ReadFile(blsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load BLS key: %w", err)
	}
	keys.BLSPublicKey, keys.BLSPoP, err = ToBLSPoP(keys.BLSSecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive BLS public key: %w", err)
	}

	// Load Ringtail key
	ringtailPath := filepath.Join(nodeDir, constants.RingtailKeyFileName)
	ringtailHex, err := os.ReadFile(ringtailPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load Ringtail key: %w", err)
	}
	keys.RingtailSecretKey, err = hex.DecodeString(string(ringtailHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode Ringtail key: %w", err)
	}
	keys.RingtailPublicKey, err = ToRingtailPublicKey(keys.RingtailSecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Ringtail public key: %w", err)
	}

	// Load ML-DSA key
	mldsaPath := filepath.Join(nodeDir, constants.MLDSAKeyFileName)
	mldsaHex, err := os.ReadFile(mldsaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ML-DSA key: %w", err)
	}
	keys.MLDSASecretKey, err = hex.DecodeString(string(mldsaHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-DSA key: %w", err)
	}
	keys.MLDSAPublicKey, err = ToMLDSAPublicKey(keys.MLDSASecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ML-DSA public key: %w", err)
	}

	return keys, nil
}

// GetQuantumNodeParams returns node id and all quantum public keys
func GetQuantumNodeParams(nodeDir string) (
	ids.NodeID,
	*QuantumKeys,
	error,
) {
	certBytes, err := os.ReadFile(filepath.Join(nodeDir, constants.StakerCertFileName))
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}
	nodeID, err := ToNodeID(certBytes)
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}
	keys, err := LoadQuantumKeys(nodeDir)
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}
	return nodeID, keys, nil
}
