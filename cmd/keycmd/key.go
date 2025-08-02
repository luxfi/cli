// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/geth/accounts/keystore"
	"github.com/luxfi/geth/crypto"
	"github.com/luxfi/cli/v2/v2/pkg/constants"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/crypto/validator"
	"github.com/spf13/cobra"
)

// NewCmd returns the key command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage cryptographic keys (validator, ethereum, etc)",
		Long: `The key command provides a unified interface for managing all types of keys
used in the Lux ecosystem including validator keys, ethereum keys, and more.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Args: cobra.NoArgs,
	}

	// Add subcommands
	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())

	return cmd
}

// newGenerateCmd creates the generate subcommand
func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate new keys",
		Long:  "Generate validator keys, ethereum keys, or other cryptographic keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newGenerateValidatorCmd())
	cmd.AddCommand(newGenerateEthereumCmd())

	return cmd
}

// newGenerateValidatorCmd creates the validator key generation command
func newGenerateValidatorCmd() *cobra.Command {
	var (
		network     string
		count       int
		outputPath  string
		seedPhrase  string
		startIndex  int
	)

	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Generate validator keys for network nodes",
		Long: `Generate validator keys including BLS keys, NodeID, and staking certificates.
These keys are used to run validator nodes on the Lux network.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateValidatorKeys(network, count, outputPath, seedPhrase, startIndex)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&network, "network", "local", "Network to generate keys for (mainnet, testnet, local)")
	cmd.Flags().IntVar(&count, "count", 1, "Number of validator keys to generate")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output path for keys (default: ~/.luxd/keys/<network>)")
	cmd.Flags().StringVar(&seedPhrase, "seed", "", "Seed phrase for deterministic generation (optional)")
	cmd.Flags().IntVar(&startIndex, "start-index", 0, "Starting index for deterministic generation")

	return cmd
}

// newGenerateEthereumCmd creates the ethereum key generation command
func newGenerateEthereumCmd() *cobra.Command {
	var (
		count      int
		outputPath string
		password   string
	)

	cmd := &cobra.Command{
		Use:   "ethereum",
		Short: "Generate Ethereum-compatible keys",
		Long:  "Generate Ethereum private keys and addresses for use with C-Chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateEthereumKeys(count, outputPath, password)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().IntVar(&count, "count", 1, "Number of keys to generate")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output path for keys (default: ~/.luxd/keys/ethereum)")
	cmd.Flags().StringVar(&password, "password", "", "Password for keystore encryption")

	return cmd
}

// newListCmd creates the list command
func newListCmd() *cobra.Command {
	var (
		network string
		keyType string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List existing keys",
		Long:  "Display information about existing keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listKeys(network, keyType)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&network, "network", "", "Network to list keys for (mainnet, testnet, local, or all)")
	cmd.Flags().StringVar(&keyType, "type", "all", "Type of keys to list (validator, ethereum, all)")

	return cmd
}

// newAddCmd creates the add command for importing keys
func newAddCmd() *cobra.Command {
	var (
		network    string
		keyType    string
		sourcePath string
		privateKey string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Import existing keys",
		Long:  "Import validator keys or ethereum keys from external sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return importKeys(network, keyType, sourcePath, privateKey)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&network, "network", "local", "Network to import keys for")
	cmd.Flags().StringVar(&keyType, "type", "validator", "Type of key to import (validator, ethereum)")
	cmd.Flags().StringVar(&sourcePath, "source", "", "Source file path for import")
	cmd.Flags().StringVar(&privateKey, "private-key", "", "Private key hex string (alternative to file)")

	return cmd
}

// generateValidatorKeys generates validator keys
func generateValidatorKeys(network string, count int, outputPath string, seedPhrase string, startIndex int) error {
	ux.Logger.PrintToUser("🔑 Generating Validator Keys")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
	ux.Logger.PrintToUser("Network: %s", network)
	ux.Logger.PrintToUser("Count: %d", count)

	// Determine output path
	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		outputPath = filepath.Join(homeDir, ".luxd", "keys", network)
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0700); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create key generator
	kg := validator.NewKeyGenerator()

	// Generate keys
	ux.Logger.PrintToUser("\n🔄 Generating keys...")
	var keys []*validator.ValidatorKeysWithTLS

	if seedPhrase != "" {
		// Deterministic generation
		for i := 0; i < count; i++ {
			keyInfo, err := kg.GenerateFromSeedWithTLS(seedPhrase, startIndex+i)
			if err != nil {
				return fmt.Errorf("failed to generate key %d: %w", i+1, err)
			}
			keys = append(keys, keyInfo)
		}
	} else {
		// Random generation
		keyInfos, err := kg.GenerateCompatibleKeys(count)
		if err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}
		keys = keyInfos
	}

	// Save keys
	for i, keyInfo := range keys {
		nodeDir := filepath.Join(outputPath, fmt.Sprintf("node%02d", i+1))
		if err := validator.SaveKeys(keyInfo, nodeDir); err != nil {
			return fmt.Errorf("failed to save key %d: %w", i+1, err)
		}
		ux.Logger.PrintToUser("✅ Generated keys for node%02d (NodeID: %s)", i+1, keyInfo.NodeID[:12]+"...")
	}

	// Save consolidated info
	var validatorInfos []*validator.ValidatorInfo
	for _, key := range keys {
		// Generate a placeholder ETH address (in real usage, this would come from C-Chain)
		ethKey, _ := crypto.GenerateKey()
		ethAddress := crypto.PubkeyToAddress(ethKey.PublicKey).Hex()

		info := validator.GenerateValidatorConfig(
			key.ValidatorKeys,
			ethAddress,
			constants.DefaultStakeAmount,
			10000, // 1% delegation fee
		)
		validatorInfos = append(validatorInfos, info)
	}

	validatorsFile := filepath.Join(outputPath, "validators.json")
	if err := validator.SaveValidatorConfigs(validatorInfos, validatorsFile); err != nil {
		return fmt.Errorf("failed to save validator configs: %w", err)
	}

	ux.Logger.PrintToUser("\n✨ Successfully generated %d validator keys!", count)
	ux.Logger.PrintToUser("\n📁 Key Locations:")
	ux.Logger.PrintToUser("   Output directory: %s", outputPath)
	ux.Logger.PrintToUser("   Validator configs: %s", validatorsFile)

	return nil
}

// generateEthereumKeys generates ethereum keys
func generateEthereumKeys(count int, outputPath string, password string) error {
	ux.Logger.PrintToUser("🔑 Generating Ethereum Keys")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))

	// Determine output path
	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		outputPath = filepath.Join(homeDir, ".luxd", "keys", "ethereum")
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0700); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create keystore
	ks := keystore.NewKeyStore(outputPath, keystore.StandardScryptN, keystore.StandardScryptP)

	// Generate keys
	var accounts []map[string]string
	for i := 0; i < count; i++ {
		account, err := ks.NewAccount(password)
		if err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}

		accounts = append(accounts, map[string]string{
			"address": account.Address.Hex(),
			"file":    account.URL.Path,
		})

		ux.Logger.PrintToUser("✅ Generated account %d: %s", i+1, account.Address.Hex())
	}

	// Save account summary
	summaryFile := filepath.Join(outputPath, "accounts.json")
	summaryData, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(summaryFile, summaryData, 0644); err != nil {
		return err
	}

	ux.Logger.PrintToUser("\n✨ Successfully generated %d Ethereum keys!", count)
	ux.Logger.PrintToUser("\n📁 Key Locations:")
	ux.Logger.PrintToUser("   Keystore directory: %s", outputPath)
	ux.Logger.PrintToUser("   Account summary: %s", summaryFile)

	return nil
}

// listKeys lists existing keys
func listKeys(network, keyType string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("📋 Listing Keys")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))

	luxdHome := filepath.Join(homeDir, ".luxd", "keys")

	// List validator keys
	if keyType == "all" || keyType == "validator" {
		ux.Logger.PrintToUser("\n🔐 Validator Keys:")
		networks := []string{"mainnet", "testnet", "local"}
		if network != "" && network != "all" {
			networks = []string{network}
		}

		for _, net := range networks {
			validatorsFile := filepath.Join(luxdHome, net, "validators.json")
			if _, err := os.Stat(validatorsFile); os.IsNotExist(err) {
				continue
			}

			data, err := os.ReadFile(validatorsFile)
			if err != nil {
				continue
			}

			var validators []*validator.ValidatorInfo
			if err := json.Unmarshal(data, &validators); err != nil {
				continue
			}

			ux.Logger.PrintToUser("\n   %s: %d validators", strings.Title(net), len(validators))
			for i, val := range validators {
				if i >= 3 {
					ux.Logger.PrintToUser("      ... and %d more", len(validators)-3)
					break
				}
				ux.Logger.PrintToUser("      • %s", val.NodeID)
			}
		}
	}

	// List ethereum keys
	if keyType == "all" || keyType == "ethereum" {
		ux.Logger.PrintToUser("\n💎 Ethereum Keys:")
		ethPath := filepath.Join(luxdHome, "ethereum")
		
		if info, err := os.Stat(ethPath); err == nil && info.IsDir() {
			files, err := os.ReadDir(ethPath)
			if err == nil {
				ethKeyCount := 0
				for _, file := range files {
					if strings.HasPrefix(file.Name(), "UTC--") {
						ethKeyCount++
					}
				}
				if ethKeyCount > 0 {
					ux.Logger.PrintToUser("   Found %d Ethereum keys in keystore", ethKeyCount)
				}
			}
		}
	}

	return nil
}

// importKeys imports existing keys
func importKeys(network, keyType, sourcePath, privateKey string) error {
	ux.Logger.PrintToUser("📥 Importing Keys")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
	ux.Logger.PrintToUser("Network: %s", network)
	ux.Logger.PrintToUser("Type: %s", keyType)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	outputPath := filepath.Join(homeDir, ".luxd", "keys", network)

	switch keyType {
	case "validator":
		if privateKey != "" {
			// Import from private key
			kg := validator.NewKeyGenerator()
			keyInfo, err := kg.GenerateFromPrivateKey(privateKey)
			if err != nil {
				return fmt.Errorf("failed to generate from private key: %w", err)
			}

			nodeDir := filepath.Join(outputPath, "imported-"+time.Now().Format("20060102-150405"))
			if err := validator.SaveKeys(keyInfo, nodeDir); err != nil {
				return fmt.Errorf("failed to save keys: %w", err)
			}

			ux.Logger.PrintToUser("✅ Imported validator key")
			ux.Logger.PrintToUser("   NodeID: %s", keyInfo.NodeID)
			ux.Logger.PrintToUser("   Location: %s", nodeDir)
		} else if sourcePath != "" {
			// Import from file
			return fmt.Errorf("file import not yet implemented")
		} else {
			return fmt.Errorf("must provide either --source or --private-key")
		}

	case "ethereum":
		return fmt.Errorf("ethereum key import not yet implemented")

	default:
		return fmt.Errorf("unknown key type: %s", keyType)
	}

	return nil
}