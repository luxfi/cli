// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package node

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/dependencies"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/config"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/models"
)

func setupLuxd(
	app *application.Lux,
	luxdBinaryPath string,
	luxdVersionSetting dependencies.LuxdVersionSettings,
	network models.Network,
	printFunc func(msg string, args ...interface{}),
) (string, error) {
	var err error
	luxdVersion := ""
	if luxdBinaryPath == "" {
		luxdVersion, err = dependencies.GetLuxdVersion(app, luxdVersionSetting, network)
		if err != nil {
			return "", err
		}
		printFunc("Using Luxd version: %s", luxdVersion)
	}
	luxdBinaryPath, err = localnet.SetupLuxdBinary(app, luxdVersion, luxdBinaryPath)
	if err != nil {
		return "", err
	}
	printFunc("Luxd path: %s\n", luxdBinaryPath)
	return luxdBinaryPath, err
}

func StartLocalNode(
	app *application.Lux,
	clusterName string,
	luxdBinaryPath string,
	numNodes uint32,
	defaultFlags map[string]interface{},
	connectionSettings localnet.ConnectionSettings,
	nodeSettings []localnet.NodeSetting,
	luxdVersionSetting dependencies.LuxdVersionSettings,
	network models.Network,
) error {
	// initializes directories
	networkDir := localnet.GetLocalClusterDir(app, clusterName)
	pluginDir := filepath.Join(networkDir, "plugins")
	if err := os.MkdirAll(networkDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("could not create network directory %s: %w", networkDir, err)
	}
	if err := os.MkdirAll(pluginDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("could not create plugin directory %s: %w", pluginDir, err)
	}

	if localnet.LocalClusterExists(app, clusterName) {
		ux.Logger.GreenCheckmarkToUser("Local cluster %s found. Booting up...", clusterName)
		if err := localnet.LoadLocalCluster(app, clusterName, luxdBinaryPath); err != nil {
			return err
		}
	} else {
		var err error
		luxdBinaryPath, err = setupLuxd(
			app,
			luxdBinaryPath,
			luxdVersionSetting,
			network,
			ux.Logger.PrintToUser,
		)
		if err != nil {
			return err
		}

		ux.Logger.GreenCheckmarkToUser("Local cluster %s not found. Creating...", clusterName)
		// network.ClusterName is not settable as it's a method

		switch {
		case network.Kind() == models.Testnet:
			ux.Logger.PrintToUser(luxlog.Yellow.Wrap("Warning: Testnet Bootstrapping can take several minutes"))
			connectionSettings.NetworkID = network.ID()
		case network.Kind() == models.Mainnet:
			ux.Logger.PrintToUser(luxlog.Yellow.Wrap("Warning: Mainnet Bootstrapping can take 6-24 hours"))
			connectionSettings.NetworkID = network.ID()
		case network.Kind() == models.Local:
			connectionSettings, err = localnet.GetLocalNetworkConnectionInfo(app)
			if err != nil {
				return err
			}
		}

		if defaultFlags == nil {
			defaultFlags = map[string]interface{}{}
		}
		defaultFlags[config.NetworkAllowPrivateIPsKey] = true
		defaultFlags[config.IndexEnabledKey] = false
		defaultFlags[config.IndexAllowIncompleteKey] = true

		ux.Logger.PrintToUser("Starting local luxd node using root: %s ...", networkDir)
		spinSession := ux.NewUserSpinner()
		spinner := spinSession.SpinToUser("Booting Network. Wait until healthy...")

		_, err = localnet.CreateLocalCluster(
			app,
			ux.Logger.PrintToUser,
			clusterName,
			luxdBinaryPath,
			pluginDir,
			defaultFlags,
			connectionSettings,
			numNodes,
			nodeSettings,
			[]ids.ID{},
			network,
			true, // Download DB
			true, // Bootstrap
		)
		if err != nil {
			ux.SpinFailWithError(spinner, "", err)
			return fmt.Errorf("failed to start local luxd: %w", err)
		}

		ux.SpinComplete(spinner)
		spinSession.Stop()
	}

	ux.Logger.GreenCheckmarkToUser("Luxgo started and ready to use from %s", networkDir)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Node logs directory: %s/<NodeID>/logs", networkDir)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Network ready to use.")
	ux.Logger.PrintToUser("")

	cluster, err := localnet.GetLocalCluster(app, clusterName)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		ux.Logger.PrintToUser("URI: %s", node.URI)
		ux.Logger.PrintToUser("NodeID: %s", node.NodeID)
		ux.Logger.PrintToUser("")
	}

	return nil
}

func LocalStatus(
	app *application.Lux,
	clusterName string,
	blockchainName string,
) error {
	var localClusters []string
	if clusterName != "" {
		if !localnet.LocalClusterExists(app, clusterName) {
			return fmt.Errorf("local node %q is not found", clusterName)
		}
		localClusters = []string{clusterName}
	} else {
		var err error
		localClusters, err = localnet.GetLocalClusters(app)
		if err != nil {
			return fmt.Errorf("failed to list local clusters: %w", err)
		}
	}
	if clusterName != "" {
		ux.Logger.PrintToUser("%s %s", luxlog.Blue.Wrap("Local cluster:"), luxlog.Green.Wrap(clusterName))
	} else if len(localClusters) > 0 {
		ux.Logger.PrintToUser(luxlog.Blue.Wrap("Local clusters:"))
	}
	for _, clusterName := range localClusters {
		currenlyRunning := ""
		healthStatus := ""
		luxdURIOuput := ""

		network, err := localnet.GetLocalClusterNetworkModel(app, clusterName)
		if err != nil {
			return fmt.Errorf("failed to get cluster network: %w", err)
		}
		networkKind := fmt.Sprintf(" [%s]", luxlog.Orange.Wrap(network.Name()))

		// load sidecar and cluster config for the cluster  if blockchainName is not empty
		blockchainID := ids.Empty
		if blockchainName != "" {
			sc, err := app.LoadSidecar(blockchainName)
			if err != nil {
				return err
			}
			blockchainID = sc.Networks[network.Name()].BlockchainID
		}
		isRunning, err := localnet.LocalClusterIsRunning(app, clusterName)
		if err != nil {
			return err
		}
		if isRunning {
			pChainHealth, l1Health, err := localnet.LocalClusterHealth(app, clusterName)
			if err != nil {
				return err
			}
			currenlyRunning = fmt.Sprintf(" [%s]", luxlog.Blue.Wrap("Running"))
			if pChainHealth && l1Health {
				healthStatus = fmt.Sprintf(" [%s]", luxlog.Green.Wrap("Healthy"))
			} else {
				healthStatus = fmt.Sprintf(" [%s]", luxlog.Red.Wrap("Unhealthy"))
			}
			runningLuxdURIs, err := localnet.GetLocalClusterURIs(app, clusterName)
			if err != nil {
				return err
			}
			for _, luxdURI := range runningLuxdURIs {
				nodeID, nodePOP, isBoot, err := getInfo(luxdURI, blockchainID.String())
				if err != nil {
					ux.Logger.RedXToUser("failed to get node  %s info: %v", luxdURI, err)
					continue
				}
				nodePOPPubKey := "0x" + hex.EncodeToString(nodePOP.PublicKey[:])
				nodePOPProof := "0x" + hex.EncodeToString(nodePOP.ProofOfPossession[:])

				isBootStr := "Primary:" + luxlog.Red.Wrap("Not Bootstrapped")
				if isBoot {
					isBootStr = "Primary:" + luxlog.Green.Wrap("Bootstrapped")
				}

				blockchainStatus := ""
				if blockchainID != ids.Empty {
					blockchainStatus, _ = getBlockchainStatus(luxdURI, blockchainID.String()) // silence errors
				}

				luxdURIOuput += fmt.Sprintf("   - %s [%s] [%s]\n     publicKey: %s \n     proofOfPossession: %s \n",
					luxlog.LightBlue.Wrap(luxdURI),
					nodeID,
					strings.TrimRight(strings.Join([]string{isBootStr, "L1:" + luxlog.Orange.Wrap(blockchainStatus)}, " "), " "),
					nodePOPPubKey,
					nodePOPProof,
				)
			}
		} else {
			currenlyRunning = fmt.Sprintf(" [%s]", luxlog.LightGray.Wrap("Stopped"))
		}
		networkDir := localnet.GetLocalClusterDir(app, clusterName)
		ux.Logger.PrintToUser("- %s: %s %s %s %s", clusterName, networkDir, networkKind, currenlyRunning, healthStatus)
		ux.Logger.PrintToUser(luxdURIOuput)
	}

	return nil
}

func getInfo(uri string, blockchainID string) (
	ids.NodeID, // nodeID
	*signer.ProofOfPossession, // nodePOP
	bool, // isBootstrapped
	error, // error
) {
	client := info.NewClient(uri)
	ctx, cancel := utils.GetAPILargeContext()
	defer cancel()
	nodeID, nodePOP, err := client.GetNodeID(ctx)
	if err != nil {
		return ids.EmptyNodeID, &signer.ProofOfPossession{}, false, err
	}
	isBootstrapped, err := client.IsBootstrapped(ctx, blockchainID)
	if err != nil {
		return nodeID, nodePOP, isBootstrapped, err
	}
	return nodeID, nodePOP, isBootstrapped, err
}

func getBlockchainStatus(uri string, blockchainID string) (
	string, // status
	error, // error
) {
	client := platformvm.NewClient(uri)
	ctx, cancel := utils.GetAPILargeContext()
	defer cancel()
	status, err := client.GetBlockchainStatus(ctx, blockchainID)
	if err != nil {
		return "", err
	}
	if status.String() == "" {
		return "Not Syncing", nil
	}
	return status.String(), nil
}
