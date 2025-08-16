// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// ML-DSA (Dilithium) key creation command

package keycmd

import (
	"crypto/rand"
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
	"github.com/luxfi/crypto/mldsa"
)

var (
	mldsaLevel     string
	mldsaBenchmark bool
)

func newCreateMLDSACmd(app *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-mldsa [keyName]",
		Short: "Create a new ML-DSA (Dilithium) post-quantum signature key",
		Long: `Create a new ML-DSA (Module Lattice Digital Signature Algorithm) key.
ML-DSA is NIST's standardized lattice-based signature scheme (FIPS 204).

Security Levels:
  44 - Level 2 security (~128-bit), 2.4 KB signatures
  65 - Level 3 security (~192-bit), 3.3 KB signatures (recommended)
  87 - Level 5 security (~256-bit), 4.6 KB signatures

Example:
  lux key create-mldsa alice --level 65
  lux key create-mldsa bob --level 87 --benchmark`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mldsaBenchmark {
				return benchmarkMLDSA()
			}
			
			keyName := ""
			if len(args) > 0 {
				keyName = args[0]
			} else {
				name, err := prompts.PromptString("Enter key name", "my-mldsa-key", prompts.ValidateNotEmpty)
				if err != nil {
					return err
				}
				keyName = name
			}
			
			// Validate key name
			if err := key.ValidateKeyName(keyName); err != nil {
				return err
			}
			
			// Select security level if not provided
			if mldsaLevel == "" {
				level, err := promptMLDSALevel()
				if err != nil {
					return err
				}
				mldsaLevel = level
			}
			
			// Validate level
			var mode mldsa.Mode
			switch mldsaLevel {
			case "44", "2":
				mode = mldsa.MLDSA44
				mldsaLevel = "44"
			case "65", "3":
				mode = mldsa.MLDSA65
				mldsaLevel = "65"
			case "87", "5":
				mode = mldsa.MLDSA87
				mldsaLevel = "87"
			default:
				return fmt.Errorf("invalid ML-DSA level: %s (use 44, 65, or 87)", mldsaLevel)
			}
			
			// Generate the key
			ux.Logger.PrintToUser("Generating ML-DSA-%s key '%s'...", mldsaLevel, keyName)
			
			start := time.Now()
			privKey, err := mldsa.GenerateKey(rand.Reader, mode)
			if err != nil {
				return fmt.Errorf("failed to generate ML-DSA key: %w", err)
			}
			elapsed := time.Since(start)
			
			// Determine output path
			keyPath := app.GetKeyPath(keyName)
			keyPath = strings.Replace(keyPath, ".pk", ".mldsa", 1)
			
			// Save key to file
			keyData := MLDSAKeyFile{
				Algorithm:  fmt.Sprintf("ML-DSA-%s", mldsaLevel),
				Name:       keyName,
				Level:      mldsaLevel,
				PrivateKey: hex.EncodeToString(privKey.Bytes()),
				PublicKey:  hex.EncodeToString(privKey.PublicKey.Bytes()),
				CreatedAt:  time.Now().Format(time.RFC3339),
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
			ux.Logger.PrintToUser("✅ ML-DSA Key Created Successfully!")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Algorithm:     ML-DSA-%s", mldsaLevel)
			ux.Logger.PrintToUser("Security:      NIST Level %s", getSecurityLevel(mldsaLevel))
			ux.Logger.PrintToUser("Key Name:      %s", keyName)
			ux.Logger.PrintToUser("Generated in:  %v", elapsed)
			ux.Logger.PrintToUser("Saved to:      %s", keyPath)
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Key Sizes:")
			
			var privSize, pubSize, sigSize int
			switch mode {
			case mldsa.MLDSA44:
				privSize, pubSize, sigSize = mldsa.MLDSA44PrivateKeySize, mldsa.MLDSA44PublicKeySize, mldsa.MLDSA44SignatureSize
			case mldsa.MLDSA65:
				privSize, pubSize, sigSize = mldsa.MLDSA65PrivateKeySize, mldsa.MLDSA65PublicKeySize, mldsa.MLDSA65SignatureSize
			case mldsa.MLDSA87:
				privSize, pubSize, sigSize = mldsa.MLDSA87PrivateKeySize, mldsa.MLDSA87PublicKeySize, mldsa.MLDSA87SignatureSize
			}
			
			ux.Logger.PrintToUser("  Private Key:   %d bytes", privSize)
			ux.Logger.PrintToUser("  Public Key:    %d bytes", pubSize)
			ux.Logger.PrintToUser("  Signature:     %d bytes", sigSize)
			
			// Usage hint
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("💡 This key can be used for post-quantum secure signatures")
			ux.Logger.PrintToUser("   in smart contracts via precompile at address 0x011%s", string(mldsaLevel[0]))
			
			return nil
		},
	}
	
	cmd.Flags().StringVar(&mldsaLevel, "level", "", "Security level: 44, 65 (recommended), or 87")
	cmd.Flags().BoolVar(&mldsaBenchmark, "benchmark", false, "Run performance benchmark for all levels")
	
	return cmd
}

// MLDSAKeyFile represents the JSON structure for storing ML-DSA keys
type MLDSAKeyFile struct {
	Algorithm  string `json:"algorithm"`
	Name       string `json:"name"`
	Level      string `json:"level"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	CreatedAt  string `json:"createdAt"`
}

func promptMLDSALevel() (string, error) {
	levels := []string{
		"65 - Recommended (Level 3 security, 3.3 KB signatures)",
		"44 - Faster/Smaller (Level 2 security, 2.4 KB signatures)",
		"87 - Maximum Security (Level 5 security, 4.6 KB signatures)",
	}
	
	selected, err := prompts.PromptSelect("Select ML-DSA security level", levels)
	if err != nil {
		return "", err
	}
	
	// Extract level number from selection
	return strings.Split(selected, " ")[0], nil
}

func getSecurityLevel(level string) string {
	switch level {
	case "44":
		return "2 (~128-bit classical, ~64-bit quantum)"
	case "65":
		return "3 (~192-bit classical, ~96-bit quantum)"
	case "87":
		return "5 (~256-bit classical, ~128-bit quantum)"
	default:
		return "Unknown"
	}
}

func benchmarkMLDSA() error {
	fmt.Println("\n🚀 ML-DSA Performance Benchmark")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	message := []byte("Test message for benchmarking ML-DSA signatures")
	
	levels := []struct {
		name string
		mode mldsa.Mode
	}{
		{"ML-DSA-44", mldsa.MLDSA44},
		{"ML-DSA-65", mldsa.MLDSA65},
		{"ML-DSA-87", mldsa.MLDSA87},
	}
	
	for _, level := range levels {
		fmt.Printf("\n%s:\n", level.name)
		
		// Key generation
		start := time.Now()
		privKey, err := mldsa.GenerateKey(rand.Reader, level.mode)
		if err != nil {
			return err
		}
		keyGenTime := time.Since(start)
		
		// Signing
		start = time.Now()
		signature, err := privKey.Sign(rand.Reader, message, nil)
		if err != nil {
			return err
		}
		signTime := time.Since(start)
		
		// Verification
		start = time.Now()
		valid := privKey.PublicKey.Verify(message, signature)
		verifyTime := time.Since(start)
		
		if !valid {
			return fmt.Errorf("signature verification failed")
		}
		
		fmt.Printf("  Key Generation: %v\n", keyGenTime)
		fmt.Printf("  Sign:           %v\n", signTime)
		fmt.Printf("  Verify:         %v\n", verifyTime)
		fmt.Printf("  Signature Size: %d bytes\n", len(signature))
	}
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 Comparison with ECDSA:")
	fmt.Println("  ECDSA Sign:     ~200 μs")
	fmt.Println("  ECDSA Verify:   ~500 μs")
	fmt.Println("  ECDSA Sig Size: 65 bytes")
	
	return nil
}