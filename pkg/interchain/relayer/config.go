// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package relayer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/models"
)

// CreateBaseRelayerConfig creates a base configuration for the relayer
func CreateBaseRelayerConfig(logLevel string, storageLocation string, metricsPort uint32, network string) (map[string]interface{}, error) {
	config := map[string]interface{}{
		"logLevel":        logLevel,
		"storageLocation": storageLocation,
		"metricsPort":     metricsPort,
		"network":         network,
		"sources":         []interface{}{},
		"destinations":    []interface{}{},
	}
	return config, nil
}

// CreateBaseRelayerConfig creates a base configuration for the relayer and writes to file
func CreateBaseRelayerConfigFile(configPath string, logLevel string, storageLocation string, metricsPort uint16, network models.Network, allowPrivateIPs bool) error {
	config, err := CreateBaseRelayerConfig(logLevel, storageLocation, uint32(metricsPort), network.Name())
	if err != nil {
		return err
	}
	config["allowPrivateIPs"] = allowPrivateIPs
	return SaveRelayerConfig(configPath, config)
}

// AddSourceToRelayerConfig adds a source blockchain configuration
func AddSourceToRelayerConfig(
	config map[string]interface{},
	subnetID string,
	blockchainID string,
	rpcEndpoint string,
	wsEndpoint string,
	messageContractAddress string,
	rewardAddress string,
) error {
	sources, ok := config["sources"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid config structure: sources not found")
	}
	
	source := map[string]interface{}{
		"subnetID":               subnetID,
		"blockchainID":           blockchainID,
		"rpcEndpoint":            rpcEndpoint,
		"wsEndpoint":             wsEndpoint,
		"messageContractAddress": messageContractAddress,
		"rewardAddress":          rewardAddress,
	}
	
	config["sources"] = append(sources, source)
	return nil
}

// AddDestinationToRelayerConfig adds a destination blockchain configuration
func AddDestinationToRelayerConfig(
	config map[string]interface{},
	subnetID string,
	blockchainID string,
	rpcEndpoint string,
	accountPrivateKey string,
) error {
	destinations, ok := config["destinations"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid config structure: destinations not found")
	}
	
	destination := map[string]interface{}{
		"subnetID":          subnetID,
		"blockchainID":      blockchainID,
		"rpcEndpoint":       rpcEndpoint,
		"accountPrivateKey": accountPrivateKey,
	}
	
	config["destinations"] = append(destinations, destination)
	return nil
}

// SaveRelayerConfig saves the relayer configuration to a file
func SaveRelayerConfig(configPath string, config map[string]interface{}) error {
	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, configBytes, 0644)
}

// LoadRelayerConfig loads the relayer configuration from a file
func LoadRelayerConfig(configPath string) (map[string]interface{}, error) {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return config, nil
}

// AddSourceToRelayerConfigFile adds a source to the relayer config file
func AddSourceToRelayerConfigFile(
	configPath string,
	rpcEndpoint string,
	wsEndpoint string,
	subnetID string,
	blockchainID string,
	warpRegistryAddress string,
	warpMessengerAddress string,
	rewardAddress string,
) error {
	config, err := LoadRelayerConfig(configPath)
	if err != nil {
		return err
	}
	err = AddSourceToRelayerConfig(
		config,
		subnetID,
		blockchainID,
		rpcEndpoint,
		wsEndpoint,
		warpMessengerAddress,
		rewardAddress,
	)
	if err != nil {
		return err
	}
	return SaveRelayerConfig(configPath, config)
}

// AddDestinationToRelayerConfigFile adds a destination to the relayer config file
func AddDestinationToRelayerConfigFile(
	configPath string,
	rpcEndpoint string,
	subnetID string,
	blockchainID string,
	accountPrivateKey string,
) error {
	config, err := LoadRelayerConfig(configPath)
	if err != nil {
		return err
	}
	err = AddDestinationToRelayerConfig(
		config,
		subnetID,
		blockchainID,
		rpcEndpoint,
		accountPrivateKey,
	)
	if err != nil {
		return err
	}
	return SaveRelayerConfig(configPath, config)
}

// DeployRelayer deploys the relayer with the given configuration
func DeployRelayer(
	version string,
	binPath string,
	binDir string,
	configPath string,
	logPath string,
	runPath string,
	storageDir string,
) (string, error) {
	// TODO: Implement actual relayer deployment
	// This would download/use the relayer binary, write config, start the relayer process, etc.
	// For now, return the binary path that would be used
	if binPath != "" {
		return binPath, nil
	}
	return fmt.Sprintf("%s/warp-relayer-%s", binDir, version), nil
}