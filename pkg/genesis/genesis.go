// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package genesis provides canonical genesis configurations for Lux networks.
// It embeds the canonical configs and also supports loading from external paths.
package genesis

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed configs/*.json
var embeddedConfigs embed.FS

// GenesisRepo is the default path to the genesis repo for external loading
const GenesisRepo = "~/work/lux/genesis/configs"

// Network identifiers
const (
	Mainnet     = "mainnet"
	Testnet     = "testnet"
	Devnet      = "devnet"
	ZooMainnet  = "zoo-mainnet"
	ZooTestnet  = "zoo-testnet"
	SPCMainnet  = "spc-mainnet"
	SPCTestnet  = "spc-testnet"
)

// Canonical genesis hashes - these MUST match for imports to work
var CanonicalGenesisHashes = map[string]string{
	Mainnet:    "0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e",
	Testnet:    "0x1c5fe37764b8bc146dc88bc1c2e0259cd8369b07a06439bcfa1782b5d4fb0995",
	ZooMainnet: "0x7c548af47de27560779ccc67dda32a540944accc71dac3343da3b9cd18f14933",
	ZooTestnet: "0x0652fb2fde1460544a5893e5eba5095ff566861cbc87fcb1c73be2b81d6d1979",
	SPCMainnet: "0x4dc9fd5cf4ee64609f140ba0aa50f320cadf0ae8b59a29415979bc05b17cfac8",
}

// GetCChainGenesis returns the canonical C-Chain genesis for a network.
// It first tries to load from the external genesis repo, then falls back to embedded.
func GetCChainGenesis(network string) (string, error) {
	// Try external path first (allows updates without rebuilding CLI)
	externalPath := expandPath(GenesisRepo)
	var configFile string

	switch network {
	case Mainnet:
		configFile = filepath.Join(externalPath, "mainnet", "cchain.json")
	case Testnet:
		configFile = filepath.Join(externalPath, "testnet", "cchain.json")
	case Devnet:
		configFile = filepath.Join(externalPath, "devnet", "cchain.json")
	default:
		return "", fmt.Errorf("unknown network: %s", network)
	}

	// Try external file
	if data, err := os.ReadFile(configFile); err == nil {
		return string(data), nil
	}

	// Fall back to embedded
	return getEmbeddedCChainGenesis(network)
}

// GetSubnetGenesis returns the canonical genesis for a subnet chain.
func GetSubnetGenesis(network string) (string, error) {
	externalPath := expandPath(GenesisRepo)
	var configFile string

	switch network {
	case ZooMainnet:
		configFile = filepath.Join(externalPath, "zoo-mainnet", "genesis.json")
	case ZooTestnet:
		configFile = filepath.Join(externalPath, "zoo-testnet", "genesis.json")
	case SPCMainnet:
		configFile = filepath.Join(externalPath, "spc-mainnet", "genesis.json")
	case SPCTestnet:
		configFile = filepath.Join(externalPath, "spc-testnet", "genesis.json")
	default:
		return "", fmt.Errorf("unknown subnet: %s", network)
	}

	// Try external file
	if data, err := os.ReadFile(configFile); err == nil {
		return string(data), nil
	}

	// Fall back to embedded
	return getEmbeddedSubnetGenesis(network)
}

// GetCChainGenesisJSON returns the C-Chain genesis as a parsed map.
func GetCChainGenesisJSON(network string) (map[string]interface{}, error) {
	genesis, err := GetCChainGenesis(network)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(genesis), &result); err != nil {
		return nil, fmt.Errorf("failed to parse genesis: %w", err)
	}
	return result, nil
}

// GetCanonicalGenesisHash returns the expected genesis hash for a network.
func GetCanonicalGenesisHash(network string) (string, bool) {
	hash, ok := CanonicalGenesisHashes[network]
	return hash, ok
}

// getEmbeddedCChainGenesis loads from embedded configs.
func getEmbeddedCChainGenesis(network string) (string, error) {
	var filename string
	switch network {
	case Mainnet:
		filename = "configs/mainnet-cchain.json"
	case Testnet:
		filename = "configs/testnet-cchain.json"
	case Devnet:
		filename = "configs/devnet-cchain.json"
	default:
		return "", fmt.Errorf("no embedded config for network: %s", network)
	}

	data, err := embeddedConfigs.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded config %s: %w", filename, err)
	}
	return string(data), nil
}

// getEmbeddedSubnetGenesis loads subnet genesis from embedded configs.
func getEmbeddedSubnetGenesis(network string) (string, error) {
	var filename string
	switch network {
	case ZooMainnet:
		filename = "configs/zoo-mainnet-genesis.json"
	case ZooTestnet:
		filename = "configs/zoo-testnet-genesis.json"
	default:
		return "", fmt.Errorf("no embedded config for subnet: %s", network)
	}

	data, err := embeddedConfigs.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded config %s: %w", filename, err)
	}
	return string(data), nil
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

// ValidateGenesisHash checks if the actual genesis hash matches the canonical one.
func ValidateGenesisHash(network, actualHash string) error {
	expected, ok := CanonicalGenesisHashes[network]
	if !ok {
		return nil // No canonical hash defined, skip validation
	}
	if actualHash != expected {
		return fmt.Errorf("genesis hash mismatch for %s: expected %s, got %s", network, expected, actualHash)
	}
	return nil
}
