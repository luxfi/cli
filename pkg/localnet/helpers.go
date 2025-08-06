// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/wallet/subnet/primary"
)

// Update network given by [networkDir], with all blockchain config of [blockchainName]
func UpdateBlockchainConfig(
	app *application.Lux,
	networkDir string,
	blockchainName string,
) error {
	networkModel, err := GetNetworkModel(networkDir)
	if err != nil {
		return err
	}
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}
	if sc.Networks[networkModel.Name()].BlockchainID == ids.Empty {
		return fmt.Errorf("blockchain %s has not been deployed to %s", blockchainName, networkModel.Name())
	}
	blockchainID := sc.Networks[networkModel.Name()].BlockchainID
	subnetID := sc.Networks[networkModel.Name()].SubnetID
	var (
		blockchainConfig   []byte
		blockchainUpgrades []byte
		subnetConfig       []byte
		nodeConfig         map[string]interface{}
	)
	vmID, err := utils.VMID(blockchainName)
	if err != nil {
		return err
	}
	vmBinaryPath, err := SetupVMBinary(app, blockchainName)
	if err != nil {
		return fmt.Errorf("failed to setup VM binary: %w", err)
	}
	if app.ChainConfigExists(blockchainName) {
		blockchainConfig, err = os.ReadFile(app.GetChainConfigPath(blockchainName))
		if err != nil {
			return err
		}
	}
	if app.NetworkUpgradeExists(blockchainName) {
		blockchainUpgrades, err = os.ReadFile(app.GetUpgradeBytesFilepath(blockchainName))
		if err != nil {
			return err
		}
	}
	if app.LuxdSubnetConfigExists(blockchainName) {
		subnetConfig, err = os.ReadFile(app.GetLuxdSubnetConfigPath(blockchainName))
		if err != nil {
			return err
		}
	}
	// Convert per-node config from map[string]interface{} to map[ids.NodeID][]byte
	rawPerNodeConfig := app.GetPerNodeBlockchainConfig(blockchainName)
	perNodeBlockchainConfig := make(map[ids.NodeID][]byte)
	for nodeIDStr, config := range rawPerNodeConfig {
		nodeID, err := ids.NodeIDFromString(nodeIDStr)
		if err != nil {
			return fmt.Errorf("invalid node ID %s: %w", nodeIDStr, err)
		}
		configBytes, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config for node %s: %w", nodeIDStr, err)
		}
		perNodeBlockchainConfig[nodeID] = configBytes
	}
	
	// general node config
	nodeConfigStr, err := app.Conf.LoadNodeConfig()
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(nodeConfigStr), &nodeConfig); err != nil {
		return fmt.Errorf("invalid common node config JSON: %w", err)
	}
	// blockchain node config
	if app.LuxdNodeConfigExists(blockchainName) {
		var blockchainNodeConfig map[string]interface{}
		if err := utils.ReadJSON(app.GetLuxdNodeConfigPath(blockchainName), &blockchainNodeConfig); err != nil {
			return err
		}
		for k, v := range blockchainNodeConfig {
			nodeConfig[k] = v
		}
	}
	return TmpNetUpdateBlockchainConfig(
		NewLoggerAdapter(app.Log),
		networkDir,
		subnetID,
		blockchainID,
		vmID,
		vmBinaryPath,
		blockchainConfig,
		perNodeBlockchainConfig,
		blockchainUpgrades,
		subnetConfig,
		nodeConfig,
	)
}

// Tracks the given [blockchainName] at network given on [networkDir]
// After P-Chain is bootstrapped, set alias [blockchainName]->blockchainID
// for the network, and persists RPC into sidecar
// Use both for local networks and local clusters
func TrackSubnet(
	app *application.Lux,
	printFunc func(msg string, args ...interface{}),
	blockchainName string,
	networkDir string,
	wallet primary.Wallet,
) error {
	if err := UpdateBlockchainConfig(
		app,
		networkDir,
		blockchainName,
	); err != nil {
		return err
	}
	networkModel, err := GetNetworkModel(networkDir)
	if err != nil {
		return err
	}
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}
	if sc.Networks[networkModel.Name()].BlockchainID == ids.Empty {
		return fmt.Errorf("blockchain %s has not been deployed to %s", blockchainName, networkModel.Name())
	}
	blockchainID := sc.Networks[networkModel.Name()].BlockchainID
	subnetID := sc.Networks[networkModel.Name()].SubnetID
	ctx, cancel := networkModel.BootstrappingContext()
	defer cancel()
	if err := TmpNetTrackSubnet(
		ctx,
		NewLoggerAdapter(app.Log),
		printFunc,
		networkDir,
		sc.Sovereign,
		blockchainID,
		subnetID,
		wallet,
	); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			printFunc("")
			printFunc("A context timeout has occurred while trying to bootstrap the blockchain.")
			printFunc("")
			logPaths, _ := GetTmpNetAvailableLogs(networkDir, blockchainID, false)
			if len(logPaths) != 0 {
				printFunc("Please check this log files for more information on the error cause:")
				for _, logPath := range logPaths {
					printFunc("  " + logPath)
				}
				printFunc("")
			}
		}
		return err
	}
	ux.Logger.GreenCheckmarkToUser("%s successfully tracking %s", networkModel.Name(), blockchainName)
	if networkModel.Kind() == models.Local {
		if err := TmpNetSetDefaultAliases(ctx, networkDir); err != nil {
			return err
		}
	}
	nodeURIs, err := GetTmpNetNodeURIsWithFix(networkDir)
	if err != nil {
		return err
	}
	_, err = app.AddDefaultBlockchainRPCsToSidecar(
		blockchainName,
		networkModel,
		nodeURIs,
	)
	return err
}

// Returns the network model for the network at [networkDir]
func GetNetworkModel(
	networkDir string,
) (models.Network, error) {
	network, err := GetTmpNetNetwork(networkDir)
	if err != nil {
		return models.Undefined, err
	}
	networkID, err := GetTmpNetNetworkID(network)
	if err != nil {
		return models.Undefined, err
	}
	return models.NetworkFromNetworkID(networkID), nil
}
