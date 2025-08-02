// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/geth/common/hexutil"
	"github.com/luxfi/geth/crypto"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/v2/v2/staking"
	"github.com/spf13/cobra"
)

// KeyInfo represents a generated key set
type KeyInfo struct {
	NodeID       string    `json:"nodeId"`
	StakingCert  string    `json:"stakingCert"`
	StakingKey   string    `json:"stakingKey"`
	BLSKey       string    `json:"blsKey"`
	BLSSignature string    `json:"blsSignature"`
	ETHAddress   string    `json:"ethAddress"`
	ETHPrivKey   string    `json:"ethPrivKey,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	Network      string    `json:"network"`
}

// ValidatorKeySet represents a complete set of validator keys
type ValidatorKeySet struct {
	Keys       []KeyInfo `json:"keys"`
	Network    string    `json:"network"`
	CreatedAt  time.Time `json:"createdAt"`
	TotalNodes int       `json:"totalNodes"`
}

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage validator and staking keys",
		Long:  `Generate, import, and manage validator keys for mainnet, testnet, and local networks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	
	// Add subcommands
	cmd.AddCommand(newGenerateKeysCmd())
	cmd.AddCommand(newListKeysCmd())
	cmd.AddCommand(newImportKeysCmd())
	
	return cmd
}

func newGenerateKeysCmd() *cobra.Command {
	var (
		network     string
		count       int
		outputPath  string
		forceCreate bool
	)
	
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate validator keys for network nodes",
		Long: `Generate a complete set of validator keys including staking certificates,
BLS keys, and Ethereum-compatible addresses for network validators.

For mainnet and testnet, this generates 21 unique validator keys.
For local network, this generates keys with pre-funded test accounts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateValidatorKeys(network, count, outputPath, forceCreate)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}
	
	cmd.Flags().StringVar(&network, "network", "local", "Network to generate keys for (mainnet, testnet, local)")
	cmd.Flags().IntVar(&count, "count", 0, "Number of keys to generate (default: 21 for mainnet/testnet, 5 for local)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output path for keys (default: ~/.luxd/keys/<network>)")
	cmd.Flags().BoolVar(&forceCreate, "force", false, "Force overwrite existing keys")
	
	return cmd
}

func newListKeysCmd() *cobra.Command {
	var network string
	
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List existing validator keys",
		Long:  `Display information about existing validator keys for a network.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listValidatorKeys(network)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}
	
	cmd.Flags().StringVar(&network, "network", "", "Network to list keys for (mainnet, testnet, local, or all)")
	
	return cmd
}

func newImportKeysCmd() *cobra.Command {
	var (
		network    string
		sourcePath string
	)
	
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import existing validator keys",
		Long:  `Import validator keys from an external source.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return importValidatorKeys(network, sourcePath)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}
	
	cmd.Flags().StringVar(&network, "network", "", "Network to import keys for")
	cmd.Flags().StringVar(&sourcePath, "source", "", "Source path for keys to import")
	cmd.MarkFlagRequired("source")
	
	return cmd
}

func generateValidatorKeys(network string, count int, outputPath string, forceCreate bool) error {
	// Determine key count based on network
	if count == 0 {
		switch network {
		case "mainnet", "testnet":
			count = 21
		case "local":
			count = 5
		default:
			return fmt.Errorf("invalid network: %s", network)
		}
	}
	
	ux.Logger.PrintToUser("🔑 Generating Validator Keys for %s", strings.Title(network))
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 60))
	ux.Logger.PrintToUser("Network: %s", network)
	ux.Logger.PrintToUser("Count: %d validator keys", count)
	
	// Determine output path
	if outputPath == "" {
		luxdHome, err := GetLuxdHome()
		if err != nil {
			return err
		}
		outputPath = filepath.Join(luxdHome, "keys", network)
	}
	
	// Check if keys already exist
	keysFile := filepath.Join(outputPath, "validators.json")
	if _, err := os.Stat(keysFile); err == nil && !forceCreate {
		ux.Logger.PrintToUser("\n⚠️  Keys already exist at %s", keysFile)
		ux.Logger.PrintToUser("Use --force to overwrite existing keys")
		return nil
	}
	
	// Create output directory
	if err := os.MkdirAll(outputPath, 0700); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Generate keys
	ux.Logger.PrintToUser("\n🔄 Generating keys...")
	
	keySet := ValidatorKeySet{
		Keys:       make([]KeyInfo, 0, count),
		Network:    network,
		CreatedAt:  time.Now(),
		TotalNodes: count,
	}
	
	for i := 0; i < count; i++ {
		keyInfo, err := generateValidatorKeySet(i+1, network)
		if err != nil {
			return fmt.Errorf("failed to generate key %d: %w", i+1, err)
		}
		
		// Save individual key files
		nodeDir := filepath.Join(outputPath, fmt.Sprintf("node%02d", i+1))
		if err := os.MkdirAll(nodeDir, 0700); err != nil {
			return err
		}
		
		// Save staking certificate and key
		certPath := filepath.Join(nodeDir, "staker.crt")
		keyPath := filepath.Join(nodeDir, "staker.key")
		
		if err := os.WriteFile(certPath, []byte(keyInfo.StakingCert), 0600); err != nil {
			return err
		}
		if err := os.WriteFile(keyPath, []byte(keyInfo.StakingKey), 0600); err != nil {
			return err
		}
		
		// Save BLS key
		blsPath := filepath.Join(nodeDir, "bls.key")
		if err := os.WriteFile(blsPath, []byte(keyInfo.BLSKey), 0600); err != nil {
			return err
		}
		
		// For local network, save ETH private key
		if network == "local" && keyInfo.ETHPrivKey != "" {
			ethKeyPath := filepath.Join(nodeDir, "eth.key")
			if err := os.WriteFile(ethKeyPath, []byte(keyInfo.ETHPrivKey), 0600); err != nil {
				return err
			}
		}
		
		keySet.Keys = append(keySet.Keys, *keyInfo)
		
		ux.Logger.PrintToUser("✅ Generated keys for node%02d (NodeID: %s)", i+1, keyInfo.NodeID[:12]+"...")
	}
	
	// Save consolidated validator file
	validatorsData, err := json.MarshalIndent(keySet, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(keysFile, validatorsData, 0600); err != nil {
		return err
	}
	
	// Display summary
	ux.Logger.PrintToUser("\n✨ Successfully generated %d validator keys!", count)
	ux.Logger.PrintToUser("\n📁 Key Locations:")
	ux.Logger.PrintToUser("   Master file: %s", keysFile)
	ux.Logger.PrintToUser("   Individual keys: %s/node*/", outputPath)
	
	if network == "local" {
		ux.Logger.PrintToUser("\n💰 Pre-funded Accounts (Local Network):")
		for i, key := range keySet.Keys[:3] { // Show first 3
			ux.Logger.PrintToUser("   Account %d: %s (1M LUX)", i+1, key.ETHAddress)
		}
	}
	
	// Show example usage
	ux.Logger.PrintToUser("\n📋 Next Steps:")
	ux.Logger.PrintToUser("1. Update genesis with new validators:")
	ux.Logger.PrintToUser("   lux network update-genesis --network %s", network)
	ux.Logger.PrintToUser("2. Import historic blockchain data:")
	ux.Logger.PrintToUser("   lux network import --network %s --source /path/to/data", network)
	ux.Logger.PrintToUser("3. Start the network:")
	ux.Logger.PrintToUser("   lux network start %s", network)
	
	return nil
}

func generateValidatorKeySet(index int, network string) (*KeyInfo, error) {
	// Generate staking certificate and key
	cert, key, err := staking.NewCertAndKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate staking cert: %w", err)
	}
	
	// Get NodeID from certificate
	parsedCert, err := staking.ParseCertificate(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	nodeID := ids.NodeIDFromCert((*ids.Certificate)(parsedCert))
	
	// Generate BLS key
	blsSecretKey, err := bls.NewSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate BLS key: %w", err)
	}
	
	blsPubKey := blsSecretKey.PublicKey()
	blsPubKeyBytes := bls.PublicKeyToCompressedBytes(blsPubKey)
	blsSig := blsSecretKey.SignProofOfPossession(blsPubKeyBytes)
	
	// Generate Ethereum-compatible address
	ethKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ETH key: %w", err)
	}
	
	ethAddress := crypto.PubkeyToAddress(ethKey.PublicKey)
	
	keyInfo := &KeyInfo{
		NodeID:       nodeID.String(),
		StakingCert:  string(cert),
		StakingKey:   string(key),
		BLSKey:       hexutil.Encode(bls.SecretKeyToBytes(blsSecretKey)),
		BLSSignature: hexutil.Encode(bls.SignatureToBytes(blsSig)),
		ETHAddress:   ethAddress.Hex(),
		CreatedAt:    time.Now(),
		Network:      network,
	}
	
	// For local network, include private key for pre-funding
	if network == "local" {
		keyInfo.ETHPrivKey = hexutil.Encode(crypto.FromECDSA(ethKey))
	}
	
	return keyInfo, nil
}

func listValidatorKeys(network string) error {
	luxdHome, err := GetLuxdHome()
	if err != nil {
		return err
	}
	
	ux.Logger.PrintToUser("📋 Validator Keys")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 60))
	
	networks := []string{"mainnet", "testnet", "local"}
	if network != "" && network != "all" {
		networks = []string{network}
	}
	
	for _, net := range networks {
		keysFile := filepath.Join(luxdHome, "keys", net, "validators.json")
		
		// Check if file exists
		if _, err := os.Stat(keysFile); os.IsNotExist(err) {
			if network != "" && network != "all" {
				ux.Logger.PrintToUser("\n❌ No keys found for %s", net)
			}
			continue
		}
		
		// Read keys
		data, err := os.ReadFile(keysFile)
		if err != nil {
			ux.Logger.PrintToUser("\n⚠️  Error reading keys for %s: %v", net, err)
			continue
		}
		
		var keySet ValidatorKeySet
		if err := json.Unmarshal(data, &keySet); err != nil {
			ux.Logger.PrintToUser("\n⚠️  Error parsing keys for %s: %v", net, err)
			continue
		}
		
		ux.Logger.PrintToUser("\n🌐 Network: %s", strings.Title(net))
		ux.Logger.PrintToUser("   Total Validators: %d", len(keySet.Keys))
		ux.Logger.PrintToUser("   Created: %s", keySet.CreatedAt.Format(time.RFC3339))
		ux.Logger.PrintToUser("   Location: %s", keysFile)
		
		// Show first few validators
		for i, key := range keySet.Keys {
			if i >= 3 {
				ux.Logger.PrintToUser("   ... and %d more validators", len(keySet.Keys)-3)
				break
			}
			ux.Logger.PrintToUser("   • Node %02d: %s", i+1, key.NodeID)
		}
	}
	
	return nil
}

func importValidatorKeys(network, sourcePath string) error {
	// Implementation for importing existing keys
	ux.Logger.PrintToUser("📥 Importing validator keys...")
	ux.Logger.PrintToUser("Network: %s", network)
	ux.Logger.PrintToUser("Source: %s", sourcePath)
	
	// TODO: Implement key import logic
	
	return nil
}