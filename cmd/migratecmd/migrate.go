// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package migratecmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func NewCmd(app *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate evm data to C-Chain for network upgrade",
		Long: `The migrate command helps with the one-time migration of evm 
data to C-Chain for the Lux network upgrade. This includes:
- Converting PebbleDB subnet data to LevelDB C-Chain format
- Setting up P-Chain genesis for the new validator set
- Bootstrapping a 5-node mainnet network`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newPrepareCmd(app))
	cmd.AddCommand(newBootstrapCmd(app))
	cmd.AddCommand(newImportCmd(app))
	cmd.AddCommand(newValidateCmd(app))

	return cmd
}

func newPrepareCmd(app *application.Lux) *cobra.Command {
	var (
		sourceDB   string
		outputDir  string
		networkID  uint32
		validators int
	)

	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare migration data for mainnet launch",
		Long: `Prepares the migration by:
1. Converting evm PebbleDB to C-Chain LevelDB
2. Creating P-Chain genesis with validator set
3. Generating node configurations for bootstrap validators`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Preparing Lux mainnet migration...")

			// Create output directory structure
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Create directories for each node
			for i := 1; i <= validators; i++ {
				nodeDir := filepath.Join(outputDir, fmt.Sprintf("node%d", i))
				if err := os.MkdirAll(filepath.Join(nodeDir, "staking"), 0755); err != nil {
					return fmt.Errorf("failed to create node%d directory: %w", i, err)
				}
			}

			// Run the migration tool
			ux.Logger.PrintToUser("Step 1: Converting evm data to C-Chain format...")
			if err := runMigration(sourceDB, filepath.Join(outputDir, "c-chain-db"), int64(networkID)); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			// Create P-Chain genesis
			ux.Logger.PrintToUser("Step 2: Creating P-Chain genesis...")
			if err := createPChainGenesis(outputDir, validators); err != nil {
				return fmt.Errorf("failed to create P-Chain genesis: %w", err)
			}

			// Generate node configurations
			ux.Logger.PrintToUser("Step 3: Generating node configurations...")
			if err := generateNodeConfigs(outputDir, validators); err != nil {
				return fmt.Errorf("failed to generate node configs: %w", err)
			}

			ux.Logger.PrintToUser("✅ Migration preparation complete!")
			ux.Logger.PrintToUser("Output directory: %s", outputDir)
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Next steps:")
			ux.Logger.PrintToUser("1. Review the generated configurations")
			ux.Logger.PrintToUser("2. Run 'lux migrate bootstrap' to start the network")

			return nil
		},
	}

	cmd.Flags().StringVar(&sourceDB, "source-db", "", "Path to evm PebbleDB")
	cmd.Flags().StringVar(&outputDir, "output", "./lux-mainnet-migration", "Output directory for migration data")
	cmd.Flags().Uint32Var(&networkID, "network-id", 96369, "Network ID for the new mainnet")
	cmd.Flags().IntVar(&validators, "validators", 5, "Number of bootstrap validators")

	cmd.MarkFlagRequired("source-db")

	return cmd
}

func newBootstrapCmd(app *application.Lux) *cobra.Command {
	var (
		migrationDir string
		detached     bool
	)

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap the new Lux mainnet with migrated data",
		Long:  `Starts the bootstrap validators with the migrated chain data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Bootstrapping Lux mainnet...")

			// Verify migration directory exists
			if _, err := os.Stat(migrationDir); err != nil {
				return fmt.Errorf("migration directory not found: %w", err)
			}

			// Start bootstrap nodes
			// Handle detached mode for background execution
			nodeCount := 1 // Default to 1 node for now
			if err := startBootstrapNodes(migrationDir, nodeCount, detached); err != nil {
				return fmt.Errorf("failed to start bootstrap nodes: %w", err)
			}

			ux.Logger.PrintToUser("✅ Bootstrap network started!")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Monitor the network with:")
			ux.Logger.PrintToUser("  lux migrate validate --migration-dir %s", migrationDir)

			return nil
		},
	}

	cmd.Flags().StringVar(&migrationDir, "migration-dir", "./lux-mainnet-migration", "Migration directory with prepared data")
	cmd.Flags().BoolVar(&detached, "detached", false, "Run nodes in background")

	return cmd
}

func newImportCmd(app *application.Lux) *cobra.Command {
	var (
		sourceRPC   string
		destRPC     string
		workers     int
		batchSize   int
		startBlock  uint64
		endBlock    uint64
		deployOldSubnet bool
		queryHeight bool
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import evm data into running C-Chain via RPC",
		Long: `Imports historical evm data from SubnetEVM into a running C-Chain via parallel RPC calls.

This command loads the old subnet at runtime and reads blocks via RPC - no file copying!
It uses maximum parallelization with configurable worker pools.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we need to deploy the old subnet first
			if deployOldSubnet {
				ux.Logger.PrintToUser("Deploying old subnet with existing data...")
				if err := deployOldSubnetForImport(); err != nil {
					return fmt.Errorf("failed to deploy old subnet: %w", err)
				}
			}

			// Query block height if requested
			if queryHeight {
				height, err := queryBlockHeight(sourceRPC)
				if err != nil {
					return fmt.Errorf("failed to query block height: %w", err)
				}
				ux.Logger.PrintToUser("Current block height: %d", height)
				if endBlock == 0 {
					endBlock = height
				}
			}

			// Start the parallel RPC import
			ux.Logger.PrintToUser("Starting parallel RPC import from SubnetEVM to C-Chain...")
			ux.Logger.PrintToUser("Source: %s", sourceRPC)
			ux.Logger.PrintToUser("Destination: %s", destRPC)
			ux.Logger.PrintToUser("Workers: %d", workers)
			ux.Logger.PrintToUser("Batch size: %d", batchSize)
			ux.Logger.PrintToUser("Block range: %d to %d", startBlock, endBlock)

			if err := runParallelRPCImport(sourceRPC, destRPC, workers, batchSize, startBlock, endBlock); err != nil {
				return fmt.Errorf("import failed: %w", err)
			}

			ux.Logger.PrintToUser("✅ Import completed successfully!")
			return nil
		},
	}

	// Use standardized ~/.lux paths
	defaultSourceRPC := "http://localhost:9640/ext/bc/2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB/rpc"
	defaultDestRPC := "http://localhost:9630/ext/bc/C/rpc"

	cmd.Flags().StringVar(&sourceRPC, "source", defaultSourceRPC, "Source RPC endpoint (SubnetEVM)")
	cmd.Flags().StringVar(&destRPC, "dest", defaultDestRPC, "Destination RPC endpoint (C-Chain)")
	cmd.Flags().IntVar(&workers, "workers", 200, "Number of parallel workers")
	cmd.Flags().IntVar(&batchSize, "batch", 1000, "Batch size for RPC calls")
	cmd.Flags().Uint64Var(&startBlock, "start", 0, "Start block number")
	cmd.Flags().Uint64Var(&endBlock, "end", 0, "End block number (0 = latest)")
	cmd.Flags().BoolVar(&deployOldSubnet, "deploy-subnet", false, "Deploy old subnet before import")
	cmd.Flags().BoolVar(&queryHeight, "query-height", true, "Query and display current block height")

	return cmd
}

func newValidateCmd(app *application.Lux) *cobra.Command {
	var migrationDir string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the migrated network",
		Long:  `Checks that the migration was successful and the network is healthy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Validating migrated network...")

			// Check node health
			if err := validateNetwork(migrationDir); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			ux.Logger.PrintToUser("✅ Network validation passed!")
			return nil
		},
	}

	cmd.Flags().StringVar(&migrationDir, "migration-dir", "./lux-mainnet-migration", "Migration directory")

	return cmd
}
