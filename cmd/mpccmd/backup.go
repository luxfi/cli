// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpccmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/cloud/storage"
	"github.com/luxfi/cli/pkg/mpc"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	backupDestination   string
	backupIncremental   bool
	backupCompression   string
	backupEncrypt       bool
	backupAgeRecipients []string
	backupAgeIdentities []string
	restoreVerifyOnly   bool
	restoreTargetPath   string
)

// newBackupCmd creates the backup command group.
func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage MPC node backups",
		Long: `Backup and restore MPC node data.

By default, backups are stored locally in ~/.lux/mpc/backups.
For cloud storage, specify a destination URI.

Supports multiple storage backends:
  - Local filesystem (default: ~/.lux/mpc/backups)
  - S3 (AWS, MinIO, Cloudflare R2, etc.)
  - GCS (Google Cloud Storage)
  - Azure Blob Storage

Examples:
  # Backup to default local directory (~/.lux/mpc/backups)
  lux mpc backup create

  # Backup to S3
  lux mpc backup create --destination s3://my-bucket/backups

  # Backup to custom local directory
  lux mpc backup create --destination file:///backups/mpc

  # List local backups (default)
  lux mpc backup list

  # List S3 backups
  lux mpc backup list --destination s3://my-bucket/backups

  # Restore from local backup
  lux mpc backup restore my-backup-20250125

  # Restore from S3
  lux mpc backup restore my-backup-20250125 --destination s3://my-bucket/backups`,
	}

	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupRestoreCmd())
	cmd.AddCommand(newBackupVerifyCmd())
	cmd.AddCommand(newBackupDeleteCmd())

	return cmd
}

func newBackupCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new backup",
		Long: `Create a backup of MPC node data.

The backup includes:
  - BadgerDB database
  - Key shares and wallet data
  - Node configuration

By default, backups are stored in ~/.lux/mpc/backups.
Backups are compressed with zstd by default and can be encrypted
with age encryption for secure storage.`,
		RunE: runBackupCreate,
	}

	cmd.Flags().StringVarP(&backupDestination, "destination", "d", "", "Storage destination (default: ~/.lux/mpc/backups)")
	cmd.Flags().BoolVar(&backupIncremental, "incremental", false, "Create incremental backup")
	cmd.Flags().StringVar(&backupCompression, "compression", "zstd", "Compression algorithm (zstd, gzip, none)")
	cmd.Flags().BoolVar(&backupEncrypt, "encrypt", false, "Encrypt backup with age")
	cmd.Flags().StringSliceVar(&backupAgeRecipients, "age-recipient", nil, "Age recipient public key(s)")

	return cmd
}

func newBackupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available backups",
		RunE:  runBackupList,
	}

	cmd.Flags().StringVarP(&backupDestination, "destination", "d", "", "Storage destination (default: ~/.lux/mpc/backups)")

	return cmd
}

func newBackupRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <backup-name>",
		Short: "Restore from a backup",
		Long: `Restore MPC node data from a backup.

This will stop the MPC node if running, restore the data,
and optionally restart the node.`,
		Args: cobra.ExactArgs(1),
		RunE: runBackupRestore,
	}

	cmd.Flags().StringVarP(&backupDestination, "destination", "d", "", "Storage destination (default: ~/.lux/mpc/backups)")
	cmd.Flags().StringVar(&restoreTargetPath, "target", "", "Target path (default: original location)")
	cmd.Flags().StringSliceVar(&backupAgeIdentities, "age-identity", nil, "Age identity file(s) for decryption")

	return cmd
}

func newBackupVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <backup-name>",
		Short: "Verify backup integrity",
		Long:  `Download and verify backup integrity without restoring.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runBackupVerify,
	}

	cmd.Flags().StringVarP(&backupDestination, "destination", "d", "", "Storage destination (default: ~/.lux/mpc/backups)")
	cmd.Flags().StringSliceVar(&backupAgeIdentities, "age-identity", nil, "Age identity file(s) for decryption")

	return cmd
}

func newBackupDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <backup-name>",
		Short: "Delete a backup",
		Args:  cobra.ExactArgs(1),
		RunE:  runBackupDelete,
	}

	cmd.Flags().StringVarP(&backupDestination, "destination", "d", "", "Storage destination (default: ~/.lux/mpc/backups)")

	return cmd
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse destination
	cfg, basePath, err := parseStorageDestination(backupDestination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	// Create storage client
	store, err := storage.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer store.Close()

	// Get MPC node info
	nodeID, nodeName, network, dbPath, err := getMpcNodeInfo()
	if err != nil {
		return fmt.Errorf("failed to get MPC node info: %w", err)
	}

	// Create backup manager
	bm := mpc.NewBackupManager(store, basePath, nodeID, nodeName, network)

	// Configure backup options
	opts := &mpc.BackupOptions{
		Incremental: backupIncremental,
		Compression: backupCompression,
		ProgressFunc: func(stage string, current, total int64) {
			ux.Logger.PrintToUser("  %s...", stage)
		},
	}

	if backupEncrypt && len(backupAgeRecipients) > 0 {
		opts.Encryption = &mpc.EncryptionInfo{
			Algorithm:  "age",
			Recipients: backupAgeRecipients,
		}
		opts.AgeRecipients = backupAgeRecipients
	}

	ux.Logger.PrintToUser("Creating backup for MPC node %s (%s)...", nodeName, network)

	manifest, err := bm.CreateBackup(ctx, dbPath, opts)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Determine display location
	location := backupDestination
	if location == "" {
		location, _ = getDefaultBackupPath()
		location = "~/.lux/mpc/backups"
	}

	ux.Logger.PrintToUser("\nBackup created successfully!")
	ux.Logger.PrintToUser("  Timestamp: %s", manifest.Timestamp.Format(time.RFC3339))
	ux.Logger.PrintToUser("  Checksum:  %s", manifest.Checksums["data"])
	ux.Logger.PrintToUser("  Location:  %s", location)

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, basePath, err := parseStorageDestination(backupDestination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	store, err := storage.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer store.Close()

	nodeID, nodeName, network, _, err := getMpcNodeInfo()
	if err != nil {
		return fmt.Errorf("failed to get MPC node info: %w", err)
	}

	bm := mpc.NewBackupManager(store, basePath, nodeID, nodeName, network)

	manifests, err := bm.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(manifests) == 0 {
		ux.Logger.PrintToUser("No backups found")
		return nil
	}

	ux.Logger.PrintToUser("Available backups:\n")
	ux.Logger.PrintToUser("%-40s  %-20s  %-12s  %-10s", "NAME", "TIMESTAMP", "TYPE", "ENCRYPTED")
	ux.Logger.PrintToUser("%s", strings.Repeat("-", 90))

	for _, m := range manifests {
		backupType := "full"
		if m.Incremental {
			backupType = "incremental"
		}
		encrypted := "no"
		if m.Encryption != nil {
			encrypted = m.Encryption.Algorithm
		}
		name := fmt.Sprintf("%s_%s_%s", m.NodeID, m.Network, m.Timestamp.Format("20060102-150405"))
		ux.Logger.PrintToUser("%-40s  %-20s  %-12s  %-10s",
			name,
			m.Timestamp.Format("2006-01-02 15:04:05"),
			backupType,
			encrypted,
		)
	}

	return nil
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	backupName := args[0]

	cfg, basePath, err := parseStorageDestination(backupDestination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	store, err := storage.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer store.Close()

	nodeID, nodeName, network, _, err := getMpcNodeInfo()
	if err != nil {
		return fmt.Errorf("failed to get MPC node info: %w", err)
	}

	bm := mpc.NewBackupManager(store, basePath, nodeID, nodeName, network)

	ux.Logger.PrintToUser("Restoring backup %s...", backupName)

	opts := &mpc.RestoreOptions{
		TargetPath:    restoreTargetPath,
		AgeIdentities: backupAgeIdentities,
		ProgressFunc: func(stage string, current, total int64) {
			ux.Logger.PrintToUser("  %s...", stage)
		},
	}

	if err := bm.RestoreBackup(ctx, backupName, opts); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	ux.Logger.PrintToUser("\nBackup restored successfully!")

	return nil
}

func runBackupVerify(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	backupName := args[0]

	cfg, basePath, err := parseStorageDestination(backupDestination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	store, err := storage.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer store.Close()

	nodeID, nodeName, network, _, err := getMpcNodeInfo()
	if err != nil {
		return fmt.Errorf("failed to get MPC node info: %w", err)
	}

	bm := mpc.NewBackupManager(store, basePath, nodeID, nodeName, network)

	ux.Logger.PrintToUser("Verifying backup %s...", backupName)

	opts := &mpc.RestoreOptions{
		VerifyOnly:    true,
		AgeIdentities: backupAgeIdentities,
		ProgressFunc: func(stage string, current, total int64) {
			ux.Logger.PrintToUser("  %s...", stage)
		},
	}

	if err := bm.RestoreBackup(ctx, backupName, opts); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	ux.Logger.PrintToUser("\nBackup verified successfully!")

	return nil
}

func runBackupDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	backupName := args[0]

	cfg, basePath, err := parseStorageDestination(backupDestination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	store, err := storage.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer store.Close()

	nodeID, nodeName, network, _, err := getMpcNodeInfo()
	if err != nil {
		return fmt.Errorf("failed to get MPC node info: %w", err)
	}

	bm := mpc.NewBackupManager(store, basePath, nodeID, nodeName, network)

	ux.Logger.PrintToUser("Deleting backup %s...", backupName)

	if err := bm.DeleteBackup(ctx, backupName); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	ux.Logger.PrintToUser("Backup deleted successfully!")

	return nil
}

// Helper functions

// getDefaultBackupPath returns the default local backup directory
func getDefaultBackupPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir + "/.lux/mpc/backups", nil
}

func parseStorageDestination(dest string) (*storage.Config, string, error) {
	// Use default local path if no destination specified
	if dest == "" {
		defaultPath, err := getDefaultBackupPath()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get default backup path: %w", err)
		}
		// Ensure directory exists
		if err := os.MkdirAll(defaultPath, 0750); err != nil {
			return nil, "", fmt.Errorf("failed to create backup directory: %w", err)
		}
		return &storage.Config{
			Provider:      storage.ProviderLocal,
			LocalBasePath: defaultPath,
		}, "", nil
	}

	cfg, key, err := storage.ParseURI(dest)
	if err != nil {
		return nil, "", err
	}

	// Load credentials from environment
	switch cfg.Provider {
	case storage.ProviderS3:
		cfg.AWSAccessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		cfg.AWSSecretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		cfg.AWSSessionToken = os.Getenv("AWS_SESSION_TOKEN")
		cfg.Region = os.Getenv("AWS_REGION")
		if cfg.Region == "" {
			cfg.Region = os.Getenv("AWS_DEFAULT_REGION")
		}
		if cfg.Region == "" {
			cfg.Region = "us-east-1"
		}
		// Check for S3-compatible endpoints
		if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
			cfg.Endpoint = endpoint
			cfg.PathStyle = true
		}
	case storage.ProviderGCS:
		cfg.GCSCredentialsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		cfg.GCSProjectID = os.Getenv("GCP_PROJECT_ID")
	}

	return cfg, key, nil
}

func getMpcNodeInfo() (nodeID, nodeName, network, dbPath string, err error) {
	// Read from MPC node configuration
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", "", err
	}

	mpcDir := os.Getenv("MPC_DATA_DIR")
	if mpcDir == "" {
		mpcDir = homeDir + "/.lux/mpc"
	}

	// Find the most recent network
	mgr := mpc.NewNodeManager(mpcDir)
	networks, err := mgr.ListNetworks()
	if err != nil || len(networks) == 0 {
		return "", "", "", "", fmt.Errorf("no MPC networks found - initialize one with 'lux mpc node init'")
	}

	// Use the most recent network
	net := networks[len(networks)-1]
	nodeID = net.NetworkID
	nodeName = net.NetworkName
	network = net.NetworkType
	dbPath = net.BaseDir // Backup the entire network directory

	return nodeID, nodeName, network, dbPath, nil
}
