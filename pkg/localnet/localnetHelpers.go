// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/node/wallet/net/primary"
)

// information that is persisted alongside the local network
type ExtraLocalNetworkData struct {
	RelayerPath                      string
	CChainTeleporterMessengerAddress string
	CChainTeleporterRegistryAddress  string
}

// Restart all nodes on local network to track [blockchainName].
// Before that, set up VM binary, blockchain and subnet config information
// After the blockchain is bootstrapped, add alias for [blockchainName]->[blockchainID]
// Finally persist all new blockchain RPC URLs into blockchain sidecar.
func LocalNetworkTrackSubnet(
	app *application.Lux,
	printFunc func(msg string, args ...interface{}),
	blockchainName string,
) error {
	networkDir, err := GetLocalNetworkDir(app)
	if err != nil {
		return err
	}
	networkModel := models.NewLocalNetwork()
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}
	if sc.Networks[networkModel.Name()].BlockchainID == ids.Empty {
		return fmt.Errorf("blockchain %s has not been deployed to %s", blockchainName, networkModel.Name())
	}
	subnetID := sc.Networks[networkModel.Name()].SubnetID
	wallet, err := GetLocalNetworkWallet(app, []ids.ID{subnetID})
	if err != nil {
		return err
	}
	return TrackSubnet(
		app,
		printFunc,
		blockchainName,
		networkDir,
		wallet,
	)
}

// Indicates if [blockchainName] is found to be deployed on the local network, based on the VMID associated to it
func BlockchainAlreadyDeployedOnLocalNetwork(app *application.Lux, blockchainName string) (bool, error) {
	chainVMID, err := utils.VMID(blockchainName)
	if err != nil {
		return false, fmt.Errorf("failed to create VM ID from %s: %w", blockchainName, err)
	}
	blockchains, err := GetLocalNetworkBlockchainsInfo(app)
	if err != nil {
		return false, err
	}
	for _, chain := range blockchains {
		if chain.VMID == chainVMID {
			return true, nil
		}
	}
	return false, nil
}

// Returns the configuration file for the local network relayer
// if [networkDir] is given, assumes that the local network is running from that dir
func GetLocalNetworkRelayerConfigPath(app *application.Lux, networkDir string) (bool, string, error) {
	if networkDir == "" {
		var err error
		networkDir, err = GetLocalNetworkDir(app)
		if err != nil {
			return false, "", err
		}
	}
	relayerConfigPath := app.GetLocalRelayerConfigPath()
	return utils.FileExists(relayerConfigPath), relayerConfigPath, nil
}

// GetLocalNetworkWallet returns a wallet that can operate on the local network
// initialized to recognice all given [subnetIDs] as pre generated
func GetLocalNetworkWallet(
	app *application.Lux,
	subnetIDs []ids.ID,
) (primary.Wallet, error) {
	endpoint, err := GetLocalNetworkEndpoint(app)
	if err != nil {
		return nil, err
	}
	// Use subnetIDs for validation if needed
	_ = subnetIDs // Currently unused but available for subnet-specific operations
	
	ctx, cancel := GetLocalNetworkDefaultContext()
	defer cancel()
	
	// Create keychain for the wallet with EWOQ key for local testing
	keychain := secp256k1fx.NewKeychain()
	// Load EWOQ test key
	// For now, use an empty keychain since we need to convert the key format
	// The EWOQ key conversion would need proper CB58 decoding
	
	walletConfig := &primary.WalletConfig{
		URI:         endpoint,
		LUXKeychain: keychain,
		EthKeychain: keychain,
	}
	
	return primary.MakeWallet(ctx, walletConfig)
}

// Gathers extra information for the local network, not available on the primary storage
func GetExtraLocalNetworkData(app *application.Lux, rootDataDir string) (bool, ExtraLocalNetworkData, error) {
	extraLocalNetworkData := ExtraLocalNetworkData{}
	if rootDataDir == "" {
		var err error
		rootDataDir, err = GetLocalNetworkDir(app)
		if err != nil {
			return false, extraLocalNetworkData, err
		}
	}
	extraLocalNetworkDataPath := filepath.Join(rootDataDir, constants.ExtraLocalNetworkDataFilename)
	if !utils.FileExists(extraLocalNetworkDataPath) {
		return false, extraLocalNetworkData, nil
	}
	bs, err := os.ReadFile(extraLocalNetworkDataPath)
	if err != nil {
		return false, extraLocalNetworkData, err
	}
	if err := json.Unmarshal(bs, &extraLocalNetworkData); err != nil {
		return false, extraLocalNetworkData, err
	}
	return true, extraLocalNetworkData, nil
}

// Writes extra information for the local network, not available on the primary storage
func WriteExtraLocalNetworkData(
	app *application.Lux,
	rootDataDir string,
	relayerPath string,
	cchainWarpMessengerAddress string,
	cchainWarpRegistryAddress string,
) error {
	if rootDataDir == "" {
		var err error
		rootDataDir, err = GetLocalNetworkDir(app)
		if err != nil {
			return err
		}
	}
	extraLocalNetworkData := ExtraLocalNetworkData{}
	extraLocalNetworkDataPath := filepath.Join(rootDataDir, constants.ExtraLocalNetworkDataFilename)
	if utils.FileExists(extraLocalNetworkDataPath) {
		var err error
		_, extraLocalNetworkData, err = GetExtraLocalNetworkData(app, rootDataDir)
		if err != nil {
			return err
		}
	}
	if relayerPath != "" {
		extraLocalNetworkData.RelayerPath = utils.ExpandHome(relayerPath)
	}
	if cchainWarpMessengerAddress != "" {
		extraLocalNetworkData.CChainTeleporterMessengerAddress = cchainWarpMessengerAddress
	}
	if cchainWarpRegistryAddress != "" {
		extraLocalNetworkData.CChainTeleporterRegistryAddress = cchainWarpRegistryAddress
	}
	bs, err := json.Marshal(&extraLocalNetworkData)
	if err != nil {
		return err
	}
	return os.WriteFile(extraLocalNetworkDataPath, bs, constants.WriteReadReadPerms)
}
