// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/snapshot"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

const networkTypeCustom = "custom"

var (
	snapshotNetworkType  string
	snapshotLive         bool
	snapshotIncremental  bool
)

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage network snapshots",
		Long: `The snapshot command allows you to save, load, list, and delete snapshots of your local network state.

Snapshots capture the entire network state including all node data, databases, and configurations.

Commands:
  save <name>      - Save current network state as a named snapshot (Legacy)
  load <name>      - Load a snapshot and restart the network (Legacy)
  list             - List all available snapshots
  delete <name>    - Delete a snapshot
  advanced         - Advanced coordinated snapshots (incremental, squash, etc)

Examples:
  lux network snapshot save my-test-state
  lux network snapshot advanced create my-prod-state --incremental
  lux network snapshot list`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newSnapshotSaveCmd())
	cmd.AddCommand(newSnapshotLoadCmd())
	cmd.AddCommand(newSnapshotListCmd())
	cmd.AddCommand(newSnapshotDeleteCmd())
	cmd.AddCommand(newAdvancedSnapshotCmd())

	return cmd
}

func newSnapshotSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save current network state as a named snapshot",
		Long: `The snapshot save command saves the current network state to a named snapshot.

The snapshot includes all node data, databases, and configurations from the current network.

By default, the network must be stopped before creating a snapshot. Use --live to create
a snapshot while the network is running (creates backup from one node without stopping it).

Use --incremental to create a smaller incremental backup if a previous backup exists.
Incremental backups only store changes since the last backup, saving significant space.

Example:
  lux network snapshot save my-test-state
  lux network snapshot save my-backup --live         # Snapshot without stopping network
  lux network snapshot save my-backup --incremental  # Incremental backup (smaller, faster)`,
		Args:         cobra.ExactArgs(1),
		RunE:         saveSnapshot,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&snapshotNetworkType, "network-type", "", "network type to snapshot (mainnet, testnet, devnet, custom)")
	cmd.Flags().BoolVar(&snapshotLive, "live", false, "create snapshot from running network (backs up one node without stopping)")
	cmd.Flags().BoolVar(&snapshotIncremental, "incremental", false, "create incremental backup (smaller, faster if previous backup exists)")

	return cmd
}

func newSnapshotLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load <name>",
		Short: "Load a snapshot and restart the network",
		Long: `The snapshot load command loads a previously saved snapshot.

If the network is currently running, it will be stopped first. The snapshot
data will be copied to the active network directory and the network will be restarted.

Example:
  lux network snapshot load my-test-state`,
		Args:         cobra.ExactArgs(1),
		RunE:         loadSnapshot,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&snapshotNetworkType, "network-type", "", "network type to load snapshot into (mainnet, testnet, devnet, custom)")

	return cmd
}

func newSnapshotListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available snapshots",
		Long: `The snapshot list command displays all saved snapshots with their metadata.

Example:
  lux network snapshot list`,
		Args:         cobra.ExactArgs(0),
		RunE:         listSnapshots,
		SilenceUsage: true,
	}

	return cmd
}

func newSnapshotDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a snapshot",
		Long: `The snapshot delete command removes a saved snapshot from disk.

Example:
  lux network snapshot delete my-test-state`,
		Args:         cobra.ExactArgs(1),
		RunE:         deleteSnapshot,
		SilenceUsage: true,
	}

	return cmd
}

// determineNetworkType determines which network type to operate on
func determineNetworkType() string {
	if snapshotNetworkType != "" {
		// Normalize "local" to "custom"
		if snapshotNetworkType == "local" {
			return networkTypeCustom
		}
		return snapshotNetworkType
	}

	// Check for running networks in priority order
	// "dev" is the multi-validator dev mode (network ID 1337)
	// "custom" is for arbitrary local networks
	for _, netType := range []string{"dev", networkTypeCustom, "devnet", "testnet", "mainnet"} {
		state, err := app.LoadNetworkStateForType(netType)
		if err == nil && state != nil && state.Running {
			return netType
		}
	}

	// Default to custom
	return networkTypeCustom
}

func saveSnapshot(_ *cobra.Command, args []string) error {
	snapshotName := args[0]

	if strings.ContainsAny(snapshotName, "/\\:*?\"<>|") {
		return fmt.Errorf("invalid snapshot name: cannot contain special characters /\\:*?\"<>|")
	}

	networkType := determineNetworkType()
	ux.Logger.PrintToUser("Saving snapshot for network: %s", networkType)

	state, err := app.LoadNetworkStateForType(networkType)
	isRunning := err == nil && state != nil && state.Running

	if isRunning && !snapshotLive {
		return fmt.Errorf("network %s is currently running. Use --live to snapshot without stopping, or stop it first with 'lux network stop --network-type %s'", networkType, networkType)
	}

	runDir := app.GetRunDir()
	sourceDir := filepath.Join(runDir, networkType)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("no network data found for %s. Start a network first", networkType)
	}

	snapshotsDir := app.GetSnapshotsDir()
	if err := os.MkdirAll(snapshotsDir, 0o750); err != nil {
		return fmt.Errorf("failed to create snapshots directory: %w", err)
	}

	snapshotDir := filepath.Join(snapshotsDir, snapshotName)
	if _, err := os.Stat(snapshotDir); err == nil {
		return fmt.Errorf("snapshot '%s' already exists. Delete it first or choose a different name", snapshotName)
	}

	// For live snapshots, we backup from node5 (least disruptive)
	if snapshotLive && isRunning {
		return saveLiveSnapshot(snapshotName, networkType, sourceDir, snapshotDir)
	}

	// For incremental snapshots, use native database backup API
	if snapshotIncremental {
		return saveIncrementalSnapshot(snapshotName, networkType)
	}

	ux.Logger.PrintToUser("Creating snapshot: %s", snapshotName)
	ux.Logger.PrintToUser("Source: %s", sourceDir)
	ux.Logger.PrintToUser("Destination: %s", snapshotDir)

	archivePath := filepath.Join(snapshotDir, "archive.tar.gz")
	if err := archiveDirectory(sourceDir, archivePath); err != nil {
		return fmt.Errorf("failed to create snapshot archive: %w", err)
	}

	metadata := map[string]string{
		"name":         snapshotName,
		"network_type": networkType,
		"created_at":   time.Now().Format(time.RFC3339),
		"source":       sourceDir,
		"compressed":   "true",
	}

	metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.txt")
	metadataContent := fmt.Sprintf("Name: %s\nNetwork Type: %s\nCreated: %s\nSource: %s\nCompressed: %s\n",
		metadata["name"],
		metadata["network_type"],
		metadata["created_at"],
		metadata["source"],
		metadata["compressed"])

	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0o644); err != nil { //nolint:gosec // G306: Metadata file should be readable
		ux.Logger.PrintToUser("Warning: failed to save metadata: %v", err)
	}

	ux.Logger.PrintToUser("✓ Snapshot '%s' created successfully", snapshotName)
	return nil
}

// saveLiveSnapshot creates a snapshot from a running network without stopping it.
// It backs up data from node5 to minimize impact on the network.
func saveLiveSnapshot(snapshotName, networkType, sourceDir, snapshotDir string) error {
	ux.Logger.PrintToUser("Creating live snapshot (network continues running)...")

	// Find the current run directory
	currentLink := filepath.Join(sourceDir, "current")
	currentRun, err := os.Readlink(currentLink)
	if err != nil {
		return fmt.Errorf("failed to read current run link: %w", err)
	}

	runDir := filepath.Join(sourceDir, currentRun)

	// Select node5 for backup (least impact on consensus)
	nodeDir := filepath.Join(runDir, "node5")
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		// Fallback to node1 if node5 doesn't exist
		nodeDir = filepath.Join(runDir, "node1")
	}

	ux.Logger.PrintToUser("Backing up from: %s", nodeDir)
	ux.Logger.PrintToUser("Destination: %s", snapshotDir)

	// Create snapshot directory
	if err := os.MkdirAll(snapshotDir, 0o750); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Create archive of the node directory
	// BadgerDB supports concurrent reads, so this is safe while running
	archivePath := filepath.Join(snapshotDir, "archive.tar.gz")
	if err := archiveDirectory(runDir, archivePath); err != nil {
		return fmt.Errorf("failed to create snapshot archive: %w", err)
	}

	// Write metadata
	metadata := fmt.Sprintf("Name: %s\nNetwork Type: %s\nCreated: %s\nSource: %s\nCompressed: true\nLive: true\n",
		snapshotName,
		networkType,
		time.Now().Format(time.RFC3339),
		runDir)

	metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.txt")
	if err := os.WriteFile(metadataPath, []byte(metadata), 0o644); err != nil { //nolint:gosec // G306
		ux.Logger.PrintToUser("Warning: failed to save metadata: %v", err)
	}

	ux.Logger.PrintToUser("✓ Live snapshot '%s' created successfully", snapshotName)
	ux.Logger.PrintToUser("  Network is still running")
	return nil
}

// saveIncrementalSnapshot creates a snapshot using native database backup API.
// This supports incremental backups which only store changes since the last backup.
func saveIncrementalSnapshot(snapshotName, networkType string) error {
	ux.Logger.PrintToUser("Creating incremental snapshot using native backup API...")
	ux.Logger.PrintToUser("Note: Network must be stopped for native backup to access databases.")

	sm := snapshot.NewSnapshotManager(app.GetBaseDir())
	if err := sm.CreateSnapshot(snapshotName, true); err != nil {
		return fmt.Errorf("failed to create incremental snapshot: %w", err)
	}

	ux.Logger.PrintToUser("✓ Incremental snapshot '%s' created successfully", snapshotName)
	return nil
}

func loadSnapshot(_ *cobra.Command, args []string) error {
	snapshotName := args[0]
	networkType := determineNetworkType()
	ux.Logger.PrintToUser("Loading snapshot for network: %s", networkType)

	snapshotsDir := app.GetSnapshotsDir()
	snapshotDir := filepath.Join(snapshotsDir, snapshotName)

	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	state, err := app.LoadNetworkStateForType(networkType)
	if err == nil && state != nil && state.Running {
		ux.Logger.PrintToUser("Stopping running network...")
		if err := StopNetwork(nil, nil); err != nil {
			return fmt.Errorf("failed to stop network: %w", err)
		}
		time.Sleep(2 * time.Second)
	}

	runDir := app.GetRunDir()
	destDir := filepath.Join(runDir, networkType)

	if _, err := os.Stat(destDir); err == nil {
		backupDir := destDir + ".backup." + time.Now().Format("20060102-150405")
		ux.Logger.PrintToUser("Backing up existing data to: %s", backupDir)
		if err := os.Rename(destDir, backupDir); err != nil {
			return fmt.Errorf("failed to backup existing data: %w", err)
		}
	}

	ux.Logger.PrintToUser("Loading snapshot: %s", snapshotName)

	archivePath := filepath.Join(snapshotDir, "archive.tar.gz")
	if _, err := os.Stat(archivePath); err == nil {
		ux.Logger.PrintToUser("Extracting compressed snapshot...")
		if err := extractArchive(archivePath, destDir); err != nil {
			return fmt.Errorf("failed to extract snapshot: %w", err)
		}
	} else {
		ux.Logger.PrintToUser("Copying snapshot directory...")
		if err := copyDirectory(snapshotDir, destDir); err != nil {
			return fmt.Errorf("failed to load snapshot: %w", err)
		}
	}

	metadataPath := filepath.Join(destDir, "snapshot_metadata.txt")
	_ = os.Remove(metadataPath)

	ux.Logger.PrintToUser("✓ Snapshot '%s' loaded successfully", snapshotName)
	ux.Logger.PrintToUser("\nTo start the network, run:")
	ux.Logger.PrintToUser("  lux network start --%s", networkType)

	return nil
}

func listSnapshots(_ *cobra.Command, _ []string) error {
	snapshotsDir := app.GetSnapshotsDir()
	if _, err := os.Stat(snapshotsDir); os.IsNotExist(err) {
		ux.Logger.PrintToUser("No snapshots found. Create one with 'lux network snapshot save <name>'")
		return nil
	}
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	var snapshots []string
	for _, entry := range entries {
		if entry.IsDir() {
			snapshots = append(snapshots, entry.Name())
		}
	}
	if len(snapshots) == 0 {
		ux.Logger.PrintToUser("No snapshots found.")
		return nil
	}

	ux.Logger.PrintToUser("Available snapshots:\n")
	for _, name := range snapshots {
		snapshotDir := filepath.Join(snapshotsDir, name)
		metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.txt")
		metadataBytes, err := os.ReadFile(metadataPath)
		if err == nil {
			ux.Logger.PrintToUser("Snapshot: %s", name)
			ux.Logger.PrintToUser("%s", string(metadataBytes))
		} else {
			info, _ := os.Stat(snapshotDir)
			ux.Logger.PrintToUser("Snapshot: %s", name)
			if info != nil {
				ux.Logger.PrintToUser("Modified: %s", info.ModTime().Format(time.RFC3339))
			}
		}
	}
	return nil
}

func deleteSnapshot(_ *cobra.Command, args []string) error {
	snapshotName := args[0]
	if snapshotName == "" || strings.Contains(snapshotName, "..") {
		return fmt.Errorf("invalid snapshot name")
	}
	snapshotsDir := app.GetSnapshotsDir()
	snapshotDir := filepath.Join(snapshotsDir, snapshotName)
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}
	if err := app.SafeRemoveAll(snapshotDir); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}
	ux.Logger.PrintToUser("Snapshot '%s' deleted successfully", snapshotName)
	return nil
}

// Advanced snapshot commands

func newAdvancedSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "advanced",
		Short: "Advanced snapshot operations for multi-node networks",
		Long: `Advanced snapshot commands for coordinated multi-node snapshots.

Commands:
  create <name>    - Create advanced snapshot of all nodes (base or incremental)
  restore <name>   - Restore network from advanced snapshot
  squash <network> <chain-id> - Squash incrementals into base
  download <name>  - Download from GitHub (placeholder)
  upload <name>    - Upload to GitHub (placeholder)

Examples:
  lux network snapshot advanced create production-backup --incremental
  lux network snapshot advanced restore production-backup
  lux network snapshot advanced squash mainnet 1`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newAdvancedSnapshotCreateCmd())
	cmd.AddCommand(newAdvancedSnapshotRestoreCmd())
	cmd.AddCommand(newAdvancedSnapshotSquashCmd())
	cmd.AddCommand(newAdvancedSnapshotDownloadCmd())
	cmd.AddCommand(newAdvancedSnapshotUploadCmd())

	return cmd
}

func newAdvancedSnapshotCreateCmd() *cobra.Command {
	var incremental bool

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create advanced snapshot of all nodes",
		Long: `Create a coordinated snapshot of all nodes in the network.
If --incremental is set, tries to create an incremental backup from the last checkpoint.
Otherwise creates a full base snapshot.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createAdvancedSnapshot(cmd, args, incremental)
		},
		SilenceUsage: true,
	}

	cmd.Flags().BoolVar(&incremental, "incremental", false, "Create incremental snapshot if possible")

	return cmd
}

func newAdvancedSnapshotRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "restore <name>",
		Short:        "Restore network from advanced snapshot",
		Args:         cobra.ExactArgs(1),
		RunE:         restoreAdvancedSnapshot,
		SilenceUsage: true,
	}
	return cmd
}

func newAdvancedSnapshotSquashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "squash <network> <chain-id> <snapshot-name>",
		Short: "Squash incrementals into base snapshot",
		Long: `Squashes all incremental snapshots for a specific chain into the base snapshot.
This creates a new base snapshot and removes the incrementals, saving space.`,
		Args:         cobra.ExactArgs(3),
		RunE:         squashAdvancedSnapshot,
		SilenceUsage: true,
	}
	return cmd
}

func newAdvancedSnapshotDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <name>",
		Short: "Download snapshot from GitHub",
		Long: `Download a snapshot from GitHub releases.

This feature will download chunked snapshot files from GitHub releases
and verify SHA256 hashes before restoring.

Note: This is a planned feature. For now, manually download snapshot
chunks and use 'lux network snapshot advanced restore' to restore.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("GitHub snapshot download is a planned feature.")
			ux.Logger.PrintToUser("For now, manually download snapshot chunks and use:")
			ux.Logger.PrintToUser("  lux network snapshot advanced restore <name>")
			return nil
		},
	}
	return cmd
}

func newAdvancedSnapshotUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <name>",
		Short: "Upload snapshot to GitHub",
		Long: `Upload a snapshot to GitHub releases.

This feature will upload chunked snapshot files (99MB each) to GitHub
releases for distribution.

Note: This is a planned feature. For now, manually upload the snapshot
chunks from ~/.lux/snapshots/<name>/.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("GitHub snapshot upload is a planned feature.")
			ux.Logger.PrintToUser("Snapshot location: ~/.lux/snapshots/%s", args[0])
			ux.Logger.PrintToUser("Manually upload the chunk files from the snapshot directory.")
			return nil
		},
	}
	return cmd
}

func createAdvancedSnapshot(cmd *cobra.Command, args []string, incremental bool) error {
	snapshotName := args[0]

	// Ensure network is stopped (because we use direct DB access in manager)
	// Or warn user
	ux.Logger.PrintToUser("Note: 'create' currently requires nodes to be stopped for DB access.")

	manager := snapshot.NewSnapshotManager(app.GetBaseDir())
	return manager.CreateSnapshot(snapshotName, incremental)
}

func restoreAdvancedSnapshot(cmd *cobra.Command, args []string) error {
	snapshotName := args[0]
	manager := snapshot.NewSnapshotManager(app.GetBaseDir())
	return manager.RestoreSnapshot(snapshotName)
}

func squashAdvancedSnapshot(cmd *cobra.Command, args []string) error {
	network := args[0]
	chainIDStr := args[1]
	snapshotName := args[2]

	chainID, err := strconv.ParseUint(chainIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chain ID: %w", err)
	}

	manager := snapshot.NewSnapshotManager(app.GetBaseDir())
	return manager.Squash(network, chainID, snapshotName)
}
