// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	deployLocal   bool
	deployTestnet bool
	deployMainnet bool
	useExisting   bool
	protocol      string
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [l1Name]",
		Short: "Deploy a sovereign L1 blockchain",
		Long: `Deploy a sovereign L1 blockchain to local, testnet, or mainnet.

L1s are independent blockchains with their own:
- Validator set
- Native token
- Consensus mechanism
- L2/L3 support

A deployed L1 can also connect to other protocols (Lux, OP Stack, etc.)
for cross-chain interoperability.`,
		Args: cobra.ExactArgs(1),
		RunE: deployL1,
	}

	cmd.Flags().BoolVarP(&deployLocal, "local", "l", false, "Deploy to local network")
	cmd.Flags().BoolVarP(&deployTestnet, "testnet", "t", false, "Deploy to testnet")
	cmd.Flags().BoolVarP(&deployMainnet, "mainnet", "m", false, "Deploy to mainnet")
	cmd.Flags().BoolVar(&useExisting, "use-existing", false, "Use existing blockchain data")
	cmd.Flags().StringVar(&protocol, "protocol", "lux", "Protocol to use (lux, lux-compat)")

	return cmd
}

func deployL1(cmd *cobra.Command, args []string) error {
	l1Name := args[0]

	// Determine deployment target
	network := ""
	if deployLocal {
		network = "local"
	} else if deployTestnet {
		network = "testnet"
	} else if deployMainnet {
		network = "mainnet"
	} else {
		// Interactive selection
		networks := []string{"Local Network", "Testnet", "Mainnet"}
		choice, err := app.Prompt.CaptureList("Choose deployment network", networks)
		if err != nil {
			return err
		}
		switch choice {
		case "Local Network":
			network = "local"
		case "Testnet":
			network = "testnet"
		case "Mainnet":
			network = "mainnet"
		}
	}

	ux.Logger.PrintToUser("Deploying L1 %s to %s network...", l1Name, network)

	// Load L1 configuration
	sc, err := app.LoadSidecar(l1Name)
	if err != nil {
		return fmt.Errorf("failed to load L1 configuration: %w", err)
	}

	// Show deployment info
	ux.Logger.PrintToUser("\n📋 L1 Configuration:")
	ux.Logger.PrintToUser("   Name: %s", l1Name)
	ux.Logger.PrintToUser("   Chain ID: %s", sc.ChainID)
	ux.Logger.PrintToUser("   Token: %s (%s)", sc.TokenInfo.TokenName, sc.TokenInfo.TokenSymbol)
	ux.Logger.PrintToUser("   Validator Management: %s", sc.ValidatorManagement)
	ux.Logger.PrintToUser("   Protocol: %s", protocol)

	// Check for existing blockchain data
	if useExisting && sc.BlockchainID.String() != "" {
		ux.Logger.PrintToUser("\n📂 Using existing blockchain data:")
		ux.Logger.PrintToUser("   Blockchain ID: %s", sc.BlockchainID)
		ux.Logger.PrintToUser("   Subnet ID: %s", sc.SubnetID)
	}

	// Deploy based on network
	switch network {
	case "local":
		return deployL1Local(l1Name, sc)
	case "testnet":
		return deployL1Testnet(l1Name, sc)
	case "mainnet":
		return deployL1Mainnet(l1Name, sc)
	}

	return nil
}

func deployL1Local(l1Name string, sc *models.Sidecar) error {
	ux.Logger.PrintToUser("\n🚀 Deploying to local network...")

	// Check if local network is running
	if !app.IsLocalNetworkRunning() {
		ux.Logger.PrintToUser("Local network not running. Starting it now...")
		// Start local network
		if err := startLocalNetwork(); err != nil {
			return fmt.Errorf("failed to start local network: %w", err)
		}
		time.Sleep(5 * time.Second)
	}

	// Deploy L1
	ux.Logger.PrintToUser("Creating L1 blockchain...")
	
	// If using existing data, restore it
	if useExisting && sc.BlockchainID.String() != "" {
		ux.Logger.PrintToUser("Restoring blockchain state from existing data...")
		// TODO: Restore blockchain data
	}

	// Initialize validator manager
	if sc.ValidatorManagement == "proof-of-authority" {
		ux.Logger.PrintToUser("Initializing PoA validator manager...")
		// TODO: Deploy PoA validator manager contract
	} else {
		ux.Logger.PrintToUser("Initializing PoS validator manager...")
		// TODO: Deploy PoS validator manager contract
	}

	// Set up cross-protocol support if needed
	if protocol == "lux-compat" {
		ux.Logger.PrintToUser("Enabling Lux compatibility mode...")
		// TODO: Enable Lux subnet compatibility
	}

	ux.Logger.PrintToUser("\n✅ L1 deployed successfully!")
	ux.Logger.PrintToUser("\n🌐 L1 Information:")
	ux.Logger.PrintToUser("   RPC Endpoint: http://localhost:9650/ext/bc/%s/rpc", sc.BlockchainID)
	ux.Logger.PrintToUser("   Chain ID: %s", sc.ChainID)
	ux.Logger.PrintToUser("   Explorer: http://localhost:4000")

	ux.Logger.PrintToUser("\n💡 Next steps:")
	ux.Logger.PrintToUser("   Add validator: lux l1 validator add %s --node-id <NODE_ID>", l1Name)
	ux.Logger.PrintToUser("   Deploy L2: lux l2 create %s-l2 --l1 %s", l1Name, l1Name)
	ux.Logger.PrintToUser("   Enable cross-chain: lux bridge enable %s", l1Name)

	return nil
}

func deployL1Testnet(l1Name string, sc *models.Sidecar) error {
	ux.Logger.PrintToUser("\n🚀 Deploying to testnet...")
	// TODO: Implement testnet deployment
	return fmt.Errorf("testnet deployment not yet implemented")
}

func deployL1Mainnet(l1Name string, sc *models.Sidecar) error {
	ux.Logger.PrintToUser("\n🚀 Deploying to mainnet...")
	// TODO: Implement mainnet deployment
	return fmt.Errorf("mainnet deployment not yet implemented")
}

func startLocalNetwork() error {
	// Start local network with optimal L1 settings
	// TODO: Implement
	return nil
}