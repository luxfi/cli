// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"encoding/json"
	"fmt"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/localnetworkinterface"
	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	deployLocal   bool
	deployTestnet bool
	deployMainnet bool
	nodeVersion   string
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [chainName]",
		Short: "Deploy a blockchain to local network, testnet, or mainnet",
		Long: `Deploy a configured blockchain to the network.

Networks:
  --local     Deploy to local 5-node network (default)
  --testnet   Deploy to Lux testnet
  --mainnet   Deploy to Lux mainnet

The local network must be running before deployment.
Start it with: lux network start --mainnet

Examples:
  # Deploy to local network
  lux chain deploy mychain --local

  # Deploy to testnet
  lux chain deploy mychain --testnet`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         deployChain,
	}

	cmd.Flags().BoolVarP(&deployLocal, "local", "l", false, "Deploy to local network")
	cmd.Flags().BoolVarP(&deployTestnet, "testnet", "t", false, "Deploy to testnet")
	cmd.Flags().BoolVarP(&deployMainnet, "mainnet", "m", false, "Deploy to mainnet")
	cmd.Flags().StringVar(&nodeVersion, "node-version", "latest", "Node version to use")

	return cmd
}

func deployChain(cmd *cobra.Command, args []string) error {
	chainName := args[0]

	// Load sidecar
	sc, err := app.LoadSidecar(chainName)
	if err != nil {
		return fmt.Errorf("chain %s not found. Create it first with: lux chain create %s", chainName, chainName)
	}

	// Load genesis
	chainGenesis, err := app.LoadRawGenesis(chainName)
	if err != nil {
		return fmt.Errorf("failed to load genesis: %w", err)
	}

	// Validate genesis
	if sc.VM == models.EVM {
		var genesis core.Genesis
		if err := json.Unmarshal(chainGenesis, &genesis); err != nil {
			return fmt.Errorf("invalid genesis format: %w", err)
		}
	}

	// Determine network
	var network models.Network
	switch {
	case deployMainnet:
		network = models.Mainnet
	case deployTestnet:
		network = models.Testnet
	case deployLocal:
		network = models.Local
	default:
		network = models.Local // Default to local
	}

	ux.Logger.PrintToUser("Deploying %s to %s", chainName, network.String())

	// All deployments use the same flow - deploy to locally running network
	return deployToNetwork(chainName, chainGenesis, &sc, network)
}

func deployToNetwork(chainName string, chainGenesis []byte, sc *models.Sidecar, network models.Network) error {
	app.Log.Debug("Deploy to network", zap.String("network", network.String()))

	// Get VM binary
	var vmBin string
	var err error

	switch sc.VM {
	case models.EVM:
		vmBin, err = binutils.SetupEVM(app, sc.VMVersion)
		if err != nil {
			return fmt.Errorf("failed to setup EVM: %w", err)
		}
	case models.CustomVM:
		vmBin = binutils.SetupCustomBin(app, chainName)
	default:
		return fmt.Errorf("unknown VM type: %s", sc.VM)
	}

	// Check RPC version compatibility
	if sc.VM != models.CustomVM {
		nc := localnetworkinterface.NewStatusChecker()
		nodeVersion, err = checkDeployCompatibility(nc, sc.RPCVersion)
		if err != nil {
			return err
		}
	}

	// Create deployer
	deployer := chain.NewLocalDeployer(app, nodeVersion, vmBin)

	// Get genesis path
	genesisPath := app.GetGenesisPath(chainName)

	// Deploy to locally-running network (works for local, testnet, mainnet started via CLI)
	subnetID, blockchainID, err := deployer.DeployToLocalNetwork(chainName, chainGenesis, genesisPath)
	if err != nil {
		if deployer.BackendStartedHere() {
			if innerErr := binutils.KillgRPCServerProcess(app); innerErr != nil {
				app.Log.Warn("failed to kill gRPC server", zap.Error(innerErr))
			}
		}
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Update sidecar with deployment info (using the target network)
	return app.UpdateSidecarNetworks(sc, network, subnetID, blockchainID)
}

func checkDeployCompatibility(network localnetworkinterface.StatusChecker, configuredRPCVersion int) (string, error) {
	runningVersion, runningRPCVersion, networkRunning, err := network.GetCurrentNetworkVersion()
	if err != nil {
		return "", err
	}

	if networkRunning {
		if nodeVersion == "latest" {
			if runningRPCVersion != configuredRPCVersion {
				return "", fmt.Errorf(
					"running node uses RPC version %d but chain requires %d",
					runningRPCVersion,
					configuredRPCVersion,
				)
			}
			return runningVersion, nil
		}
		if runningVersion != nodeVersion {
			return "", fmt.Errorf("incompatible node version: running %s, requested %s", runningVersion, nodeVersion)
		}
	}

	return nodeVersion, nil
}
