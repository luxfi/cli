// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snapshotcmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/snapshot"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the top-level snapshot command
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Create and manage network snapshots",
		Long: `The snapshot command creates native incremental backups of running networks.

This uses BadgerDB's native backup API for:
  - Incremental backups (only changes since last backup)
  - Consistent snapshots (atomic database state)
  - Fast restore times
  - Smaller backup sizes (zstd compressed)

USAGE:

  # Create snapshot of running network (auto-detects which network)
  lux snapshot

  # Create snapshot of specific network
  lux snapshot --mainnet
  lux snapshot --testnet

  # Create snapshot with custom name
  lux snapshot --name my-backup

  # Force full backup (not incremental)
  lux snapshot --full

  # Restore from snapshot
  lux snapshot restore my-backup

  # List available snapshots
  lux snapshot list

INCREMENTAL BACKUPS:

  By default, snapshots are incremental - they only include data that changed
  since the last backup. This makes them much smaller and faster.

  First backup: Full backup (~90MB compressed for fresh network)
  Subsequent:   Incremental (~1-10MB for typical changes)

  Use --full to force a complete backup.`,
		RunE: createSnapshot,
	}

	// Subcommands
	cmd.AddCommand(newRestoreCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCleanCmd())

	// Flags for main snapshot command
	cmd.Flags().StringVar(&snapshotName, "name", "", "snapshot name (default: <network>-<date>)")
	cmd.Flags().BoolVar(&fullBackup, "full", false, "create full backup instead of incremental")
	cmd.Flags().BoolVar(&snapshotMainnet, "mainnet", false, "snapshot mainnet network")
	cmd.Flags().BoolVar(&snapshotTestnet, "testnet", false, "snapshot testnet network")
	cmd.Flags().BoolVar(&snapshotDevnet, "devnet", false, "snapshot devnet network")

	return cmd
}

var (
	snapshotName    string
	fullBackup      bool
	snapshotMainnet bool
	snapshotTestnet bool
	snapshotDevnet  bool
)

func createSnapshot(cmd *cobra.Command, args []string) error {
	// Determine network type
	networkType := ""
	if snapshotMainnet {
		networkType = "mainnet"
	} else if snapshotTestnet {
		networkType = "testnet"
	} else if snapshotDevnet {
		networkType = "devnet"
	}

	// Auto-detect if not specified
	if networkType == "" {
		runningNetworks := app.GetAllRunningNetworks()
		if len(runningNetworks) == 0 {
			return fmt.Errorf("no network running. Start a network first with 'lux network start'")
		}
		if len(runningNetworks) > 1 {
			ux.Logger.PrintToUser("Multiple networks running: %s", strings.Join(runningNetworks, ", "))
			ux.Logger.PrintToUser("Please specify which one to snapshot:")
			for _, net := range runningNetworks {
				ux.Logger.PrintToUser("  lux snapshot --%s", net)
			}
			return fmt.Errorf("ambiguous: multiple networks running")
		}
		networkType = runningNetworks[0]
	}

	// Generate snapshot name if not provided
	if snapshotName == "" {
		snapshotName = fmt.Sprintf("%s-%s", networkType, time.Now().Format("2006-01-02"))
	}

	ux.Logger.PrintToUser("Creating %s snapshot: %s", func() string {
		if fullBackup {
			return "full"
		}
		return "incremental"
	}(), snapshotName)

	// Create snapshot using native backup
	sm := snapshot.NewSnapshotManager(app.GetBaseDir())
	if err := sm.CreateSnapshot(snapshotName, !fullBackup); err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Get snapshot info
	info, err := sm.GetSnapshotInfo(snapshotName)
	if err == nil {
		ux.Logger.PrintToUser("Snapshot created successfully:")
		ux.Logger.PrintToUser("  Name:        %s", info.Name)
		ux.Logger.PrintToUser("  Size:        %s", snapshot.FormatBytes(info.Size))
		ux.Logger.PrintToUser("  Incremental: %v", info.Incremental)
		ux.Logger.PrintToUser("  Path:        %s", info.Path)
	} else {
		ux.Logger.PrintToUser("Snapshot '%s' created successfully.", snapshotName)
	}

	return nil
}

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [name]",
		Short: "Restore network from snapshot",
		Long: `Restore a network from a previously created snapshot.

The network must be stopped before restoring. After restore, start the
network with 'lux network start'.

EXAMPLES:

  # Restore from snapshot
  lux snapshot restore my-backup

  # Restore mainnet snapshot
  lux snapshot restore mainnet-2026-01-19 --mainnet`,
		Args: cobra.ExactArgs(1),
		RunE: restoreSnapshot,
	}
	cmd.Flags().BoolVar(&snapshotMainnet, "mainnet", false, "restore to mainnet")
	cmd.Flags().BoolVar(&snapshotTestnet, "testnet", false, "restore to testnet")
	cmd.Flags().BoolVar(&snapshotDevnet, "devnet", false, "restore to devnet")
	return cmd
}

func restoreSnapshot(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check no network is running
	runningNetworks := app.GetAllRunningNetworks()
	if len(runningNetworks) > 0 {
		return fmt.Errorf("network(s) running: %s. Stop them first with 'lux network stop'",
			strings.Join(runningNetworks, ", "))
	}

	ux.Logger.PrintToUser("Restoring from snapshot: %s", name)

	sm := snapshot.NewSnapshotManager(app.GetBaseDir())
	if err := sm.RestoreSnapshot(name); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	ux.Logger.PrintToUser("Snapshot restored successfully.")
	ux.Logger.PrintToUser("Start the network with: lux network start")

	return nil
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available snapshots",
		RunE:  listSnapshots,
	}
}

func listSnapshots(cmd *cobra.Command, args []string) error {
	sm := snapshot.NewSnapshotManager(app.GetBaseDir())
	snapshots, err := sm.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		ux.Logger.PrintToUser("No snapshots found.")
		ux.Logger.PrintToUser("Create one with: lux snapshot")
		return nil
	}

	ux.Logger.PrintToUser("Available snapshots:")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("%-30s %-12s %-12s %s", "NAME", "SIZE", "TYPE", "DATE")
	ux.Logger.PrintToUser("%-30s %-12s %-12s %s", "----", "----", "----", "----")

	for _, s := range snapshots {
		snapType := "full"
		if s.Incremental {
			snapType = "incremental"
		}
		ux.Logger.PrintToUser("%-30s %-12s %-12s %s",
			s.Name,
			snapshot.FormatBytes(s.Size),
			snapType,
			s.Created.Format("2006-01-02 15:04"))
	}

	return nil
}

func newCleanCmd() *cobra.Command {
	var dryRun bool
	var keepLast int

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up old snapshots and logs",
		Long: `Clean up old snapshots, large log files, and stale run directories.

This command frees disk space by removing:
  - Old backup directories
  - Large netrunner log files (>100MB)
  - Stale run directories from previous sessions

EXAMPLES:

  # Preview what would be cleaned
  lux snapshot clean --dry-run

  # Clean everything, keep last 3 snapshots
  lux snapshot clean --keep 3

  # Clean all old data
  lux snapshot clean`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := snapshot.NewSnapshotManager(app.GetBaseDir())
			cfg := snapshot.DefaultCleanupConfig()
			cfg.DryRun = dryRun
			cfg.Verbose = true

			if dryRun {
				ux.Logger.PrintToUser("Dry run - showing what would be cleaned:")
			}

			result := sm.Cleanup(cfg)

			if result.TotalBytesFreed() > 0 || dryRun {
				ux.Logger.PrintToUser("")
				if dryRun {
					ux.Logger.PrintToUser("Would free: %s", snapshot.FormatBytes(result.TotalBytesFreed()))
				} else {
					ux.Logger.PrintToUser("Freed: %s", snapshot.FormatBytes(result.TotalBytesFreed()))
				}
				ux.Logger.PrintToUser("  Logs:      %d files", result.LogsDeleted)
				ux.Logger.PrintToUser("  Backups:   %d directories", result.BackupsDeleted)
				ux.Logger.PrintToUser("  Stale:     %d run directories", result.StaleRunsDeleted)
			} else {
				ux.Logger.PrintToUser("Nothing to clean.")
			}

			for _, err := range result.Errors {
				ux.Logger.PrintToUser("Warning: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be cleaned without deleting")
	cmd.Flags().IntVar(&keepLast, "keep", 3, "number of recent snapshots to keep")

	return cmd
}
