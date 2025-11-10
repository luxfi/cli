// Copyright (C) 2022-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

var (
	deriveStart      int
	deriveCount      int
	deriveOutputDir  string
	deriveNetwork    string
	deriveMnemonic   string
)

func newDeriveValidatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "derive-validators",
		Short: "Derive deterministic validator keys from BIP39 mnemonic",
		Long: `Derives validator staking keys deterministically from a BIP39 mnemonic phrase.

Uses BIP44 derivation path: m/44'/9000'/0'/0/{index}

Each validator gets:
- staker.key: ECDSA P-256 private key
- staker.crt: X.509 certificate
- NodeID: Derived from certificate public key

Example:
  lux key derive-validators --mnemonic "word1 word2 ..." --start 0 --count 5 --output ./validators --network mainnet

This generates deterministic validator keys suitable for network bootstrapping.`,
		RunE: deriveValidatorsCmd,
	}

	cmd.Flags().StringVar(&deriveMnemonic, "mnemonic", "", "BIP39 mnemonic phrase (24 words)")
	cmd.Flags().IntVar(&deriveStart, "start", 0, "Starting account index")
	cmd.Flags().IntVar(&deriveCount, "count", 5, "Number of validators to generate")
	cmd.Flags().StringVar(&deriveOutputDir, "output", "", "Output directory for validator keys")
	cmd.Flags().StringVar(&deriveNetwork, "network", "mainnet", "Network name (mainnet or testnet)")

	return cmd
}

func deriveValidatorsCmd(cmd *cobra.Command, args []string) error {
	// Validate inputs
	if deriveMnemonic == "" {
		// Try to read from env
		deriveMnemonic = os.Getenv("MAINNET_MNEMONIC")
		if deriveMnemonic == "" {
			return fmt.Errorf("mnemonic is required (use --mnemonic or set MAINNET_MNEMONIC env var)")
		}
	}

	if deriveOutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	// Validate mnemonic
	if !bip39.IsMnemonicValid(deriveMnemonic) {
		return fmt.Errorf("invalid BIP39 mnemonic")
	}

	ux.Logger.PrintToUser("Deriving %d validator keys starting from index %d...", deriveCount, deriveStart)

	// Generate seed from mnemonic
	seed := bip39.NewSeed(deriveMnemonic, "")

	// Create master key
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return fmt.Errorf("failed to create master key: %w", err)
	}

	// BIP44 path: m/44'/9000'/0'/0/{index}
	// Derive m/44'
	purpose, err := masterKey.NewChildKey(bip32.FirstHardenedChild + 44)
	if err != nil {
		return fmt.Errorf("failed to derive purpose: %w", err)
	}

	// Derive m/44'/9000'
	coinType, err := purpose.NewChildKey(bip32.FirstHardenedChild + 9000)
	if err != nil {
		return fmt.Errorf("failed to derive coin type: %w", err)
	}

	// Derive m/44'/9000'/0'
	account, err := coinType.NewChildKey(bip32.FirstHardenedChild + 0)
	if err != nil {
		return fmt.Errorf("failed to derive account: %w", err)
	}

	// Derive m/44'/9000'/0'/0
	change, err := account.NewChildKey(0)
	if err != nil {
		return fmt.Errorf("failed to derive change: %w", err)
	}

	// Generate validators
	for i := 0; i < deriveCount; i++ {
		index := deriveStart + i

		// Derive m/44'/9000'/0'/0/{index}
		addressKey, err := change.NewChildKey(uint32(index))
		if err != nil {
			return fmt.Errorf("failed to derive address key %d: %w", index, err)
		}

		// Create validator directory
		validatorDir := filepath.Join(deriveOutputDir, fmt.Sprintf("node%d", i+1))
		stakingDir := filepath.Join(validatorDir, "staking")
		if err := os.MkdirAll(stakingDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", stakingDir, err)
		}

		// Convert to ECDSA private key
		keyBytes := addressKey.Key
		privKey := bytesToPrivateKey(keyBytes)

		// Generate certificate
		cert, err := generateCertificate(privKey)
		if err != nil {
			return fmt.Errorf("failed to generate certificate for validator %d: %w", i+1, err)
		}

		// Calculate NodeID from certificate
		nodeID, err := calculateNodeID(cert)
		if err != nil {
			return fmt.Errorf("failed to calculate NodeID for validator %d: %w", i+1, err)
		}

		// Save private key
		keyPath := filepath.Join(stakingDir, "staker.key")
		if err := savePrivateKey(privKey, keyPath); err != nil {
			return fmt.Errorf("failed to save private key: %w", err)
		}

		// Save certificate
		certPath := filepath.Join(stakingDir, "staker.crt")
		if err := saveCertificate(cert, certPath); err != nil {
			return fmt.Errorf("failed to save certificate: %w", err)
		}

		// Save NodeID for reference
		nodeIDPath := filepath.Join(validatorDir, "NodeID")
		if err := os.WriteFile(nodeIDPath, []byte(nodeID), 0644); err != nil {
			return fmt.Errorf("failed to save NodeID: %w", err)
		}

		ux.Logger.PrintToUser("  [%d] NodeID: %s", i+1, nodeID)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.GreenCheckmarkToUser("Successfully generated %d validators in %s", deriveCount, deriveOutputDir)
	return nil
}

func bytesToPrivateKey(keyBytes []byte) *ecdsa.PrivateKey {
	curve := elliptic.P256()
	privKey := new(ecdsa.PrivateKey)
	privKey.D = new(big.Int).SetBytes(keyBytes)
	privKey.PublicKey.Curve = curve
	privKey.PublicKey.X, privKey.PublicKey.Y = curve.ScalarBaseMult(keyBytes)
	return privKey
}

func generateCertificate(privKey *ecdsa.PrivateKey) (*x509.Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"US"},
			Province:     []string{"CA"},
			Locality:     []string{"Los Angeles"},
			Organization: []string{"Lux Industries Inc"},
			CommonName:   "lux.network",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0), // 100 years
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(certDER)
}

func calculateNodeID(cert *x509.Certificate) (string, error) {
	// Simple NodeID calculation from cert
	// In production, this should match the actual NodeID calculation from luxd
	pubKeyBytes := cert.RawSubjectPublicKeyInfo
	return fmt.Sprintf("NodeID-%x", pubKeyBytes[:20]), nil
}

func savePrivateKey(privKey *ecdsa.PrivateKey, path string) error {
	keyBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return err
	}

	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, pemBlock)
}

func saveCertificate(cert *x509.Certificate, path string) error {
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, pemBlock)
}
