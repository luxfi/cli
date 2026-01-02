// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

const networkTypeCustom = "custom"

var snapshotNetworkType string

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage network snapshots",
		Long: `The snapshot command allows you to save, load, list, and delete snapshots of your local network state.

Snapshots capture the entire network state including all node data, databases, and configurations.
This is useful for:
  - Creating backpoints before major changes
  - Sharing network states across development environments
  - Testing different scenarios from the same initial state

Commands:
  save <name>   - Save current network state as a named snapshot
  load <name>   - Load a snapshot and restart the network
  list          - List all available snapshots
  delete <name> - Delete a snapshot

Examples:
  lux network snapshot save my-test-state
  lux network snapshot list
  lux network snapshot load my-test-state
  lux network snapshot delete my-test-state`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newSnapshotSaveCmd())
	cmd.AddCommand(newSnapshotLoadCmd())
	cmd.AddCommand(newSnapshotListCmd())
	cmd.AddCommand(newSnapshotDeleteCmd())

	return cmd
}

func newSnapshotSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save current network state as a named snapshot",
		Long: `The snapshot save command saves the current network state to a named snapshot.

The snapshot includes all node data, databases, and configurations from the current network.
The network must be stopped before creating a snapshot.

Example:
  lux network snapshot save my-test-state`,
		Args:         cobra.ExactArgs(1),
		RunE:         saveSnapshot,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&snapshotNetworkType, "network-type", "", "network type to snapshot (mainnet, testnet, devnet, custom)")

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
	for _, netType := range []string{networkTypeCustom, "devnet", "testnet", "mainnet"} {
		state, err := app.LoadNetworkStateForType(netType)
		if err == nil && state != nil && state.Running {
			return netType
		}
	}

	// Default to custom
	return networkTypeCustom
}

func saveSnapshot(cmd *cobra.Command, args []string) error {
	snapshotName := args[0]

	// Validate snapshot name
	if strings.ContainsAny(snapshotName, "/\\:*?\"<>|") {
		return fmt.Errorf("invalid snapshot name: cannot contain special characters /\\:*?\"<>|")
	}

	networkType := determineNetworkType()
	ux.Logger.PrintToUser("Saving snapshot for network: %s", networkType)

	// Check if network is running
	state, err := app.LoadNetworkStateForType(networkType)
	if err == nil && state != nil && state.Running {
		return fmt.Errorf("network %s is currently running. Please stop it first with 'lux network stop --network-type %s'", networkType, networkType)
	}

	// Get source directory (the current network state)
	runDir := app.GetRunDir()
	sourceDir := filepath.Join(runDir, networkType)

	// Check if source exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("no network data found for %s. Start a network first", networkType)
	}

	// Get snapshots directory
	snapshotsDir := app.GetSnapshotsDir()
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create snapshots directory: %w", err)
	}

	// Create snapshot directory
	snapshotDir := filepath.Join(snapshotsDir, snapshotName)
	if _, err := os.Stat(snapshotDir); err == nil {
		return fmt.Errorf("snapshot '%s' already exists. Delete it first or choose a different name", snapshotName)
	}

	ux.Logger.PrintToUser("Creating snapshot: %s", snapshotName)
	ux.Logger.PrintToUser("Source: %s", sourceDir)
	ux.Logger.PrintToUser("Destination: %s", snapshotDir)

	// Copy the directory
	if err := copyDirectory(sourceDir, snapshotDir); err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Save metadata
	metadata := map[string]string{
		"name":         snapshotName,
		"network_type": networkType,
		"created_at":   time.Now().Format(time.RFC3339),
		"source":       sourceDir,
	}

	metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.txt")
	metadataContent := fmt.Sprintf("Name: %s\nNetwork Type: %s\nCreated: %s\nSource: %s\n",
		metadata["name"],
		metadata["network_type"],
		metadata["created_at"],
		metadata["source"])

	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0o644); err != nil {
		ux.Logger.PrintToUser("Warning: failed to save metadata: %v", err)
	}

	ux.Logger.PrintToUser("✓ Snapshot '%s' created successfully", snapshotName)
	return nil
}

func loadSnapshot(cmd *cobra.Command, args []string) error {
	snapshotName := args[0]

	networkType := determineNetworkType()
	ux.Logger.PrintToUser("Loading snapshot for network: %s", networkType)

	// Get snapshot directory
	snapshotsDir := app.GetSnapshotsDir()
	snapshotDir := filepath.Join(snapshotsDir, snapshotName)

	// Check if snapshot exists
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	// Check if network is running
	state, err := app.LoadNetworkStateForType(networkType)
	if err == nil && state != nil && state.Running {
		ux.Logger.PrintToUser("Stopping running network...")
		if err := StopNetwork(nil, nil); err != nil {
			return fmt.Errorf("failed to stop network: %w", err)
		}
		// Wait a bit for graceful shutdown
		time.Sleep(2 * time.Second)
	}

	// Get destination directory
	runDir := app.GetRunDir()
	destDir := filepath.Join(runDir, networkType)

	// Backup existing data if it exists
	if _, err := os.Stat(destDir); err == nil {
		backupDir := destDir + ".backup." + time.Now().Format("20060102-150405")
		ux.Logger.PrintToUser("Backing up existing data to: %s", backupDir)
		if err := os.Rename(destDir, backupDir); err != nil {
			return fmt.Errorf("failed to backup existing data: %w", err)
		}
	}

	ux.Logger.PrintToUser("Loading snapshot: %s", snapshotName)
	ux.Logger.PrintToUser("Source: %s", snapshotDir)
	ux.Logger.PrintToUser("Destination: %s", destDir)

	// Copy snapshot to run directory
	if err := copyDirectory(snapshotDir, destDir); err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Remove metadata file from destination (it's not part of the network data)
	metadataPath := filepath.Join(destDir, "snapshot_metadata.txt")
	_ = os.Remove(metadataPath)

	ux.Logger.PrintToUser("✓ Snapshot '%s' loaded successfully", snapshotName)
	ux.Logger.PrintToUser("\nTo start the network, run:")
	ux.Logger.PrintToUser("  lux network start --%s", networkType)

	return nil
}

func listSnapshots(cmd *cobra.Command, args []string) error {
	snapshotsDir := app.GetSnapshotsDir()

	// Check if snapshots directory exists
	if _, err := os.Stat(snapshotsDir); os.IsNotExist(err) {
		ux.Logger.PrintToUser("No snapshots found. Create one with 'lux network snapshot save <name>'")
		return nil
	}

	// Read directory
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	// Filter directories only
	var snapshots []string
	for _, entry := range entries {
		if entry.IsDir() {
			snapshots = append(snapshots, entry.Name())
		}
	}

	if len(snapshots) == 0 {
		ux.Logger.PrintToUser("No snapshots found. Create one with 'lux network snapshot save <name>'")
		return nil
	}

	ux.Logger.PrintToUser("Available snapshots:\n")

	for _, name := range snapshots {
		snapshotDir := filepath.Join(snapshotsDir, name)

		// Try to read metadata
		metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.txt")
		metadataBytes, err := os.ReadFile(metadataPath)

		if err == nil {
			// Print metadata
			ux.Logger.PrintToUser("Snapshot: %s", name)
			ux.Logger.PrintToUser("%s", string(metadataBytes))

			// Calculate size
			size, _ := getDirSize(snapshotDir)
			ux.Logger.PrintToUser("Size: %s\n", formatBytes(size))
		} else {
			// No metadata, just print name and basic info
			info, _ := os.Stat(snapshotDir)
			ux.Logger.PrintToUser("Snapshot: %s", name)
			if info != nil {
				ux.Logger.PrintToUser("Modified: %s", info.ModTime().Format(time.RFC3339))
			}
			size, _ := getDirSize(snapshotDir)
			ux.Logger.PrintToUser("Size: %s\n", formatBytes(size))
		}
	}

	return nil
}

func deleteSnapshot(cmd *cobra.Command, args []string) error {
	snapshotName := args[0]

	// Validate snapshot name to prevent path traversal attacks
	if snapshotName == "" || snapshotName == "." || snapshotName == ".." || filepath.Base(snapshotName) != snapshotName {
		return fmt.Errorf("invalid snapshot name: %s", snapshotName)
	}

	snapshotsDir := app.GetSnapshotsDir()
	snapshotDir := filepath.Join(snapshotsDir, snapshotName)

	// Check if snapshot exists
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	// Safety check: ensure we're only deleting within the snapshots directory
	absSnapshotDir, err := filepath.Abs(snapshotDir)
	if err != nil {
		return fmt.Errorf("failed to resolve snapshot path: %w", err)
	}
	absSnapshotsDir, err := filepath.Abs(snapshotsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve snapshots directory: %w", err)
	}

	// Verify that snapshotDir is directly inside snapshotsDir
	if filepath.Dir(absSnapshotDir) != absSnapshotsDir {
		return fmt.Errorf("SAFETY: snapshot directory must be directly inside snapshots directory")
	}

	ux.Logger.PrintToUser("Deleting snapshot: %s", snapshotName)

	// Use SafeRemoveAll to ensure we don't accidentally delete protected directories
	if err := app.SafeRemoveAll(snapshotDir); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	ux.Logger.PrintToUser("Snapshot '%s' deleted successfully", snapshotName)
	return nil
}

// Helper function to calculate directory size
func getDirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// Helper function to format bytes as human-readable size
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
