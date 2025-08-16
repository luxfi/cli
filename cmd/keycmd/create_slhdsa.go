// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// SLH-DSA (SPHINCS+) key creation command

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
	"github.com/luxfi/crypto/slhdsa"
)

var (
	slhdsaVariant string
	slhdsaCompare bool
)

func newCreateSLHDSACmd(app *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-slhdsa [keyName]",
		Short: "Create a new SLH-DSA (SPHINCS+) hash-based signature key",
		Long: `Create a new SLH-DSA (Stateless Hash-based Digital Signature Algorithm) key.
SLH-DSA is NIST's standardized hash-based signature scheme (FIPS 205).

Variants:
  128s - Level 1, small signatures (7.9 KB), slower
  128f - Level 1, fast signing, large signatures (17.1 KB)
  192s - Level 3, small signatures (16.2 KB), slower
  192f - Level 3, fast signing, large signatures (35.7 KB)
  256s - Level 5, small signatures (29.8 KB), slower
  256f - Level 5, fast signing, large signatures (49.9 KB)

Example:
  lux key create-slhdsa alice --variant 128f
  lux key create-slhdsa bob --variant 256s --compare`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if slhdsaCompare {
				return compareSLHDSAVariants()
			}
			
			keyName := ""
			if len(args) > 0 {
				keyName = args[0]
			} else {
				name, err := prompts.PromptString("Enter key name", "my-slhdsa-key", prompts.ValidateNotEmpty)
				if err != nil {
					return err
				}
				keyName = name
			}
			
			// Validate key name
			if err := key.ValidateKeyName(keyName); err != nil {
				return err
			}
			
			// Select variant if not provided
			if slhdsaVariant == "" {
				variant, err := promptSLHDSAVariant()
				if err != nil {
					return err
				}
				slhdsaVariant = variant
			}
			
			// Parse variant
			var mode slhdsa.Mode
			switch strings.ToLower(slhdsaVariant) {
			case "128s":
				mode = slhdsa.SLHDSA128s
			case "128f":
				mode = slhdsa.SLHDSA128f
			case "192s":
				mode = slhdsa.SLHDSA192s
			case "192f":
				mode = slhdsa.SLHDSA192f
			case "256s":
				mode = slhdsa.SLHDSA256s
			case "256f":
				mode = slhdsa.SLHDSA256f
			default:
				return fmt.Errorf("invalid SLH-DSA variant: %s", slhdsaVariant)
			}
			
			// Generate the key
			ux.Logger.PrintToUser("Generating SLH-DSA-%s key '%s'...", strings.ToUpper(slhdsaVariant), keyName)
			
			start := time.Now()
			privKey, err := slhdsa.GenerateKey(rand.Reader, mode)
			if err != nil {
				return fmt.Errorf("failed to generate SLH-DSA key: %w", err)
			}
			elapsed := time.Since(start)
			
			// Determine output path
			keyPath := app.GetKeyPath(keyName)
			keyPath = strings.Replace(keyPath, ".pk", ".slhdsa", 1)
			
			// Save key to file
			keyData := SLHDSAKeyFile{
				Algorithm:  fmt.Sprintf("SLH-DSA-%s", strings.ToUpper(slhdsaVariant)),
				Name:       keyName,
				Variant:    slhdsaVariant,
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
			
			// Get sizes
			privSize, pubSize, sigSize := getSLHDSASizes(mode)
			
			// Display key information
			ux.Logger.PrintToUser("✅ SLH-DSA Key Created Successfully!")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Algorithm:     SLH-DSA-%s", strings.ToUpper(slhdsaVariant))
			ux.Logger.PrintToUser("Security:      %s", getSLHDSASecurityLevel(slhdsaVariant))
			ux.Logger.PrintToUser("Key Name:      %s", keyName)
			ux.Logger.PrintToUser("Generated in:  %v", elapsed)
			ux.Logger.PrintToUser("Saved to:      %s", keyPath)
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Key Sizes:")
			ux.Logger.PrintToUser("  Private Key:   %d bytes", privSize)
			ux.Logger.PrintToUser("  Public Key:    %d bytes", pubSize)
			ux.Logger.PrintToUser("  Signature:     %d KB", sigSize/1024)
			
			// Variant explanation
			ux.Logger.PrintToUser("")
			if strings.HasSuffix(slhdsaVariant, "f") {
				ux.Logger.PrintToUser("⚡ Fast variant: Optimized for signing speed")
				ux.Logger.PrintToUser("   Trade-off: Larger signatures (%d KB)", sigSize/1024)
			} else {
				ux.Logger.PrintToUser("📦 Small variant: Optimized for signature size")
				ux.Logger.PrintToUser("   Trade-off: Slower signing operations")
			}
			
			// Usage hint
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("🔒 Hash-based signatures are quantum-resistant")
			ux.Logger.PrintToUser("   and don't rely on number theory assumptions")
			
			return nil
		},
	}
	
	cmd.Flags().StringVar(&slhdsaVariant, "variant", "", "SLH-DSA variant (128s/f, 192s/f, 256s/f)")
	cmd.Flags().BoolVar(&slhdsaCompare, "compare", false, "Compare all SLH-DSA variants")
	
	return cmd
}

// SLHDSAKeyFile represents the JSON structure for storing SLH-DSA keys
type SLHDSAKeyFile struct {
	Algorithm  string `json:"algorithm"`
	Name       string `json:"name"`
	Variant    string `json:"variant"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	CreatedAt  string `json:"createdAt"`
}

func promptSLHDSAVariant() (string, error) {
	variants := []string{
		"128f - Fast signing, 17 KB signatures (Level 1)",
		"128s - Small 8 KB signatures, slower (Level 1)",
		"192f - Fast signing, 36 KB signatures (Level 3)",
		"192s - Small 16 KB signatures, slower (Level 3)",
		"256f - Fast signing, 50 KB signatures (Level 5)",
		"256s - Small 30 KB signatures, slower (Level 5)",
	}
	
	selected, err := prompts.PromptSelect("Select SLH-DSA variant", variants)
	if err != nil {
		return "", err
	}
	
	// Extract variant code from selection
	return strings.Split(selected, " ")[0], nil
}

func getSLHDSASizes(mode slhdsa.Mode) (privSize, pubSize, sigSize int) {
	switch mode {
	case slhdsa.SLHDSA128s:
		return slhdsa.SLHDSA128sPrivateKeySize, slhdsa.SLHDSA128sPublicKeySize, slhdsa.SLHDSA128sSignatureSize
	case slhdsa.SLHDSA128f:
		return slhdsa.SLHDSA128fPrivateKeySize, slhdsa.SLHDSA128fPublicKeySize, slhdsa.SLHDSA128fSignatureSize
	case slhdsa.SLHDSA192s:
		return slhdsa.SLHDSA192sPrivateKeySize, slhdsa.SLHDSA192sPublicKeySize, slhdsa.SLHDSA192sSignatureSize
	case slhdsa.SLHDSA192f:
		return slhdsa.SLHDSA192fPrivateKeySize, slhdsa.SLHDSA192fPublicKeySize, slhdsa.SLHDSA192fSignatureSize
	case slhdsa.SLHDSA256s:
		return slhdsa.SLHDSA256sPrivateKeySize, slhdsa.SLHDSA256sPublicKeySize, slhdsa.SLHDSA256sSignatureSize
	case slhdsa.SLHDSA256f:
		return slhdsa.SLHDSA256fPrivateKeySize, slhdsa.SLHDSA256fPublicKeySize, slhdsa.SLHDSA256fSignatureSize
	default:
		return 0, 0, 0
	}
}

func getSLHDSASecurityLevel(variant string) string {
	switch variant {
	case "128s", "128f":
		return "NIST Level 1 (~128-bit classical)"
	case "192s", "192f":
		return "NIST Level 3 (~192-bit classical)"
	case "256s", "256f":
		return "NIST Level 5 (~256-bit classical)"
	default:
		return "Unknown"
	}
}

func compareSLHDSAVariants() error {
	fmt.Println("\n📊 SLH-DSA Variants Comparison")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("%-10s %8s %8s %10s %12s %10s\n", "Variant", "Priv Key", "Pub Key", "Signature", "Speed", "Security")
	fmt.Println("-" + strings.Repeat("-", 70))
	
	variants := []struct {
		name string
		mode slhdsa.Mode
		speed string
	}{
		{"128s", slhdsa.SLHDSA128s, "Slow"},
		{"128f", slhdsa.SLHDSA128f, "Fast"},
		{"192s", slhdsa.SLHDSA192s, "Slow"},
		{"192f", slhdsa.SLHDSA192f, "Fast"},
		{"256s", slhdsa.SLHDSA256s, "Slow"},
		{"256f", slhdsa.SLHDSA256f, "Fast"},
	}
	
	for _, v := range variants {
		privSize, pubSize, sigSize := getSLHDSASizes(v.mode)
		level := getSLHDSASecurityLevel(v.name)
		
		fmt.Printf("%-10s %8d B %8d B %10s %12s %10s\n",
			"SLH-DSA-"+v.name,
			privSize,
			pubSize,
			fmt.Sprintf("%.1f KB", float64(sigSize)/1024),
			v.speed,
			strings.Split(level, " ")[2], // Extract just "Level X"
		)
	}
	
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("\n📝 Guidelines:")
	fmt.Println("  • 's' variants: Choose when signature size matters (storage, bandwidth)")
	fmt.Println("  • 'f' variants: Choose when signing speed matters (high throughput)")
	fmt.Println("  • Level 1: Basic quantum resistance")
	fmt.Println("  • Level 3: Recommended for most applications")
	fmt.Println("  • Level 5: Maximum security for critical systems")
	
	return nil
}