// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayercmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// lux interchain relayer config
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure Warp relayer",
		Long:  `Configure the Warp relayer for blockchain communication.`,
		RunE:  configRelayer,
		Args:  cobrautils.ExactArgs(0),
	}

	cmd.Flags().StringVar(&relayerConfigPath, "config-path", "", "Path to relayer config file")
	cmd.Flags().BoolVar(&addSourceBlockchain, "add-source", false, "Add a source blockchain")
	cmd.Flags().BoolVar(&addDestinationBlockchain, "add-destination", false, "Add a destination blockchain")
	cmd.Flags().StringVar(&blockchainID, "blockchain-id", "", "Blockchain ID to add")
	cmd.Flags().StringVar(&rpcEndpoint, "rpc-endpoint", "", "RPC endpoint for the blockchain")

	return cmd
}

var (
	relayerConfigPath        string
	addSourceBlockchain      bool
	addDestinationBlockchain bool
	blockchainID             string
	rpcEndpoint              string
)

// RelayerConfig represents the relayer configuration
type RelayerConfig struct {
	SourceBlockchains      []SourceBlockchain      `json:"sourceBlockchains"`
	DestinationBlockchains []DestinationBlockchain `json:"destinationBlockchains"`
}

// SourceBlockchain configuration
type SourceBlockchain struct {
	SubnetID     string    `json:"subnetId"`
	BlockchainID string    `json:"blockchainId"`
	VM           string    `json:"vm"`
	RPCEndpoint  APIConfig `json:"rpcEndpoint"`
}

// DestinationBlockchain configuration
type DestinationBlockchain struct {
	SubnetID     string    `json:"subnetId"`
	BlockchainID string    `json:"blockchainId"`
	VM           string    `json:"vm"`
	RPCEndpoint  APIConfig `json:"rpcEndpoint"`
}

// APIConfig for RPC endpoints
type APIConfig struct {
	BaseURL string `json:"baseUrl"`
}

func configRelayer(_ *cobra.Command, _ []string) error {
	// Load existing config or create new one
	configPath := relayerConfigPath
	if configPath == "" {
		configPath = filepath.Join(app.GetBaseDir(), "relayer", "config.json")
	}

	var config RelayerConfig
	if _, err := os.Stat(configPath); err == nil {
		// Load existing config
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Add source or destination blockchain
	if addSourceBlockchain {
		if blockchainID == "" || rpcEndpoint == "" {
			return fmt.Errorf("blockchain-id and rpc-endpoint are required when adding a blockchain")
		}

		source := SourceBlockchain{
			SubnetID:     blockchainID,
			BlockchainID: blockchainID,
			VM:           "evm",
			RPCEndpoint:  APIConfig{BaseURL: rpcEndpoint},
		}
		config.SourceBlockchains = append(config.SourceBlockchains, source)
		ux.Logger.PrintToUser("Added source blockchain: %s", blockchainID)
	}

	if addDestinationBlockchain {
		if blockchainID == "" || rpcEndpoint == "" {
			return fmt.Errorf("blockchain-id and rpc-endpoint are required when adding a blockchain")
		}

		dest := DestinationBlockchain{
			SubnetID:     blockchainID,
			BlockchainID: blockchainID,
			VM:           "evm",
			RPCEndpoint:  APIConfig{BaseURL: rpcEndpoint},
		}
		config.DestinationBlockchains = append(config.DestinationBlockchains, dest)
		ux.Logger.PrintToUser("Added destination blockchain: %s", blockchainID)
	}

	// Save config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	ux.Logger.PrintToUser("Relayer configuration saved to: %s", configPath)
	return nil
}
