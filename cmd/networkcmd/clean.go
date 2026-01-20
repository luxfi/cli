// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"errors"
	"time"

	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/snapshot"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/warp/relayer"
	"github.com/luxfi/cli/pkg/warp/signatureaggregator"
	"github.com/luxfi/constants"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/models"

	"github.com/spf13/cobra"
)

var (
	resetPlugins   bool
	cleanLogs      bool   // Clean up large log files
	cleanBackups   bool   // Clean up old backup directories
	cleanStaleRuns bool   // Clean up stale run directories
	cleanAll       bool   // Clean all of the above
	cleanDryRun    bool   // Show what would be deleted without deleting
	cleanMaxLogMB  int    // Maximum log file size in MB
	cleanMaxAgeDays int   // Maximum age for backups/logs in days
)

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Stop network and delete runtime data (preserves chain configs)",
		Long: `The network clean command stops the network and deletes runtime data.

⚠️  IMPORTANT - WHAT GETS DELETED:

  Runtime Data (DELETED):
  - Network snapshots (blockchain state, databases)
  - Validator state
  - Log files
  - Running processes

  Chain Configs (PRESERVED):
  - Chain configurations in ~/.lux/chains/
  - Genesis files
  - Sidecar metadata

BEHAVIOR:

  1. Stops the running network gracefully
  2. Deletes network runtime data and snapshots
  3. Removes local deployment info from sidecars
  4. Preserves chain configurations for redeployment

  After cleaning, you can redeploy your chains to a fresh network:
    lux network start --devnet
    lux chain deploy mychain

OPTIONS:

  --reset-plugins    Also delete the plugins directory (removes user-installed VMs)

EXAMPLES:

  # Clean network runtime (most common)
  lux network clean

  # Clean and also remove custom VM plugins
  lux network clean --reset-plugins

WHEN TO USE:

  ✓ Network state is corrupted
  ✓ Want to start fresh but keep chain configs
  ✓ Testing deployment from scratch
  ✓ Cleaning up after development session

  ✗ Just want to stop the network (use 'lux network stop')
  ✗ Want to delete a specific chain (use 'lux chain delete <name>')

CLEAN vs STOP:

  lux network stop     - Saves state for resuming later
  lux network clean    - Deletes runtime data, preserves chain configs
  lux chain delete     - Deletes a specific chain configuration

STORAGE CLEANUP:

  The CLI can accumulate significant storage over time:
  - netrunner-server.log files (can grow to 100GB+)
  - .backup.* directories from snapshot loads
  - Stale run directories from previous sessions

  Use --logs, --backups, --stale-runs, or --all to clean these:
    lux network clean --all              # Clean everything
    lux network clean --logs             # Clean large logs only
    lux network clean --all --dry-run    # Preview what would be deleted

NOTE: Chain configurations are explicitly preserved. To delete a chain
configuration, use: lux chain delete <chainName>`,
		RunE: clean,
		Args: cobrautils.ExactArgs(0),
	}
	cmd.Flags().BoolVar(&resetPlugins, "reset-plugins", false, "also reset the plugins directory (removes user-installed VMs)")
	cmd.Flags().BoolVar(&cleanLogs, "logs", false, "clean up large netrunner-server.log files")
	cmd.Flags().BoolVar(&cleanBackups, "backups", false, "clean up old .backup.* directories")
	cmd.Flags().BoolVar(&cleanStaleRuns, "stale-runs", false, "clean up stale run directories from previous sessions")
	cmd.Flags().BoolVar(&cleanAll, "all", false, "clean all: logs, backups, and stale runs")
	cmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "show what would be deleted without actually deleting")
	cmd.Flags().IntVar(&cleanMaxLogMB, "max-log-mb", 100, "maximum log file size in MB before cleanup")
	cmd.Flags().IntVar(&cleanMaxAgeDays, "max-age-days", 7, "maximum age in days for backups and stale runs")

	return cmd
}

func clean(*cobra.Command, []string) error {
	if err := localnet.LocalNetworkStop(app); err != nil && !errors.Is(err, localnet.ErrNetworkNotRunning) {
		return err
	} else if err == nil {
		ux.Logger.PrintToUser("Process terminated.")
	} else {
		ux.Logger.PrintToUser("%s", luxlog.Red.Wrap("No network is running."))
	}

	if err := relayer.Cleanup(
		app.GetLocalRelayerRunPath(models.Local),
		app.GetLocalRelayerLogPath(models.Local),
		app.GetLocalRelayerStorageDir(models.Local),
	); err != nil {
		return err
	}

	// Clean up signature aggregator
	network := models.NewLocalNetwork()
	if err := signatureaggregator.Cleanup(
		app.GetLocalRelayerRunPath(network),
		app.GetLocalRelayerStorageDir(network),
	); err != nil {
		return err
	}

	if resetPlugins {
		if err := app.ResetPluginsDir(); err != nil {
			return err
		}
	}

	if err := removeLocalDeployInfoFromSidecars(); err != nil {
		return err
	}

	// SAFETY: Use SafeRemoveAll to prevent accidental deletion of protected directories
	// Only delete the snapshot, NOT the chains directory
	snapshotPath := app.GetSnapshotPath(constants.DefaultSnapshotName)
	if err := app.SafeRemoveAll(snapshotPath); err != nil {
		// Log warning but don't fail - the snapshot may not exist
		ux.Logger.PrintToUser("Warning: could not clean snapshot: %v", err)
	}

	clusterNames, err := localnet.GetRunningLocalClustersConnectedToLocalNetwork(app)
	if err != nil {
		return err
	}
	for _, clusterName := range clusterNames {
		if err := localnet.LocalClusterRemove(app, clusterName); err != nil {
			return err
		}
	}

	// Storage cleanup: logs, backups, stale runs
	if cleanAll || cleanLogs || cleanBackups || cleanStaleRuns {
		if err := performStorageCleanup(); err != nil {
			ux.Logger.PrintToUser("Warning: storage cleanup encountered errors: %v", err)
		}
	}

	// Explicitly note that chain configs are preserved
	ux.Logger.PrintToUser("Note: Chain configurations in %s are preserved. Use 'lux chain delete <name>' to remove individual chains.", app.GetChainsDir())

	return nil
}

// performStorageCleanup cleans up logs, backups, and stale runs based on flags
func performStorageCleanup() error {
	sm := snapshot.NewSnapshotManager(app.GetBaseDir())

	cfg := snapshot.CleanupConfig{
		MaxLogSize:     int64(cleanMaxLogMB) * 1024 * 1024,
		MaxLogAge:      time.Duration(cleanMaxAgeDays) * 24 * time.Hour,
		MaxBackupAge:   time.Duration(cleanMaxAgeDays) * 24 * time.Hour,
		MaxStaleRunAge: time.Duration(cleanMaxAgeDays) * 24 * time.Hour,
		DryRun:         cleanDryRun,
		Verbose:        true,
	}

	// If specific flags are set, only clean those categories
	// Otherwise, if --all is set, clean everything
	if !cleanAll {
		// Disable categories not explicitly requested
		if !cleanLogs {
			cfg.MaxLogSize = 0 // Skip log cleanup
		}
		if !cleanBackups {
			cfg.MaxBackupAge = 0 // Skip backup cleanup
		}
		if !cleanStaleRuns {
			cfg.MaxStaleRunAge = 0 // Skip stale run cleanup
		}
	}

	if cleanDryRun {
		ux.Logger.PrintToUser("Dry run - showing what would be cleaned:")
	} else {
		ux.Logger.PrintToUser("Cleaning up storage...")
	}

	result := sm.Cleanup(cfg)

	// Report results
	if result.LogsDeleted > 0 || result.BackupsDeleted > 0 || result.StaleRunsDeleted > 0 {
		action := "Would free"
		if !cleanDryRun {
			action = "Freed"
		}
		ux.Logger.PrintToUser("%s %s total:", action, snapshot.FormatBytes(result.TotalBytesFreed()))
		if result.LogsDeleted > 0 {
			ux.Logger.PrintToUser("  - %d log files (%s)", result.LogsDeleted, snapshot.FormatBytes(result.LogBytesFreed))
		}
		if result.BackupsDeleted > 0 {
			ux.Logger.PrintToUser("  - %d backup directories (%s)", result.BackupsDeleted, snapshot.FormatBytes(result.BackupBytesFreed))
		}
		if result.StaleRunsDeleted > 0 {
			ux.Logger.PrintToUser("  - %d stale run directories (%s)", result.StaleRunsDeleted, snapshot.FormatBytes(result.StaleRunBytesFreed))
		}
	} else {
		ux.Logger.PrintToUser("No items found to clean.")
	}

	// Report errors
	for _, err := range result.Errors {
		ux.Logger.PrintToUser("Warning: %v", err)
	}

	return nil
}

func removeLocalDeployInfoFromSidecars() error {
	// Remove all local deployment info from sidecar files
	deployedChains, err := chain.GetLocallyDeployedChainsFromFile(app)
	if err != nil {
		return err
	}

	for _, chain := range deployedChains {
		sc, err := app.LoadSidecar(chain)
		if err != nil {
			return err
		}

		delete(sc.Networks, models.Local.String())
		if err = app.UpdateSidecar(&sc); err != nil {
			return err
		}
	}
	return nil
}
