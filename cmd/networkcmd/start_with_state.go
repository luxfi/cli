// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// Known blockchain IDs
const (
	// MainnetSubnetBlockchainID is the known blockchain ID for the mainnet subnet
	MainnetSubnetBlockchainID = "2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"
)

var (
	// New flags for existing state support
	subnetStatePath string
	subnetID        string
	blockchainID    string
	statePath       string
	importChainData string
)

// LoadExistingSubnetState loads an existing subnet database into the network
func LoadExistingSubnetState(networkDir string) error {
	// Check for default existing state if no paths specified
	if subnetStatePath == "" && statePath == "" {
		// Check for default mainnet-regenesis database
		defaultPath := filepath.Join(os.Getenv("HOME"), ".lux-cli", "runs", "mainnet-regenesis", "node1", "chains", "2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB", "db")
		info, err := os.Stat(defaultPath)
		if err != nil || !info.IsDir() {
			return nil // No existing state to load
		}
		ux.Logger.PrintToUser("Found existing mainnet-regenesis database at default location")
		subnetStatePath = defaultPath
		blockchainID = MainnetSubnetBlockchainID
	}

	// Determine which path to use
	pathToUse := subnetStatePath
	if pathToUse == "" {
		pathToUse = statePath
	}

	// Check if the state path exists
	if _, err := os.Stat(pathToUse); os.IsNotExist(err) {
		return fmt.Errorf("state path does not exist: %s", pathToUse)
	}

	// If it's a directory in ~/work/lux/state/chaindata, handle it specially
	expectedPrefix := filepath.Join(os.Getenv("HOME"), "work", "lux", "state", "chaindata")
	if strings.HasPrefix(pathToUse, expectedPrefix) {
		return loadStateFromChaindata(pathToUse, networkDir)
	}

	// For subnet databases, we need the blockchain ID
	if blockchainID == "" {
		// Try to detect it from known subnet configurations
		blockchainID = detectBlockchainID(pathToUse)
		if blockchainID == "" {
			return fmt.Errorf("blockchain ID not provided and could not be detected")
		}
	}

	// Target directory for the subnet database
	targetDir := filepath.Join(networkDir, "node1", "data", "chains", blockchainID, "db")

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o750); err != nil {
		return fmt.Errorf("failed to create target directory structure: %w", err)
	}

	// Copy the database
	ux.Logger.PrintToUser("Loading existing subnet state from %s", pathToUse)
	if err := copyDirectory(pathToUse, targetDir); err != nil {
		return fmt.Errorf("failed to copy subnet database: %w", err)
	}

	ux.Logger.PrintToUser("Successfully loaded existing subnet state for blockchain %s", blockchainID)
	return nil
}

// loadStateFromChaindata handles loading state from the ~/work/lux/state/chaindata directory
func loadStateFromChaindata(chainDataPath string, networkDir string) error {
	// Check if it's a network-specific directory like lux-mainnet-96369
	dbPath := filepath.Join(chainDataPath, "db")
	if _, err := os.Stat(dbPath); err == nil {
		// This is a chain data directory with a db subdirectory
		// Extract blockchain ID from metadata if available
		metadataPath := filepath.Join(chainDataPath, "metadata.json")
		if _, err := os.Stat(metadataPath); err == nil {
			// Parse metadata to get blockchain ID
			// For now, we'll use a known mapping
			if blockchainID == "" {
				blockchainID = MainnetSubnetBlockchainID // Known subnet blockchain ID
			}
		}

		targetDir := filepath.Join(networkDir, "node1", "data", "chains", blockchainID, "db")
		if err := os.MkdirAll(filepath.Dir(targetDir), 0o750); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}

		ux.Logger.PrintToUser("Loading chain state from %s", dbPath)
		if err := copyDirectory(dbPath, targetDir); err != nil {
			return fmt.Errorf("failed to copy chain database: %w", err)
		}

		ux.Logger.PrintToUser("Successfully loaded chain state for blockchain %s", blockchainID)
		return nil
	}

	return fmt.Errorf("invalid state directory structure: %s", chainDataPath)
}

// detectBlockchainID attempts to detect the blockchain ID from the path or database
func detectBlockchainID(dbPath string) string {
	// Check if the path contains a known blockchain ID
	knownIDs := map[string]string{
		MainnetSubnetBlockchainID:                            "LUX Mainnet Subnet",
		"2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG": "LUX Subnet",
	}

	for id := range knownIDs {
		if filepath.Base(filepath.Dir(dbPath)) == id {
			return id
		}
	}

	// Default to the known mainnet subnet ID if not detected
	return MainnetSubnetBlockchainID
}

// copyDirectory recursively copies a directory from src to dst
func copyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory with same permissions
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src) //nolint:gosec // G304: Copying files within app's directories
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode()) //nolint:gosec // G304: Writing to app's directory
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// AddStateFlags adds the state-related flags to the command
func AddStateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&subnetStatePath, "subnet-state-path", "", "path to existing subnet database to load")
	cmd.Flags().StringVar(&statePath, "state-path", "", "path to existing state directory (e.g., ~/work/lux/state/chaindata/lux-mainnet-96369)")
	cmd.Flags().StringVar(&subnetID, "subnet-id", "", "subnet ID for the loaded state")
	cmd.Flags().StringVar(&blockchainID, "blockchain-id", "", "blockchain ID for the loaded state")
	cmd.Flags().StringVar(&importChainData, "import-chain-data", "", "path to import blockchain data from another chain into C-Chain")
}
