// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// Post-quantum key creation command

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
	"github.com/luxfi/crypto/slhdsa"
	"github.com/luxfi/crypto/mlkem"
)

var (
	algorithm   string
	outputPath  string
	showSizes   bool
	benchmark   bool
)

func newCreatePostQuantumCmd(app *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pq [keyName]",
		Short: "Create a new post-quantum cryptographic key",
		Long: `Create a new post-quantum cryptographic key using NIST-approved algorithms.

Available algorithms:
  - ml-dsa-44:  ML-DSA (Dilithium) Level 2 - Balanced security/performance
  - ml-dsa-65:  ML-DSA (Dilithium) Level 3 - Recommended for most uses
  - ml-dsa-87:  ML-DSA (Dilithium) Level 5 - Maximum security
  - slh-dsa-128f: SLH-DSA (SPHINCS+) 128-bit fast - Smaller computation, larger signatures
  - slh-dsa-128s: SLH-DSA (SPHINCS+) 128-bit small - Smaller signatures, more computation
  - slh-dsa-256f: SLH-DSA (SPHINCS+) 256-bit fast - Maximum security, large signatures
  - ml-kem-768: ML-KEM (Kyber) Level 3 - For key encapsulation (not signatures)

Example:
  lux key create-pq mykey --algorithm ml-dsa-65
  lux key create-pq mykey --algorithm slh-dsa-128f --output ~/keys/
  lux key create-pq --show-sizes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showSizes {
				return showAlgorithmSizes()
			}
			
			if benchmark {
				return runBenchmark()
			}
			
			keyName := ""
			if len(args) > 0 {
				keyName = args[0]
			} else {
				name, err := prompts.PromptString("Enter key name", "my-pq-key", prompts.ValidateNotEmpty)
				if err != nil {
					return err
				}
				keyName = name
			}
			
			// Validate key name
			if err := key.ValidateKeyName(keyName); err != nil {
				return err
			}
			
			// Select algorithm if not provided
			if algorithm == "" {
				selectedAlg, err := promptAlgorithm()
				if err != nil {
					return err
				}
				algorithm = selectedAlg
			}
			
			// Generate the key
			ux.Logger.PrintToUser("Generating %s key '%s'...", strings.ToUpper(algorithm), keyName)
			
			privKey, pubKey, err := generatePostQuantumKey(algorithm)
			if err != nil {
				return fmt.Errorf("failed to generate key: %w", err)
			}
			
			// Determine output path
			keyPath := app.GetKeyPath(keyName)
			if outputPath != "" {
				keyPath = filepath.Join(outputPath, keyName+".pq.key")
			}
			
			// Save key to file
			keyData := PostQuantumKeyFile{
				Algorithm:  algorithm,
				Name:       keyName,
				PrivateKey: hex.EncodeToString(privKey),
				PublicKey:  hex.EncodeToString(pubKey),
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
			ux.Logger.PrintToUser("Post-Quantum Key Created Successfully!")
			ux.Logger.PrintToUser("Algorithm: %s", strings.ToUpper(algorithm))
			ux.Logger.PrintToUser("Key Name: %s", keyName)
			ux.Logger.PrintToUser("Saved to: %s", keyPath)
			
			// Show sizes
			privSize, pubSize, sigSize := getAlgorithmSizes(algorithm)
			ux.Logger.PrintToUser("\nKey Sizes:")
			ux.Logger.PrintToUser("  Private Key: %d bytes", privSize)
			ux.Logger.PrintToUser("  Public Key:  %d bytes", pubSize)
			ux.Logger.PrintToUser("  Signature:   %d bytes", sigSize)
			
			// Security level
			level := getSecurityLevel(algorithm)
			ux.Logger.PrintToUser("  Security:    %s", level)
			
			// Warning for large keys
			if privSize > 2000 || sigSize > 5000 {
				ux.Logger.PrintToUser("\n⚠️  Note: This algorithm uses large keys/signatures.")
				ux.Logger.PrintToUser("   Consider ML-DSA for smaller sizes if appropriate.")
			}
			
			return nil
		},
	}
	
	cmd.Flags().StringVar(&algorithm, "algorithm", "", "Post-quantum algorithm to use")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output directory for key file")
	cmd.Flags().BoolVar(&showSizes, "show-sizes", false, "Show algorithm sizes comparison")
	cmd.Flags().BoolVar(&benchmark, "benchmark", false, "Run performance benchmark")
	
	return cmd
}

// PostQuantumKeyFile represents the JSON structure for storing PQ keys
type PostQuantumKeyFile struct {
	Algorithm  string `json:"algorithm"`
	Name       string `json:"name"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address,omitempty"`
}

func generatePostQuantumKey(algorithm string) (privKey, pubKey []byte, err error) {
	switch strings.ToLower(algorithm) {
	case "ml-dsa-44", "mldsa44":
		priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
		if err != nil {
			return nil, nil, err
		}
		return priv.Bytes(), priv.PublicKey.Bytes(), nil
		
	case "ml-dsa-65", "mldsa65":
		priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
		if err != nil {
			return nil, nil, err
		}
		return priv.Bytes(), priv.PublicKey.Bytes(), nil
		
	case "ml-dsa-87", "mldsa87":
		priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA87)
		if err != nil {
			return nil, nil, err
		}
		return priv.Bytes(), priv.PublicKey.Bytes(), nil
		
	case "slh-dsa-128f", "slhdsa128f":
		priv, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA128f)
		if err != nil {
			return nil, nil, err
		}
		return priv.Bytes(), priv.PublicKey.Bytes(), nil
		
	case "slh-dsa-128s", "slhdsa128s":
		priv, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA128s)
		if err != nil {
			return nil, nil, err
		}
		return priv.Bytes(), priv.PublicKey.Bytes(), nil
		
	case "slh-dsa-256f", "slhdsa256f":
		priv, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA256f)
		if err != nil {
			return nil, nil, err
		}
		return priv.Bytes(), priv.PublicKey.Bytes(), nil
		
	case "ml-kem-768", "mlkem768":
		priv, err := mlkem.GenerateKeyPair(rand.Reader, mlkem.MLKEM768)
		if err != nil {
			return nil, nil, err
		}
		return priv.Bytes(), priv.PublicKey.Bytes(), nil
		
	default:
		return nil, nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

func promptAlgorithm() (string, error) {
	algorithms := []string{
		"ml-dsa-65 (Recommended - balanced)",
		"ml-dsa-44 (Smaller, faster)",
		"ml-dsa-87 (Maximum security)",
		"slh-dsa-128f (Hash-based, large signatures)",
		"ml-kem-768 (Key encapsulation only)",
	}
	
	selected, err := prompts.PromptSelect("Select post-quantum algorithm", algorithms)
	if err != nil {
		return "", err
	}
	
	// Extract algorithm name from selection
	parts := strings.Split(selected, " ")
	return parts[0], nil
}

func getAlgorithmSizes(algorithm string) (privSize, pubSize, sigSize int) {
	switch strings.ToLower(algorithm) {
	case "ml-dsa-44", "mldsa44":
		return mldsa.MLDSA44PrivateKeySize, mldsa.MLDSA44PublicKeySize, mldsa.MLDSA44SignatureSize
	case "ml-dsa-65", "mldsa65":
		return mldsa.MLDSA65PrivateKeySize, mldsa.MLDSA65PublicKeySize, mldsa.MLDSA65SignatureSize
	case "ml-dsa-87", "mldsa87":
		return mldsa.MLDSA87PrivateKeySize, mldsa.MLDSA87PublicKeySize, mldsa.MLDSA87SignatureSize
	case "slh-dsa-128f", "slhdsa128f":
		return slhdsa.SLHDSA128fPrivateKeySize, slhdsa.SLHDSA128fPublicKeySize, slhdsa.SLHDSA128fSignatureSize
	case "slh-dsa-128s", "slhdsa128s":
		return slhdsa.SLHDSA128sPrivateKeySize, slhdsa.SLHDSA128sPublicKeySize, slhdsa.SLHDSA128sSignatureSize
	case "slh-dsa-256f", "slhdsa256f":
		return slhdsa.SLHDSA256fPrivateKeySize, slhdsa.SLHDSA256fPublicKeySize, slhdsa.SLHDSA256fSignatureSize
	case "ml-kem-768", "mlkem768":
		return mlkem.MLKEM768PrivateKeySize, mlkem.MLKEM768PublicKeySize, mlkem.MLKEM768CiphertextSize
	default:
		return 0, 0, 0
	}
}

func getSecurityLevel(algorithm string) string {
	switch strings.ToLower(algorithm) {
	case "ml-dsa-44", "mldsa44":
		return "NIST Level 2 (~128-bit)"
	case "ml-dsa-65", "mldsa65":
		return "NIST Level 3 (~192-bit)"
	case "ml-dsa-87", "mldsa87":
		return "NIST Level 5 (~256-bit)"
	case "slh-dsa-128f", "slhdsa128f", "slh-dsa-128s", "slhdsa128s":
		return "NIST Level 1 (~128-bit)"
	case "slh-dsa-256f", "slhdsa256f", "slh-dsa-256s", "slhdsa256s":
		return "NIST Level 5 (~256-bit)"
	case "ml-kem-768", "mlkem768":
		return "NIST Level 3 (~192-bit)"
	default:
		return "Unknown"
	}
}

func showAlgorithmSizes() error {
	fmt.Println("\nPost-Quantum Algorithm Comparison:")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "Algorithm", "Priv Key", "Pub Key", "Signature", "Security")
	fmt.Println("-" + strings.Repeat("-", 70))
	
	// Traditional
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "ECDSA", "32 B", "64 B", "65 B", "~128-bit")
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "BLS12-381", "32 B", "48 B", "96 B", "~128-bit")
	fmt.Println("-" + strings.Repeat("-", 70))
	
	// ML-DSA
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "ML-DSA-44", "2.5 KB", "1.3 KB", "2.4 KB", "Level 2")
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "ML-DSA-65", "4.0 KB", "2.0 KB", "3.3 KB", "Level 3")
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "ML-DSA-87", "4.9 KB", "2.6 KB", "4.6 KB", "Level 5")
	
	// SLH-DSA
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "SLH-DSA-128s", "64 B", "32 B", "7.9 KB", "Level 1")
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "SLH-DSA-128f", "64 B", "32 B", "17.1 KB", "Level 1")
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "SLH-DSA-256f", "128 B", "64 B", "49.9 KB", "Level 5")
	
	// ML-KEM
	fmt.Printf("%-15s %10s %10s %10s %15s\n", "ML-KEM-768", "2.4 KB", "1.2 KB", "1.1 KB*", "Level 3")
	
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("* ML-KEM produces ciphertext, not signatures")
	fmt.Println("\nRecommendations:")
	fmt.Println("  - General use: ML-DSA-65 (balanced size and security)")
	fmt.Println("  - High security: ML-DSA-87 or SLH-DSA-256f")
	fmt.Println("  - Space-constrained: ML-DSA-44")
	fmt.Println("  - Stateless hash-based: SLH-DSA (larger but different security model)")
	
	return nil
}

func runBenchmark() error {
	fmt.Println("\nRunning Post-Quantum Performance Benchmark...")
	fmt.Println("This may take a few moments...\n")
	
	// Benchmark each algorithm
	algorithms := []string{"ml-dsa-44", "ml-dsa-65", "ml-dsa-87", "slh-dsa-128f"}
	
	for _, alg := range algorithms {
		fmt.Printf("Benchmarking %s...\n", strings.ToUpper(alg))
		
		// Generate key and measure time
		start := time.Now()
		privKey, pubKey, err := generatePostQuantumKey(alg)
		if err != nil {
			return err
		}
		keyGenTime := time.Since(start)
		
		// Create a test message
		message := []byte("Test message for benchmarking post-quantum signatures")
		
		// Sign and measure time
		start = time.Now()
		var signature []byte
		
		switch alg {
		case "ml-dsa-44":
			priv, _ := mldsa.PrivateKeyFromBytes(privKey, mldsa.MLDSA44)
			signature, _ = priv.Sign(rand.Reader, message, nil)
		case "ml-dsa-65":
			priv, _ := mldsa.PrivateKeyFromBytes(privKey, mldsa.MLDSA65)
			signature, _ = priv.Sign(rand.Reader, message, nil)
		case "ml-dsa-87":
			priv, _ := mldsa.PrivateKeyFromBytes(privKey, mldsa.MLDSA87)
			signature, _ = priv.Sign(rand.Reader, message, nil)
		case "slh-dsa-128f":
			priv, _ := slhdsa.PrivateKeyFromBytes(privKey, slhdsa.SLHDSA128f)
			signature, _ = priv.Sign(rand.Reader, message, nil)
		}
		signTime := time.Since(start)
		
		// Verify and measure time
		start = time.Now()
		switch alg {
		case "ml-dsa-44":
			pub, _ := mldsa.PublicKeyFromBytes(pubKey, mldsa.MLDSA44)
			pub.Verify(message, signature)
		case "ml-dsa-65":
			pub, _ := mldsa.PublicKeyFromBytes(pubKey, mldsa.MLDSA65)
			pub.Verify(message, signature)
		case "ml-dsa-87":
			pub, _ := mldsa.PublicKeyFromBytes(pubKey, mldsa.MLDSA87)
			pub.Verify(message, signature)
		case "slh-dsa-128f":
			pub, _ := slhdsa.PublicKeyFromBytes(pubKey, slhdsa.SLHDSA128f)
			pub.Verify(message, signature)
		}
		verifyTime := time.Since(start)
		
		fmt.Printf("  Key Gen:  %v\n", keyGenTime)
		fmt.Printf("  Sign:     %v\n", signTime)
		fmt.Printf("  Verify:   %v\n", verifyTime)
		fmt.Println()
	}
	
	return nil
}