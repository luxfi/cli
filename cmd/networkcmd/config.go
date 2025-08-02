// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/v2/v2/pkg/constants"
	"github.com/mitchellh/go-homedir"
)

// NetworkConfig represents the configuration for a network
type NetworkConfig struct {
	NetworkID      uint32 `json:"networkId"`
	Endpoint       string `json:"endpoint"`
	DBType         string `json:"dbType"`
	StakingKeyPath string `json:"stakingKeyPath"`
	DataDir        string `json:"dataDir"`
	LogDir         string `json:"logDir"`
	DBPath         string `json:"dbPath"`
}

// LuxdConfig represents the unified configuration
type LuxdConfig struct {
	Home           string                       `json:"home"`
	DefaultNetwork string                       `json:"defaultNetwork"`
	Networks       map[string]NetworkConfig     `json:"networks"`
	Paths          map[string]string            `json:"paths"`
}

// GetLuxdHome returns the luxd home directory
func GetLuxdHome() (string, error) {
	if home := os.Getenv("LUXD_HOME"); home != "" {
		return home, nil
	}
	
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	
	return filepath.Join(homeDir, ".luxd"), nil
}

// LoadConfig loads the luxd configuration
func LoadConfig() (*LuxdConfig, error) {
	luxdHome, err := GetLuxdHome()
	if err != nil {
		return nil, err
	}
	
	configPath := filepath.Join(luxdHome, "configs", "default.json")
	
	// Check if config exists, if not initialize it
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := InitializeLuxdHome(); err != nil {
			return nil, fmt.Errorf("failed to initialize luxd home: %w", err)
		}
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	
	var config LuxdConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	return &config, nil
}

// InitializeLuxdHome creates the unified directory structure
func InitializeLuxdHome() error {
	luxdHome, err := GetLuxdHome()
	if err != nil {
		return err
	}
	
	// Create directory structure
	dirs := []string{
		filepath.Join(luxdHome, "networks", "mainnet"),
		filepath.Join(luxdHome, "networks", "testnet"),
		filepath.Join(luxdHome, "networks", "local"),
		filepath.Join(luxdHome, "staking-keys", "mainnet"),
		filepath.Join(luxdHome, "staking-keys", "testnet"),
		filepath.Join(luxdHome, "staking-keys", "local"),
		filepath.Join(luxdHome, "snapshots", "mainnet"),
		filepath.Join(luxdHome, "snapshots", "testnet"),
		filepath.Join(luxdHome, "snapshots", "local"),
		filepath.Join(luxdHome, "configs", "node"),
		filepath.Join(luxdHome, "configs", "chains"),
		filepath.Join(luxdHome, "configs", "subnets"),
		filepath.Join(luxdHome, "data", "db"),
		filepath.Join(luxdHome, "data", "chaindata"),
		filepath.Join(luxdHome, "logs"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	// Create default configuration
	config := LuxdConfig{
		Home:           luxdHome,
		DefaultNetwork: "mainnet",
		Networks: map[string]NetworkConfig{
			"mainnet": {
				NetworkID:      constants.MainnetID,
				Endpoint:       "https://api.lux.network",
				DBType:         "badgerdb",
				StakingKeyPath: filepath.Join(luxdHome, "staking-keys", "mainnet"),
				DataDir:        filepath.Join(luxdHome, "data", "mainnet"),
				LogDir:         filepath.Join(luxdHome, "logs", "mainnet"),
				DBPath:         filepath.Join(luxdHome, "data", "mainnet", "db"),
			},
			"testnet": {
				NetworkID:      constants.TestnetID,
				Endpoint:       "https://api.lux-test.network",
				DBType:         "badgerdb",
				StakingKeyPath: filepath.Join(luxdHome, "staking-keys", "testnet"),
				DataDir:        filepath.Join(luxdHome, "data", "testnet"),
				LogDir:         filepath.Join(luxdHome, "logs", "testnet"),
				DBPath:         filepath.Join(luxdHome, "data", "testnet", "db"),
			},
			"local": {
				NetworkID:      constants.LocalID,
				Endpoint:       "http://localhost:9650",
				DBType:         "badgerdb",
				StakingKeyPath: filepath.Join(luxdHome, "staking-keys", "local"),
				DataDir:        filepath.Join(luxdHome, "data", "local"),
				LogDir:         filepath.Join(luxdHome, "logs", "local"),
				DBPath:         filepath.Join(luxdHome, "data", "local", "db"),
			},
		},
		Paths: map[string]string{
			"stakingKeys": filepath.Join(luxdHome, "staking-keys"),
			"snapshots":   filepath.Join(luxdHome, "snapshots"),
			"data":        filepath.Join(luxdHome, "data"),
			"logs":        filepath.Join(luxdHome, "logs"),
		},
	}
	
	// Create network-specific configurations
	for network, netConfig := range config.Networks {
		netConfigPath := filepath.Join(luxdHome, "networks", network, "config.json")
		netConfigData, err := json.MarshalIndent(netConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal network config: %w", err)
		}
		if err := os.WriteFile(netConfigPath, netConfigData, 0644); err != nil {
			return fmt.Errorf("failed to write network config: %w", err)
		}
	}
	
	// Write main config
	configPath := filepath.Join(luxdHome, "configs", "default.json")
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	
	return nil
}

// GetNetworkConfig returns the configuration for a specific network
func GetNetworkConfig(network string) (*NetworkConfig, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	
	if network == "" {
		network = config.DefaultNetwork
	}
	
	netConfig, ok := config.Networks[network]
	if !ok {
		return nil, fmt.Errorf("unknown network: %s", network)
	}
	
	return &netConfig, nil
}