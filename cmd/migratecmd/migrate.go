// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
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
		Short: "Migrate subnet-evm data to C-Chain for network upgrade",
		Long: `The migrate command helps with the one-time migration of subnet-evm 
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
1. Converting subnet-evm PebbleDB to C-Chain LevelDB
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
					return fmt.Errorf("failed to create node%d directory: %w", err, i)
				}
			}

			// Run the migration tool
			ux.Logger.PrintToUser("Step 1: Converting subnet-evm data to C-Chain format...")
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
			if err := generateNodeConfigs(outputDir, validators, networkID); err != nil {
				return fmt.Errorf("failed to generate node configs: %w", err)
			}

			ux.Logger.PrintToUser("✅ Migration preparation complete!")
			ux.Logger.PrintToUser(fmt.Sprintf("Output directory: %s", outputDir))
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Next steps:")
			ux.Logger.PrintToUser("1. Review the generated configurations")
			ux.Logger.PrintToUser("2. Run 'lux migrate bootstrap' to start the network")

			return nil
		},
	}

	cmd.Flags().StringVar(&sourceDB, "source-db", "", "Path to subnet-evm PebbleDB")
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
			if err := startBootstrapNodes(migrationDir, detached); err != nil {
				return fmt.Errorf("failed to start bootstrap nodes: %w", err)
			}

			ux.Logger.PrintToUser("✅ Bootstrap network started!")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Monitor the network with:")
			ux.Logger.PrintToUser("  lux migrate validate --migration-dir " + migrationDir)

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
		Short: "Import subnet-evm data into running C-Chain",
		Long:  `Imports historical subnet-evm data into a running C-Chain`,
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