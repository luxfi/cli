// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// BLS key creation command

package keycmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/crypto/bls"
)

var (
	blsWithProof bool
	blsValidator bool
)

func newCreateBLSCmd(app *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-bls [keyName]",
		Short: "Create a new BLS key for validator operations",
		Long: `Create a new BLS (Boneh-Lynn-Shacham) key using BLS12-381 curve.
BLS keys are used for validator consensus operations on the P-Chain.

Features:
  - Aggregatable signatures (combine multiple signatures into one)
  - Compact signatures (96 bytes)
  - Used for validator consensus
  - Proof of Possession support

Example:
  lux key create-bls validator1 --validator
  lux key create-bls alice --with-proof`,
		RunE: func(cmd *cobra.Command, args []string) error {
			keyName := ""
			if len(args) > 0 {
				keyName = args[0]
			} else {
				name, err := prompts.PromptString("Enter key name", "my-bls-key", prompts.ValidateNotEmpty)
				if err != nil {
					return err
				}
				keyName = name
			}
			
			// Validate key name
			if err := key.ValidateKeyName(keyName); err != nil {
				return err
			}
			
			// Generate BLS key
			ux.Logger.PrintToUser("Generating BLS key '%s'...", keyName)
			
			start := time.Now()
			privKey, err := bls.NewSecretKey()
			if err != nil {
				return fmt.Errorf("failed to generate BLS key: %w", err)
			}
			elapsed := time.Since(start)
			
			pubKey := bls.PublicFromSecretKey(privKey)
			
			// Generate proof of possession if requested
			var proofOfPossession []byte
			if blsWithProof || blsValidator {
				pop := bls.SignProofOfPossession(privKey, pubKey)
				proofOfPossession = bls.SignatureToBytes(pop)
			}
			
			// Determine output path
			keyPath := app.GetKeyPath(keyName)
			if blsValidator {
				keyPath = strings.Replace(keyPath, ".pk", ".bls.validator", 1)
			} else {
				keyPath = strings.Replace(keyPath, ".pk", ".bls", 1)
			}
			
			// Save key to file
			keyData := BLSKeyFile{
				Algorithm:         "BLS12-381",
				Name:             keyName,
				PrivateKey:       hex.EncodeToString(bls.SecretKeyToBytes(privKey)),
				PublicKey:        hex.EncodeToString(bls.PublicKeyToCompressedBytes(pubKey)),
				ProofOfPossession: hex.EncodeToString(proofOfPossession),
				IsValidator:      blsValidator,
				CreatedAt:        time.Now().Format(time.RFC3339),
			}
			
			jsonData, err := json.MarshalIndent(keyData, "", "  ")
			if err != nil {
				return err
			}
			
			if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
				return err
			}
			
			if err := os.WriteFile(keyPath, jsonData, 0600); err != nil {
				return fmt.Errorf("failed to save key: %w", err)
			}
			
			// Display key information
			ux.Logger.PrintToUser("✅ BLS Key Created Successfully!")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Algorithm:     BLS12-381")
			ux.Logger.PrintToUser("Key Name:      %s", keyName)
			if blsValidator {
				ux.Logger.PrintToUser("Type:          Validator Key")
			}
			ux.Logger.PrintToUser("Generated in:  %v", elapsed)
			ux.Logger.PrintToUser("Saved to:      %s", keyPath)
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Key Sizes:")
			ux.Logger.PrintToUser("  Private Key:   32 bytes")
			ux.Logger.PrintToUser("  Public Key:    48 bytes (compressed)")
			ux.Logger.PrintToUser("  Signature:     96 bytes")
			if blsWithProof || blsValidator {
				ux.Logger.PrintToUser("  PoP:           96 bytes (generated)")
			}
			
			// Usage hints
			ux.Logger.PrintToUser("")
			if blsValidator {
				ux.Logger.PrintToUser("🔐 This key can be used for validator operations on the P-Chain")
				ux.Logger.PrintToUser("   Public Key: %s", hex.EncodeToString(bls.PublicKeyToCompressedBytes(pubKey))[:16]+"...")
			} else {
				ux.Logger.PrintToUser("💡 BLS signatures are aggregatable - multiple signatures can be combined")
				ux.Logger.PrintToUser("   into a single signature for efficient consensus")
			}
			
			return nil
		},
	}
	
	cmd.Flags().BoolVar(&blsWithProof, "with-proof", false, "Generate proof of possession")
	cmd.Flags().BoolVar(&blsValidator, "validator", false, "Create as validator key with proof of possession")
	
	return cmd
}

// BLSKeyFile represents the JSON structure for storing BLS keys
type BLSKeyFile struct {
	Algorithm         string `json:"algorithm"`
	Name              string `json:"name"`
	PrivateKey        string `json:"privateKey"`
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession,omitempty"`
	IsValidator       bool   `json:"isValidator"`
	CreatedAt         string `json:"createdAt"`
}