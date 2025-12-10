// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package net

import (
	"io"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
)

func GetLocallyDeployedSubnetsFromFile(app *application.Lux) ([]string, error) {
	allSubnetDirs, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		return nil, err
	}

	deployedSubnets := []string{}

	for _, subnetDir := range allSubnetDirs {
		if !subnetDir.IsDir() {
			continue
		}
		// read sidecar file
		sc, err := app.LoadSidecar(subnetDir.Name())
		if err == os.ErrNotExist {
			// don't fail on missing sidecar file, just warn
			ux.Logger.PrintToUser("warning: inconsistent subnet directory. No sidecar file found for subnet %s", subnetDir.Name())
			continue
		}
		if err != nil {
			return nil, err
		}

		// check if sidecar contains local deployment info in Networks map
		// if so, add to list of deployed subnets
		if _, ok := sc.Networks[models.Local.String()]; ok {
			deployedSubnets = append(deployedSubnets, sc.Name)
		}
	}

	return deployedSubnets, nil
}

// GetLocallyDeployedSubnetIDs returns a list of subnet IDs for locally deployed subnets
// This is used for auto-tracking subnets when starting the local network
// Deprecated: Use GetLocallyDeployedNetIDs instead
func GetLocallyDeployedSubnetIDs(app *application.Lux) ([]string, error) {
	return GetLocallyDeployedNetIDs(app)
}

// GetLocallyDeployedNetIDs returns a list of net IDs for locally deployed nets
// This is used for auto-tracking nets when starting the local network
func GetLocallyDeployedNetIDs(app *application.Lux) ([]string, error) {
	allSubnetDirs, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	subnetIDs := []string{}

	for _, subnetDir := range allSubnetDirs {
		if !subnetDir.IsDir() {
			continue
		}
		// read sidecar file
		sc, err := app.LoadSidecar(subnetDir.Name())
		if err != nil {
			continue // skip on any error
		}

		// check if sidecar contains local deployment info with a valid SubnetID
		if network, ok := sc.Networks[models.Local.String()]; ok {
			if network.SubnetID.String() != "" && network.SubnetID.String() != "11111111111111111111111111111111LpoYY" {
				subnetIDs = append(subnetIDs, network.SubnetID.String())
			}
		}
	}

	return subnetIDs, nil
}

// CopySubnetChainConfigsToNetwork copies chain configs from ~/.lux/subnets/<name>/ to each node's
// chainConfigs/<blockchainID>/ directory. This is necessary because subnet-evm requires genesis.json
// in the chain config directory for initialization.
// The canonical source is always ~/.lux/subnets/<name>/ and this function ensures
// the running network nodes have access to these configs.
func CopySubnetChainConfigsToNetwork(app *application.Lux, networkDir string) error {
	allSubnetDirs, err := os.ReadDir(app.GetSubnetDir())
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
	for _, subnetDir := range allSubnetDirs {
		if !subnetDir.IsDir() {
			continue
		}

		subnetName := subnetDir.Name()
		sc, err := app.LoadSidecar(subnetName)
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
		subnetConfigDir := filepath.Join(app.GetSubnetDir(), subnetName)
		genesisFile := filepath.Join(subnetConfigDir, constants.GenesisFileName)
		chainConfigFile := filepath.Join(subnetConfigDir, constants.ChainConfigFileName)

		// Check if genesis exists (required for subnet-evm)
		if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
			continue
		}

		// Copy to each node's chainConfigs directory
		for _, nodeDir := range nodeDirs {
			destDir := filepath.Join(nodeDir, "chainConfigs", blockchainID)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				ux.Logger.PrintToUser("Warning: failed to create chain config dir for %s: %v", subnetName, err)
				continue
			}

			// Copy genesis.json
			destGenesis := filepath.Join(destDir, "genesis.json")
			if err := copyFile(genesisFile, destGenesis); err != nil {
				ux.Logger.PrintToUser("Warning: failed to copy genesis for %s: %v", subnetName, err)
				continue
			}

			// Copy chain.json as config.json if it exists
			if _, err := os.Stat(chainConfigFile); err == nil {
				destConfig := filepath.Join(destDir, "config.json")
				if err := copyFile(chainConfigFile, destConfig); err != nil {
					ux.Logger.PrintToUser("Warning: failed to copy chain config for %s: %v", subnetName, err)
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
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// PrepareCanonicalChainConfigs creates a canonical chain-configs directory at ~/.lux/chain-configs/
// with subdirectories for each locally deployed subnet's blockchain ID.
// This directory can be passed to nodes via --chain-config-dir flag so all nodes share
// the same chain configs from a single source.
// Returns the canonical chain configs directory path.
func PrepareCanonicalChainConfigs(app *application.Lux) (string, error) {
	// Create canonical chain-configs directory
	chainConfigsDir := filepath.Join(app.GetBaseDir(), "chain-configs")
	if err := os.MkdirAll(chainConfigsDir, 0755); err != nil {
		return "", err
	}

	allSubnetDirs, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		if os.IsNotExist(err) {
			return chainConfigsDir, nil
		}
		return "", err
	}

	preparedCount := 0
	for _, subnetDir := range allSubnetDirs {
		if !subnetDir.IsDir() {
			continue
		}

		subnetName := subnetDir.Name()
		sc, err := app.LoadSidecar(subnetName)
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

		// Source files from canonical subnet location
		subnetConfigDir := filepath.Join(app.GetSubnetDir(), subnetName)
		genesisFile := filepath.Join(subnetConfigDir, constants.GenesisFileName)
		chainConfigFile := filepath.Join(subnetConfigDir, constants.ChainConfigFileName)

		// Check if genesis exists (required for subnet-evm)
		if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
			continue
		}

		// Create blockchain ID subdirectory
		blockchainDir := filepath.Join(chainConfigsDir, blockchainID)
		if err := os.MkdirAll(blockchainDir, 0755); err != nil {
			ux.Logger.PrintToUser("Warning: failed to create chain config dir for %s: %v", subnetName, err)
			continue
		}

		// Copy genesis.json
		destGenesis := filepath.Join(blockchainDir, "genesis.json")
		if err := copyFile(genesisFile, destGenesis); err != nil {
			ux.Logger.PrintToUser("Warning: failed to copy genesis for %s: %v", subnetName, err)
			continue
		}

		// Copy chain.json as config.json if it exists
		if _, err := os.Stat(chainConfigFile); err == nil {
			destConfig := filepath.Join(blockchainDir, "config.json")
			if err := copyFile(chainConfigFile, destConfig); err != nil {
				ux.Logger.PrintToUser("Warning: failed to copy chain config for %s: %v", subnetName, err)
			}
		}
		preparedCount++
	}

	if preparedCount > 0 {
		ux.Logger.PrintToUser("Prepared chain configs for %d net(s) in %s", preparedCount, chainConfigsDir)
	}

	return chainConfigsDir, nil
}
