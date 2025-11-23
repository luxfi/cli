// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package migratecmd

import (
	"fmt"
	"os"

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
		sourceRPC       string
		destRPC         string
		blockchainID    string
		outputDir       string
		networkID       uint32
		validators      int
	)

	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare migration data via RPC",
		Long: `Prepares the migration using RPC calls to source and destination nodes:
1. Export blocks from source EVM via RPC
2. Import blocks to destination C-Chain via RPC
3. Create P-Chain genesis with validator set (if needed)

This command uses netrunner to deploy and control nodes via RPC.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Preparing Lux migration via RPC...")
			ux.Logger.PrintToUser("")

			// Create output directory structure
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Run the RPC-based migration
			ux.Logger.PrintToUser("Step 1: Exporting/importing via RPC...")
			if err := runMigration(sourceRPC, destRPC, blockchainID, int64(networkID)); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("✅ Migration RPC calls complete!")
			ux.Logger.PrintToUser("Output directory: %s", outputDir)
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("💡 Next Steps:")
			ux.Logger.PrintToUser("1. Use netrunner to deploy source EVM node:")
			ux.Logger.PrintToUser("   netrunner engine start evm-source --data-dir=/path/to/readonly/db")
			ux.Logger.PrintToUser("2. Use netrunner to deploy destination C-Chain:")
			ux.Logger.PrintToUser("   netrunner engine start c-chain")
			ux.Logger.PrintToUser("3. Run migration (RPC endpoints auto-discovered):")
			ux.Logger.PrintToUser("   lux migrate prepare")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Note: Internal RPC at port 9630 (not 9650)")
			ux.Logger.PrintToUser("      Hosts/ports discovered from netrunner at runtime")

			return nil
		},
	}

	cmd.Flags().StringVar(&sourceRPC, "source-rpc", "", "Source EVM RPC endpoint (discovered from netrunner if not specified)")
	cmd.Flags().StringVar(&destRPC, "dest-rpc", "", "Destination C-Chain RPC endpoint (discovered from netrunner if not specified)")
	cmd.Flags().StringVar(&blockchainID, "blockchain-id", "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ", "Old blockchain ID for ext/bc/<id>/rpc path")
	cmd.Flags().StringVar(&outputDir, "output", "./lux-mainnet-migration", "Output directory for migration data")
	cmd.Flags().Uint32Var(&networkID, "network-id", 96369, "Network ID")
	cmd.Flags().IntVar(&validators, "validators", 5, "Number of validators (for genesis creation)")

	// RPC endpoints discovered from netrunner at runtime
	// Source: ext/bc/<blockchain-id>/rpc (old 96369 net)
	// Dest: ext/bc/C/rpc (C-Chain)

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
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import evm data into running C-Chain",
		Long:  `Imports historical evm data into a running C-Chain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("This command will be implemented for importing data into a running network")
			return nil
		},
	}

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
