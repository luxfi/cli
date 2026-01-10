// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/sdk/models"
)

// GetLocallyDeployedChainsFromFile reads the list of locally deployed chains from file.
func GetLocallyDeployedChainsFromFile(app *application.Lux) ([]string, error) {
	allChainDirs, err := os.ReadDir(app.GetChainsDir())
	if err != nil {
		return nil, err
	}

	deployedChains := []string{}

	for _, chainDir := range allChainDirs {
		if !chainDir.IsDir() {
			continue
		}
		// read sidecar file
		sc, err := app.LoadSidecar(chainDir.Name())
		if errors.Is(err, os.ErrNotExist) {
			// don't fail on missing sidecar file, just warn
			ux.Logger.PrintToUser("warning: inconsistent chain directory. No sidecar file found for chain %s", chainDir.Name())
			continue
		}
		if err != nil {
			return nil, err
		}

		// check if sidecar contains local deployment info in Networks map
		// if so, add to list of deployed chains
		if _, ok := sc.Networks[models.Local.String()]; ok {
			deployedChains = append(deployedChains, sc.Name)
		}
	}

	return deployedChains, nil
}

// GetLocallyDeployedChainIDs returns a list of chain IDs for locally deployed chains
// This is used for auto-tracking chains when starting the local network
// Deprecated: Use GetLocallyDeployedNetIDs instead
func GetLocallyDeployedChainIDs(app *application.Lux) ([]string, error) {
	return GetLocallyDeployedNetIDs(app)
}

// GetLocallyDeployedNetIDs returns a list of net IDs for locally deployed nets
// This is used for auto-tracking nets when starting the local network
func GetLocallyDeployedNetIDs(app *application.Lux) ([]string, error) {
	allChainDirs, err := os.ReadDir(app.GetChainsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	chainIDs := []string{}

	for _, chainDir := range allChainDirs {
		if !chainDir.IsDir() {
			continue
		}
		// read sidecar file
		sc, err := app.LoadSidecar(chainDir.Name())
		if err != nil {
			continue // skip on any error
		}

		// check if sidecar contains local deployment info with a valid SubnetID
		if network, ok := sc.Networks[models.Local.String()]; ok {
			if network.SubnetID.String() != "" && network.SubnetID.String() != PChainID {
				chainIDs = append(chainIDs, network.SubnetID.String())
			}
		}
	}

	return chainIDs, nil
}

// CopyChainChainConfigsToNetwork copies chain configs from ~/.lux/chains/<name>/ to each node's
// chainConfigs/<blockchainID>/ directory. This is necessary because evm requires genesis.json
// in the chain config directory for initialization.
// The canonical source is always ~/.lux/chains/<name>/ and this function ensures
// the running network nodes have access to these configs.
func CopyChainChainConfigsToNetwork(app *application.Lux, networkDir string) error {
	allChainDirs, err := os.ReadDir(app.GetChainsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Find all node directories in the network
	nodeDirs := []string{}
	entries, err := os.ReadDir(networkDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && (e.Name() == "node1" || e.Name() == "node2" || e.Name() == "node3" ||
			e.Name() == "node4" || e.Name() == "node5" || e.Name() == "node6" ||
			e.Name() == "node7" || e.Name() == "node8" || e.Name() == "node9" || e.Name() == "node10") {
			nodeDirs = append(nodeDirs, filepath.Join(networkDir, e.Name()))
		}
	}

	if len(nodeDirs) == 0 {
		return nil
	}

	copiedCount := 0
	for _, chainDir := range allChainDirs {
		if !chainDir.IsDir() {
			continue
		}

		chainName := chainDir.Name()
		sc, err := app.LoadSidecar(chainName)
		if err != nil {
			continue
		}

		// Get blockchain ID from Local Network deployment info
		network, ok := sc.Networks[models.Local.String()]
		if !ok {
			continue
		}

		blockchainID := network.BlockchainID.String()
		if blockchainID == "" || blockchainID == "11111111111111111111111111111111LpoYY" {
			continue
		}

		// Source files from canonical location
		chainConfigDir := filepath.Join(app.GetChainsDir(), chainName)
		genesisFile := filepath.Join(chainConfigDir, constants.GenesisFileName)
		chainConfigFile := filepath.Join(chainConfigDir, constants.ChainConfigFile)

		// Check if genesis exists (required for evm)
		if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
			continue
		}

		// Copy to each node's chainConfigs directory
		for _, nodeDir := range nodeDirs {
			destDir := filepath.Join(nodeDir, "chainConfigs", blockchainID)
			if err := os.MkdirAll(destDir, 0o750); err != nil {
				ux.Logger.PrintToUser("Warning: failed to create chain config dir for %s: %v", chainName, err)
				continue
			}

			// Copy genesis.json
			destGenesis := filepath.Join(destDir, "genesis.json")
			if err := copyFile(genesisFile, destGenesis); err != nil {
				ux.Logger.PrintToUser("Warning: failed to copy genesis for %s: %v", chainName, err)
				continue
			}

			// Copy chain.json as config.json if it exists
			if _, err := os.Stat(chainConfigFile); err == nil {
				destConfig := filepath.Join(destDir, "config.json")
				if err := copyFile(chainConfigFile, destConfig); err != nil {
					ux.Logger.PrintToUser("Warning: failed to copy chain config for %s: %v", chainName, err)
				}
			}
		}
		copiedCount++
	}

	if copiedCount > 0 {
		ux.Logger.PrintToUser("Copied chain configs for %d net(s) to network nodes", copiedCount)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src) //nolint:gosec // G304: Copying files within app's config directories
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst) //nolint:gosec // G304: Creating file within app's config directories
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// PrepareCanonicalChainConfigs creates the canonical chain configs directory at ~/.lux/chains/
// with subdirectories for each locally deployed chain's blockchain ID.
// This directory can be passed to nodes via --chain-config-dir flag so all nodes share
// the same chain configs from a single source.
// Returns the canonical chain configs directory path.
func PrepareCanonicalChainConfigs(app *application.Lux) (string, error) {
	// Use ChainsDir for all chain configs - consolidating chain-configs into chains/
	chainConfigsDir := app.GetChainConfigDir()
	if err := os.MkdirAll(chainConfigsDir, 0o750); err != nil {
		return "", err
	}

	allChainDirs, err := os.ReadDir(app.GetChainsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return chainConfigsDir, nil
		}
		return "", err
	}

	preparedCount := 0
	for _, chainDir := range allChainDirs {
		if !chainDir.IsDir() {
			continue
		}

		chainName := chainDir.Name()
		sc, err := app.LoadSidecar(chainName)
		if err != nil {
			continue
		}

		// Get blockchain ID from Local Network deployment info
		network, ok := sc.Networks[models.Local.String()]
		if !ok {
			continue
		}

		blockchainID := network.BlockchainID.String()
		if blockchainID == "" || blockchainID == "11111111111111111111111111111111LpoYY" {
			continue
		}

		// Source files from canonical chain location
		chainConfigDir := filepath.Join(app.GetChainsDir(), chainName)
		genesisFile := filepath.Join(chainConfigDir, constants.GenesisFileName)
		chainConfigFile := filepath.Join(chainConfigDir, constants.ChainConfigFile)

		// Check if genesis exists (required for evm)
		if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
			continue
		}

		// Create blockchain ID subdirectory
		blockchainDir := filepath.Join(chainConfigsDir, blockchainID)
		if err := os.MkdirAll(blockchainDir, 0o750); err != nil {
			ux.Logger.PrintToUser("Warning: failed to create chain config dir for %s: %v", chainName, err)
			continue
		}

		// Copy genesis.json
		destGenesis := filepath.Join(blockchainDir, "genesis.json")
		if err := copyFile(genesisFile, destGenesis); err != nil {
			ux.Logger.PrintToUser("Warning: failed to copy genesis for %s: %v", chainName, err)
			continue
		}

		// Create or update chain config with admin API enabled
		destConfig := filepath.Join(blockchainDir, "config.json")
		if err := writeChainConfig(chainConfigFile, destConfig); err != nil {
			ux.Logger.PrintToUser("Warning: failed to write chain config for %s: %v", chainName, err)
		}
		preparedCount++
	}

	if preparedCount > 0 {
		ux.Logger.PrintToUser("Prepared chain configs for %d net(s) in %s", preparedCount, chainConfigsDir)
	}

	return chainConfigsDir, nil
}

// writeChainConfig creates a chain config for a chain with admin API enabled
// If srcConfig exists, it merges admin settings into it; otherwise creates a default config
func writeChainConfig(srcConfig, destConfig string) error {
	config := map[string]interface{}{
		"eth-apis": []string{
			"eth", "eth-filter", "net", "web3",
			"internal-eth", "internal-blockchain", "internal-transaction", "internal-account",
			"admin",
		},
		"admin-api-enabled": true,
		"log-level":         "info",
	}

	// If source config exists, read and merge
	if _, err := os.Stat(srcConfig); err == nil {
		data, err := os.ReadFile(srcConfig) //nolint:gosec // G304: Reading from app's config directory
		if err == nil {
			var srcCfg map[string]interface{}
			if json.Unmarshal(data, &srcCfg) == nil {
				// Merge source config into our config (source takes precedence except for admin)
				for k, v := range srcCfg {
					if k != "eth-apis" && k != "admin-api-enabled" {
						config[k] = v
					}
				}
			}
		}
	}

	// Write the config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destConfig, data, 0o644) //nolint:gosec // G306: Config needs to be readable
}
