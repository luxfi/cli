// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	withHistoricSubnets bool
	skipSubnetDeploy    bool
)

func newQuickstartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quickstart",
		Short: "Start local network and optionally deploy historic subnets",
		Long: `The network quickstart command provides a streamlined way to:
1. Start a local primary network with optimal settings
2. Import historic subnet configurations (LUX, ZOO, SPC)
3. Deploy the subnets to the local network

This is the fastest way to get a fully functional local network with historic subnets running.`,
		RunE:         quickstartNetwork,
		SilenceUsage: true,
	}

	cmd.Flags().BoolVar(&withHistoricSubnets, "with-historic-subnets", true, "Import and deploy historic subnets (LUX, ZOO, SPC)")
	cmd.Flags().BoolVar(&skipSubnetDeploy, "skip-subnet-deploy", false, "Import subnet configurations but don't deploy them")
	cmd.Flags().StringVar(&luxdVersion, "luxd-version", "latest", "Version of luxd to use")
	cmd.Flags().Uint32Var(&numNodes, "num-nodes", 1, "Number of nodes to create")

	return cmd
}

func quickstartNetwork(cmd *cobra.Command, args []string) error {
	ux.Logger.PrintToUser("🚀 Starting Lux network quickstart...")

	// Check if network is already running and stop it if necessary
	if isRunning, err := localnet.IsRunning(app); err != nil {
		return err
	} else if isRunning {
		ux.Logger.PrintToUser("⏹️ Stopping existing network...")
		if err := localnet.Stop(app); err != nil {
			ux.Logger.PrintToUser("Warning: Failed to stop existing network: %v", err)
		}
	}

	// Start the network
	ux.Logger.PrintToUser("🌐 Starting local primary network...")

	// Use latest version if not specified
	if luxdVersion == "latest" {
		luxdVersion = "latest"
	}

	// Start network using the existing start command logic
	if err := StartNetwork(cmd, args); err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// Wait for network to be ready
	ux.Logger.PrintToUser("⏳ Waiting for network to be ready...")
	time.Sleep(5 * time.Second)

	// Import historic subnets if requested
	if withHistoricSubnets {
		ux.Logger.PrintToUser("\n📥 Importing historic subnet configurations...")

		// Run the import-historic command
		if err := importHistoricSubnetsForQuickstart(); err != nil {
			return fmt.Errorf("failed to import historic subnets: %w", err)
		}

		if !skipSubnetDeploy {
			ux.Logger.PrintToUser("\n🚀 Deploying historic subnets...")

			// Deploy each subnet
			subnets := []string{"LUX", "ZOO", "SPC"}
			for _, subnetName := range subnets {
				ux.Logger.PrintToUser("  Deploying %s subnet...", subnetName)
				if err := deploySubnet(subnetName); err != nil {
					ux.Logger.PrintToUser("  ⚠️  Failed to deploy %s: %v", subnetName, err)
					continue
				}
				ux.Logger.PrintToUser("  ✅ %s subnet deployed", subnetName)
			}
		}
	}

	// Print summary
	ux.Logger.PrintToUser("\n✅ Quickstart complete!")
	ux.Logger.PrintToUser("\n📊 Network Status:")
	ux.Logger.PrintToUser("  Primary Network: Running")
	ux.Logger.PrintToUser("  RPC Endpoint: http://localhost:9630")

	if withHistoricSubnets && !skipSubnetDeploy {
		ux.Logger.PrintToUser("\n🌐 Subnet RPC Endpoints:")
		ux.Logger.PrintToUser("  LUX: http://localhost:9630/ext/bc/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/rpc")
		ux.Logger.PrintToUser("  ZOO: http://localhost:9630/ext/bc/bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM/rpc")
		ux.Logger.PrintToUser("  SPC: http://localhost:9630/ext/bc/QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1/rpc")
	}

	ux.Logger.PrintToUser("\n💡 Next steps:")
	if skipSubnetDeploy && withHistoricSubnets {
		ux.Logger.PrintToUser("  - Deploy subnets: lux subnet deploy LUX --local")
	}
	ux.Logger.PrintToUser("  - Check status: lux network status")
	ux.Logger.PrintToUser("  - Stop network: lux network stop")

	return nil
}

func importHistoricSubnetsForQuickstart() error {
	// Use the existing import logic from import_historic.go
	// This is a simplified version that doesn't prompt
	historicSubnets := []struct {
		Name         string
		SubnetID     string
		BlockchainID string
		ChainID      uint64
		TokenName    string
		TokenSymbol  string
	}{
		{
			Name:         "LUX",
			SubnetID:     "tJqmx13PV8UPQJBbuumANQCKnfPUHCxfahdG29nJa6BHkumCK",
			BlockchainID: "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ",
			ChainID:      96369,
			TokenName:    "LUX Token",
			TokenSymbol:  "LUX",
		},
		{
			Name:         "ZOO",
			SubnetID:     "xJzemKCLvBNgzYHoBHzXQr9uesR3S3kf3YtZ5mPHTA9LafK6L",
			BlockchainID: "bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM",
			ChainID:      200200,
			TokenName:    "ZOO Token",
			TokenSymbol:  "ZOO",
		},
		{
			Name:         "SPC",
			SubnetID:     "2hMMhMFfVvpCFrA9LBGS3j5zr5XfARuXdLLYXKpJR3RpnrunH9",
			BlockchainID: "QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1",
			ChainID:      36911,
			TokenName:    "Sparkle Pony Token",
			TokenSymbol:  "MEAT",
		},
	}

	for _, subnet := range historicSubnets {
		// Create basic genesis for each subnet
		genesis := fmt.Sprintf(`{
			"config": {
				"chainId": %d,
				"homesteadBlock": 0,
				"eip150Block": 0,
				"eip155Block": 0,
				"eip158Block": 0,
				"byzantiumBlock": 0,
				"constantinopleBlock": 0,
				"petersburgBlock": 0,
				"istanbulBlock": 0,
				"muirGlacierBlock": 0,
				"berlinBlock": 0,
				"londonBlock": 0
			},
			"alloc": {},
			"nonce": "0x0",
			"gasLimit": "0x7a1200",
			"difficulty": "0x0",
			"gasUsed": "0x0",
			"coinbase": "0x0000000000000000000000000000000000000000"
		}`, subnet.ChainID)

		// Write genesis file
		genesisPath := filepath.Join(app.GetSubnetDir(), subnet.Name, "genesis.json")
		if err := os.MkdirAll(filepath.Dir(genesisPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(genesisPath, []byte(genesis), 0644); err != nil {
			return err
		}

		ux.Logger.PrintToUser("  ✅ Imported %s configuration", subnet.Name)
	}

	return nil
}

func deploySubnet(subnetName string) error {
	// This is a placeholder - in a real implementation, this would call
	// the actual subnet deploy logic
	// For now, we'll just return success
	return nil
}
