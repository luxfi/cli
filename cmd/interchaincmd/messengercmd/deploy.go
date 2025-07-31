// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package messengercmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/contract"
	"github.com/luxfi/cli/pkg/interchain"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/utils/logging"

	"github.com/spf13/cobra"
)

type DeployFlags struct {
	Network                      networkoptions.NetworkFlags
	ChainFlags                   contract.ChainSpec
	KeyName                      string
	GenesisKey                   bool
	DeployMessenger              bool
	DeployRegistry               bool
	ForceRegistryDeploy          bool
	RPCURL                       string
	Version                      string
	MessengerContractAddressPath string
	MessengerDeployerAddressPath string
	MessengerDeployerTxPath      string
	RegistryBydecodePath         string
	PrivateKeyFlags              contract.PrivateKeyFlags
	IncludeCChain                bool
	CChainKeyName                string
}

const (
	cChainAlias = "C"
	cChainName  = "c-chain"
)

var deployFlags DeployFlags

// lux interchain messenger deploy
func NewDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys Warp Messenger and Registry into a given L1",
		Long: `Deploys Warp Messenger and Registry into a given L1.

For Local Networks, it also deploys into C-Chain.`,
		RunE: deploy,
		Args: cobrautils.ExactArgs(0),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &deployFlags.Network, true, networkoptions.DefaultSupportedNetworkOptions)
	deployFlags.PrivateKeyFlags.AddToCmd(cmd, "to fund Warp deploy")
	deployFlags.ChainFlags.SetEnabled(true, true, false, false, true)
	deployFlags.ChainFlags.AddToCmd(cmd, "deploy Warp into %s")
	cmd.Flags().BoolVar(&deployFlags.DeployMessenger, "deploy-messenger", true, "deploy Warp Messenger")
	cmd.Flags().BoolVar(&deployFlags.DeployRegistry, "deploy-registry", true, "deploy Warp Registry")
	cmd.Flags().BoolVar(&deployFlags.ForceRegistryDeploy, "force-registry-deploy", false, "deploy Warp Registry even if Messenger has already been deployed")
	cmd.Flags().StringVar(&deployFlags.RPCURL, "rpc-url", "", "use the given RPC URL to connect to the subnet")
	cmd.Flags().StringVar(&deployFlags.Version, "version", "latest", "version to deploy")
	cmd.Flags().StringVar(&deployFlags.MessengerContractAddressPath, "messenger-contract-address-path", "", "path to a messenger contract address file")
	cmd.Flags().StringVar(&deployFlags.MessengerDeployerAddressPath, "messenger-deployer-address-path", "", "path to a messenger deployer address file")
	cmd.Flags().StringVar(&deployFlags.MessengerDeployerTxPath, "messenger-deployer-tx-path", "", "path to a messenger deployer tx file")
	cmd.Flags().StringVar(&deployFlags.RegistryBydecodePath, "registry-bytecode-path", "", "path to a registry bytecode file")
	cmd.Flags().BoolVar(&deployFlags.IncludeCChain, "include-cchain", false, "deploy Warp also to C-Chain")
	cmd.Flags().StringVar(&deployFlags.CChainKeyName, "cchain-key", "", "key to be used to pay fees to deploy Warp to C-Chain")
	return cmd
}

func deploy(_ *cobra.Command, args []string) error {
	return CallDeploy(args, deployFlags, models.UndefinedNetwork)
}

func CallDeploy(_ []string, flags DeployFlags, network models.Network) error {
	var err error
	if network == models.UndefinedNetwork {
		network, err = networkoptions.GetNetworkFromCmdLineFlags(
			app,
			"On what Network do you want to deploy the Warp Messenger?",
			flags.Network,
			true,
			false,
			networkoptions.DefaultSupportedNetworkOptions,
			"",
		)
		if err != nil {
			return err
		}
	}
	if err := flags.ChainFlags.CheckMutuallyExclusiveFields(); err != nil {
		return err
	}
	if !flags.DeployMessenger && !flags.DeployRegistry {
		return fmt.Errorf("you should set at least one of --deploy-messenger/--deploy-registry to true")
	}
	if !flags.ChainFlags.Defined() {
		prompt := "Which Blockchain would you like to deploy Warp to?"
		if cancel, err := contract.PromptChain(
			app,
			network,
			prompt,
			"",
			&flags.ChainFlags,
		); err != nil {
			return err
		} else if cancel {
			return nil
		}
	}
	rpcURL := flags.RPCURL
	if rpcURL == "" {
		rpcURL, _, err = contract.GetBlockchainEndpoints(app, network, flags.ChainFlags, true, false)
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser(logging.Yellow.Wrap("RPC Endpoint: %s"), rpcURL)
	}

	genesisAddress, genesisPrivateKey, err := contract.GetEVMSubnetPrefundedKey(
		app,
		network,
		flags.ChainFlags,
	)
	if err != nil {
		return err
	}
	privateKey, err := flags.PrivateKeyFlags.GetPrivateKey(app, genesisPrivateKey)
	if err != nil {
		return err
	}
	if privateKey == "" {
		privateKey, err = prompts.PromptPrivateKey(
			app.Prompt,
			"deploy Warp",
			app.GetKeyDir(),
			app.GetKey,
			genesisAddress,
			genesisPrivateKey,
		)
		if err != nil {
			return err
		}
	}
	var warpVersion string
	switch {
	case flags.MessengerContractAddressPath != "" || flags.MessengerDeployerAddressPath != "" || flags.MessengerDeployerTxPath != "" || flags.RegistryBydecodePath != "":
		if flags.MessengerContractAddressPath == "" || flags.MessengerDeployerAddressPath == "" || flags.MessengerDeployerTxPath == "" || flags.RegistryBydecodePath == "" {
			return fmt.Errorf("if setting any Warp asset path, you must set all Warp asset paths")
		}
	case flags.Version != "" && flags.Version != "latest":
		warpVersion = flags.Version
	default:
		warpInfo, err := interchain.GetWarpInfo(app)
		if err != nil {
			return err
		}
		warpVersion = warpInfo.Version
	}
	// deploy to subnet
	td := interchain.WarpDeployer{}
	if flags.MessengerContractAddressPath != "" {
		if err := td.SetAssetsFromPaths(
			flags.MessengerContractAddressPath,
			flags.MessengerDeployerAddressPath,
			flags.MessengerDeployerTxPath,
			flags.RegistryBydecodePath,
		); err != nil {
			return err
		}
	} else {
		if err := td.DownloadAssets(
			app.GetWarpContractsBinDir(),
			warpVersion,
		); err != nil {
			return err
		}
	}
	blockchainDesc, err := contract.GetBlockchainDesc(flags.ChainFlags)
	if err != nil {
		return err
	}
	alreadyDeployed, messengerAddress, registryAddress, err := td.Deploy(
		blockchainDesc,
		rpcURL,
		privateKey,
		flags.DeployMessenger,
		flags.DeployRegistry,
		flags.ForceRegistryDeploy,
	)
	if err != nil {
		return err
	}
	if flags.ChainFlags.BlockchainName != "" && (!alreadyDeployed || flags.ForceRegistryDeploy) {
		// update sidecar
		sc, err := app.LoadSidecar(flags.ChainFlags.BlockchainName)
		if err != nil {
			return fmt.Errorf("failed to load sidecar: %w", err)
		}
		sc.TeleporterReady = true
		sc.TeleporterVersion = warpVersion
		networkInfo := sc.Networks[network.Name()]
		if messengerAddress != "" {
			networkInfo.TeleporterMessengerAddress = messengerAddress
		}
		if registryAddress != "" {
			networkInfo.TeleporterRegistryAddress = registryAddress
		}
		sc.Networks[network.Name()] = networkInfo
		if err := app.UpdateSidecar(&sc); err != nil {
			return err
		}
	}
	// automatic deploy to cchain for local
	if !flags.ChainFlags.CChain && (network.Kind == models.Local || flags.IncludeCChain) {
		if flags.CChainKeyName == "" {
			flags.CChainKeyName = "ewoq"
		}
		ewoq, err := app.GetKey(flags.CChainKeyName, network, false)
		if err != nil {
			return err
		}
		alreadyDeployed, messengerAddress, registryAddress, err := td.Deploy(
			cChainName,
			network.BlockchainEndpoint(cChainAlias),
			ewoq.PrivKeyHex(),
			flags.DeployMessenger,
			flags.DeployRegistry,
			false,
		)
		if err != nil {
			return err
		}
		if !alreadyDeployed {
			if network.Kind == models.Local {
				if err := localnet.WriteExtraLocalNetworkData(
					app,
					"",
					"",
					messengerAddress,
					registryAddress,
				); err != nil {
					return err
				}
			}
			if network.ClusterName != "" {
				clusterConfig, err := app.GetClusterConfig(network.ClusterName)
				if err != nil {
					return err
				}
				if messengerAddress != "" {
					clusterConfig.ExtraNetworkData.CChainTeleporterMessengerAddress = messengerAddress
				}
				if registryAddress != "" {
					clusterConfig.ExtraNetworkData.CChainTeleporterRegistryAddress = registryAddress
				}
				if err := app.SetClusterConfig(network.ClusterName, clusterConfig); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
