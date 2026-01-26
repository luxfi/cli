// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package kmscmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/luxfi/cli/pkg/kms"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	// Server flags
	serverAddr    string
	serverDataDir string
	serverAPIKey  string
	serverInMem   bool

	// Key flags
	keyName        string
	keyType        string
	keyUsage       string
	keyDescription string
	keyProjectID   string

	// Secret flags
	secretName        string
	secretValue       string
	secretEnvironment string
	secretPath        string
)

// NewCmd creates the kms command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kms",
		Short: "Key Management Service operations",
		Long: `Key Management Service (KMS) for managing cryptographic keys and secrets.

The KMS provides:
  - Key generation (AES-256, RSA, ECDSA, Ed25519)
  - Encryption/decryption operations
  - Digital signatures
  - Secret management
  - MPC wallet integration

QUICK START:

  # Start the KMS server
  lux kms server start

  # Generate a new key
  lux kms key create --name mykey --type aes-256-gcm --usage encrypt-decrypt

  # List keys
  lux kms key list

  # Create a secret
  lux kms secret create --name API_KEY --value "sk-xxx" --env production

STORAGE:

  KMS data is stored in ~/.lux/kms/ by default.
  The root encryption key is derived from your system keychain or environment.

API:

  The KMS server exposes a REST API compatible with the kms-go SDK.
  Default address: http://localhost:8200

Available subcommands:
  server  - Manage the KMS server
  key     - Key management operations
  secret  - Secret management operations`,
	}

	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newKeyCmd())
	cmd.AddCommand(newSecretCmd())

	return cmd
}

// newServerCmd creates the server management command group.
func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the KMS server",
		Long:  `Commands for starting and managing the KMS server.`,
	}

	cmd.AddCommand(newServerStartCmd())

	return cmd
}

func newServerStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the KMS server",
		Long: `Start the KMS HTTP API server.

The server provides a REST API for key management, encryption, and secret
operations. It is compatible with the kms-go SDK client.

Examples:
  # Start with default settings
  lux kms server start

  # Start on a custom port
  lux kms server start --addr :9200

  # Start with in-memory storage (for testing)
  lux kms server start --in-memory

  # Start with API key authentication
  lux kms server start --api-key your-secret-key`,
		RunE: runServerStart,
	}

	cmd.Flags().StringVar(&serverAddr, "addr", ":8200", "Server listen address")
	cmd.Flags().StringVar(&serverDataDir, "data-dir", "", "Data directory (default: ~/.lux/kms)")
	cmd.Flags().StringVar(&serverAPIKey, "api-key", "", "API key for authentication")
	cmd.Flags().BoolVar(&serverInMem, "in-memory", false, "Use in-memory storage (data lost on restart)")

	return cmd
}

func runServerStart(cmd *cobra.Command, args []string) error {
	// Determine data directory
	dataDir := serverDataDir
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".lux", "kms")
	}

	// Create data directory if it doesn't exist
	if !serverInMem {
		if err := os.MkdirAll(dataDir, 0700); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	// Get or generate root key
	rootKey, err := getRootKey(dataDir)
	if err != nil {
		return fmt.Errorf("failed to get root key: %w", err)
	}

	// Create KMS
	kmsConfig := &kms.Config{
		RootKey:     rootKey,
		DataDir:     dataDir,
		InMemory:    serverInMem,
		Compression: true,
	}

	kmsInstance, err := kms.New(kmsConfig)
	if err != nil {
		return fmt.Errorf("failed to create KMS: %w", err)
	}
	defer kmsInstance.Close()

	// Create server
	serverConfig := &kms.ServerConfig{
		Addr:          serverAddr,
		APIKey:        serverAPIKey,
		EnableMPC:     true,
		EnableSecrets: true,
	}

	server := kms.NewServer(kmsInstance, serverConfig)

	// Handle shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		ux.Logger.PrintToUser("Shutting down KMS server...")
		server.Stop(ctx)
	}()

	ux.Logger.PrintToUser("Starting KMS server on %s", serverAddr)
	if serverInMem {
		ux.Logger.PrintToUser("Using in-memory storage (data will be lost on restart)")
	} else {
		ux.Logger.PrintToUser("Data directory: %s", dataDir)
	}
	if serverAPIKey != "" {
		ux.Logger.PrintToUser("API key authentication enabled")
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("API Endpoints:")
	ux.Logger.PrintToUser("  Health: GET /health")
	ux.Logger.PrintToUser("  Keys:   POST /v1/kms/keys")
	ux.Logger.PrintToUser("  Secret: GET  /v3/secrets/raw")
	ux.Logger.PrintToUser("  MPC:    POST /v1/mpc/wallets")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Press Ctrl+C to stop")

	if err := server.Start(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// getRootKey gets or generates the root encryption key.
func getRootKey(dataDir string) ([]byte, error) {
	// Check environment variable first
	if envKey := os.Getenv("LUX_KMS_ROOT_KEY"); envKey != "" {
		key, err := hex.DecodeString(envKey)
		if err != nil {
			return nil, fmt.Errorf("invalid LUX_KMS_ROOT_KEY: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("LUX_KMS_ROOT_KEY must be 32 bytes (64 hex chars)")
		}
		return key, nil
	}

	// Check for existing key file
	keyFile := filepath.Join(dataDir, ".root_key")
	if data, err := os.ReadFile(keyFile); err == nil {
		key, err := hex.DecodeString(string(data))
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}

	// Generate new key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate root key: %w", err)
	}

	// Save key (create directory if needed)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	if err := os.WriteFile(keyFile, []byte(hex.EncodeToString(key)), 0600); err != nil {
		return nil, fmt.Errorf("failed to save root key: %w", err)
	}

	ux.Logger.PrintToUser("Generated new root encryption key")
	return key, nil
}

// newKeyCmd creates the key management command group.
func newKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Key management operations",
		Long:  `Commands for managing cryptographic keys.`,
	}

	cmd.AddCommand(newKeyCreateCmd())
	cmd.AddCommand(newKeyListCmd())
	cmd.AddCommand(newKeyDeleteCmd())

	return cmd
}

func newKeyCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new key",
		Long: `Create a new cryptographic key.

Supported key types:
  - aes-256-gcm   : Symmetric encryption (default)
  - rsa-4096      : RSA asymmetric key
  - ecdsa-p256    : ECDSA P-256 curve
  - ecdsa-p384    : ECDSA P-384 curve
  - ed25519       : EdDSA Ed25519

Usage types:
  - encrypt-decrypt : For encryption operations
  - sign-verify     : For digital signatures

Examples:
  lux kms key create --name mykey --type aes-256-gcm
  lux kms key create --name signing --type ecdsa-p256 --usage sign-verify`,
		RunE: runKeyCreate,
	}

	cmd.Flags().StringVar(&keyName, "name", "", "Key name (required)")
	cmd.Flags().StringVar(&keyType, "type", "aes-256-gcm", "Key type")
	cmd.Flags().StringVar(&keyUsage, "usage", "encrypt-decrypt", "Key usage")
	cmd.Flags().StringVar(&keyDescription, "description", "", "Key description")
	cmd.Flags().StringVar(&keyProjectID, "project", "", "Project ID")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runKeyCreate(cmd *cobra.Command, args []string) error {
	ux.Logger.PrintToUser("Key creation requires a running KMS server.")
	ux.Logger.PrintToUser("Start the server with: lux kms server start")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Then use the API:")
	ux.Logger.PrintToUser("  curl -X POST http://localhost:8200/v1/kms/keys \\")
	ux.Logger.PrintToUser("    -H 'Content-Type: application/json' \\")
	ux.Logger.PrintToUser("    -d '{\"name\":\"%s\",\"encryptionAlgorithm\":\"%s\",\"keyUsage\":\"%s\"}'", keyName, keyType, keyUsage)
	return nil
}

func newKeyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Key listing requires a running KMS server.")
			ux.Logger.PrintToUser("Use the API: curl http://localhost:8200/v1/kms/keys")
			return nil
		},
	}

	return cmd
}

func newKeyDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [keyID]",
		Short: "Delete a key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Key deletion requires a running KMS server.")
			ux.Logger.PrintToUser("Use the API: curl -X DELETE http://localhost:8200/v1/kms/keys/%s", args[0])
			return nil
		},
	}

	return cmd
}

// newSecretCmd creates the secret management command group.
func newSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Secret management operations",
		Long:  `Commands for managing encrypted secrets.`,
	}

	cmd.AddCommand(newSecretCreateCmd())
	cmd.AddCommand(newSecretListCmd())
	cmd.AddCommand(newSecretGetCmd())

	return cmd
}

func newSecretCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new secret",
		Long: `Create a new encrypted secret.

Examples:
  lux kms secret create --name API_KEY --value "sk-xxx"
  lux kms secret create --name DB_PASSWORD --value "secret" --env production`,
		RunE: runSecretCreate,
	}

	cmd.Flags().StringVar(&secretName, "name", "", "Secret name (required)")
	cmd.Flags().StringVar(&secretValue, "value", "", "Secret value (required)")
	cmd.Flags().StringVar(&secretEnvironment, "env", "", "Environment (dev, staging, prod)")
	cmd.Flags().StringVar(&secretPath, "path", "/", "Secret path")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("value")

	return cmd
}

func runSecretCreate(cmd *cobra.Command, args []string) error {
	ux.Logger.PrintToUser("Secret creation requires a running KMS server.")
	ux.Logger.PrintToUser("Start the server with: lux kms server start")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Then use the API:")
	ux.Logger.PrintToUser("  curl -X POST http://localhost:8200/v3/secrets/raw/%s \\", secretName)
	ux.Logger.PrintToUser("    -H 'Content-Type: application/json' \\")
	ux.Logger.PrintToUser("    -d '{\"secretValue\":\"%s\",\"environment\":\"%s\"}'", secretValue, secretEnvironment)
	return nil
}

func newSecretListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Secret listing requires a running KMS server.")
			ux.Logger.PrintToUser("Use the API: curl http://localhost:8200/v3/secrets/raw")
			return nil
		},
	}

	return cmd
}

func newSecretGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [secretName]",
		Short: "Get a secret value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Secret retrieval requires a running KMS server.")
			ux.Logger.PrintToUser("Use the API: curl 'http://localhost:8200/v3/secrets/raw/%s'", args[0])
			return nil
		},
	}

	return cmd
}
