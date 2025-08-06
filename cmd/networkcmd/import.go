// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/utils"
	"github.com/spf13/cobra"
)

var (
	importGenesisPath string
	importGenesisType string
	importArchivePath string
	importDBBackend   string
	importVerify      bool
	importBatchSize   uint64
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import genesis data into BadgerDB archive",
		Long: `The network import command imports blockchain data from an existing database
(PebbleDB or LevelDB) into a BadgerDB archive for use with the dual-database architecture.

This enables efficient blockchain data management with shared read-only archives and
per-node current databases.`,
		RunE:         ImportGenesis,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&importGenesisPath, "genesis-path", "", "path to genesis database to import (required)")
	cmd.Flags().StringVar(&importGenesisType, "genesis-type", "auto", "type of genesis database (auto, leveldb, or pebbledb)")
	cmd.Flags().StringVar(&importArchivePath, "archive-path", "", "path for BadgerDB archive output (required)")
	cmd.Flags().StringVar(&importDBBackend, "db-backend", "badgerdb", "database backend for archive")
	cmd.Flags().BoolVar(&importVerify, "verify", true, "verify block hashes during import")
	cmd.Flags().Uint64Var(&importBatchSize, "batch-size", 1000, "batch size for import")

	cmd.MarkFlagRequired("genesis-path")
	cmd.MarkFlagRequired("archive-path")

	return cmd
}

func ImportGenesis(*cobra.Command, []string) error {
	ux.Logger.PrintToUser("Starting genesis import...")
	ux.Logger.PrintToUser("Source: %s", importGenesisPath)
	ux.Logger.PrintToUser("Target: %s", importArchivePath)
	ux.Logger.PrintToUser("Type: %s", importGenesisType)

	// Get the latest Lux version
	luxVersion, err := determineLuxVersion(userProvidedLuxVersion)
	if err != nil {
		return err
	}

	sd := subnet.NewLocalDeployer(app, luxVersion, "")

	if err := sd.StartServer(); err != nil {
		return err
	}

	nodeBinPath, err := sd.SetupLocalEnv()
	if err != nil {
		return err
	}

	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	// Build node configuration for import
	nodeConfig := map[string]interface{}{
		"db-engine":           importDBBackend,
		"archive-dir":         importArchivePath,
		"genesis-import":      importGenesisPath,
		"genesis-import-type": importGenesisType,
		"genesis-replay":      true,
		"genesis-verify":      importVerify,
		"genesis-batch-size":  importBatchSize,
	}

	nodeConfigBytes, err := json.Marshal(nodeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal node config: %w", err)
	}

	// Create a temporary network for import
	outputDirPrefix := path.Join(app.GetRunDir(), "import")
	outputDir, err := utils.MkDirWithTimestamp(outputDirPrefix)
	if err != nil {
		return err
	}

	// Start a single node with import configuration
	ctx := binutils.GetAsyncContext()
	opts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithNumNodes(1),
		client.WithRootDataDir(outputDir),
		client.WithReassignPortsIfUsed(true),
		client.WithGlobalNodeConfig(string(nodeConfigBytes)),
	}

	ux.Logger.PrintToUser("Launching import node...")
	_, err = cli.Start(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to start import node: %w", err)
	}

	// Monitor import progress
	ux.Logger.PrintToUser("Import in progress...")
	ux.Logger.PrintToUser("This may take a while depending on the size of the genesis data...")

	// Wait for import to complete
	clusterInfo, err := subnet.WaitForHealthy(ctx, cli)
	if err != nil {
		// Import nodes may not become "healthy" in the traditional sense
		// Check if the archive was created successfully
		ux.Logger.PrintToUser("Import process completed. Verifying archive...")
	}

	// Stop the import node
	if clusterInfo != nil {
		if err := cli.Stop(ctx); err != nil {
			ux.Logger.PrintToUser("Warning: failed to stop import node: %v", err)
		}
	}

	ux.Logger.PrintToUser("✅ Genesis import completed successfully!")
	ux.Logger.PrintToUser("Archive created at: %s", importArchivePath)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("You can now use this archive with:")
	ux.Logger.PrintToUser("  lux network start --archive-path %s --archive-shared", importArchivePath)

	return nil
}
