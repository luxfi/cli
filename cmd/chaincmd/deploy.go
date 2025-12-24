// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/localnetworkinterface"
	"github.com/luxfi/cli/pkg/utils"
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
		err = fmt.Errorf("chain %s not found. Create it first with: lux chain create %s", chainName, chainName)
		ux.Logger.PrintError("%s", err)
		return err
	}

	// Load genesis
	chainGenesis, err := app.LoadRawGenesis(chainName)
	if err != nil {
		err = fmt.Errorf("failed to load genesis: %w", err)
		ux.Logger.PrintError("%s", err)
		return err
	}

	// Validate genesis
	if sc.VM == models.EVM {
		var genesis core.Genesis
		if err := json.Unmarshal(chainGenesis, &genesis); err != nil {
			err = fmt.Errorf("invalid genesis format: %w", err)
			ux.Logger.PrintError("%s", err)
			return err
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
	if err := deployToNetwork(chainName, chainGenesis, &sc, network); err != nil {
		ux.Logger.PrintError("%s", err)
		return err
	}
	return nil
}

// verifyVMInstalled checks that the VM plugin is installed before deployment.
// Returns nil if VM is ready, otherwise returns an actionable error.
func verifyVMInstalled(chainName string, sc *models.Sidecar) error {
	// Get the actual VM name based on VM type
	// The VMID is computed from the VM name, not the chain name
	vmName := "Lux EVM" // Default for EVM chains
	if sc.VM == models.CustomVM {
		vmName = chainName // For custom VMs, use chain name
	}

	// Compute VMID from VM name
	vmID, err := utils.VMID(vmName)
	if err != nil {
		return fmt.Errorf("failed to compute VMID for VM %s: %w", vmName, err)
	}
	vmIDStr := vmID.String()

	// Get plugins directory path - plugins/current is the active plugins directory
	pluginPath := filepath.Join(app.GetCurrentPluginsDir(), vmIDStr)

	// Check if plugin exists
	info, err := os.Lstat(pluginPath)
	if os.IsNotExist(err) {
		// Plugin does not exist - provide actionable error
		displayName := getVMDisplayName(sc.VM)
		return fmt.Errorf(`VM '%s' not installed (VMID: %s)

To fix, run:
  lux vm link "%s" --path ~/work/lux/evm/build/evm`,
			displayName, vmIDStr, vmName)
	}
	if err != nil {
		return fmt.Errorf("failed to check VM plugin at %s: %w", pluginPath, err)
	}

	// Check if it's a symlink with a missing target
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to read symlink %s: %w", pluginPath, err)
		}

		// Resolve target path (handle relative symlinks)
		if !filepath.IsAbs(target) {
			target = filepath.Join(app.GetCurrentPluginsDir(), target)
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			vmName := getVMDisplayName(sc.VM)
			return fmt.Errorf(`VM '%s' symlink exists but target is missing (VMID: %s)

Plugin symlink: %s
Target (missing): %s

To fix, update the symlink:
  rm %s
  lux vm link %s --path <path-to-vm-binary>`,
				vmName, vmIDStr,
				pluginPath, target,
				pluginPath, chainName)
		}
	}

	// Verify it's executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("VM plugin at %s is not executable", pluginPath)
	}

	app.Log.Debug("VM plugin verified", zap.String("vmid", vmIDStr), zap.String("path", pluginPath))
	return nil
}

// getVMDisplayName returns a human-readable name for the VM type
func getVMDisplayName(vm models.VMType) string {
	switch vm {
	case models.EVM:
		return "Lux EVM"
	case models.CustomVM:
		return "Custom VM"
	default:
		return string(vm)
	}
}

// getVMVersion returns the VM version or a default
func getVMVersion(sc *models.Sidecar) string {
	if sc.VMVersion != "" {
		return sc.VMVersion
	}
	return "latest"
}

func deployToNetwork(chainName string, chainGenesis []byte, sc *models.Sidecar, network models.Network) error {
	app.Log.Debug("Deploy to network", zap.String("network", network.String()))

	// Map deploy target to network type
	targetType := "local"
	switch network {
	case models.Testnet:
		targetType = "testnet"
	case models.Mainnet:
		targetType = "mainnet"
	case models.Local:
		targetType = "local"
	}

	// Load network state to get gRPC port for correct network
	networkState, stateErr := app.LoadNetworkState()
	if stateErr != nil {
		return fmt.Errorf("failed to load network state: %w\nIs the network running? Start with: lux network start --%s", stateErr, targetType)
	}
	if networkState == nil || !networkState.Running {
		return fmt.Errorf("no network running. Start the network first with: lux network start --%s", targetType)
	}

	// Verify that the running network matches the requested target
	if networkState.NetworkType != targetType {
		return fmt.Errorf(
			"network mismatch: trying to deploy to %s but %s is running. "+
				"Either stop the current network with 'lux network stop' and start the correct one, "+
				"or use the correct --testnet/--mainnet/--local flag",
			targetType, networkState.NetworkType)
	}

	// Log gRPC port being used
	app.Log.Debug("Using gRPC port from network state", zap.Int("port", networkState.GRPCPort), zap.String("network", networkState.NetworkType))

	// Preflight check: verify VM is installed before any network operations
	if err := verifyVMInstalled(chainName, sc); err != nil {
		return err
	}

	// Get VM binary - prefer linked plugin over downloaded
	var vmBin string
	var err error

	// Compute VMID for plugin lookup
	vmName := "Lux EVM"
	if sc.VM == models.CustomVM {
		vmName = chainName
	}
	vmID, _ := utils.VMID(vmName)
	vmIDStr := vmID.String()

	switch sc.VM {
	case models.EVM:
		// First check if EVM plugin already exists (linked or copied)
		pluginPath := filepath.Join(app.GetCurrentPluginsDir(), vmIDStr)
		if info, pluginErr := os.Stat(pluginPath); pluginErr == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			// Plugin exists and is executable, use it directly
			vmBin = pluginPath
			app.Log.Debug("Using existing EVM plugin", zap.String("path", vmBin))
		} else {
			// Fall back to downloading
			vmBin, err = binutils.SetupEVM(app, sc.VMVersion)
			if err != nil {
				return fmt.Errorf("failed to setup EVM: %w", err)
			}
		}
	case models.CustomVM:
		vmBin = binutils.SetupCustomBin(app, chainName)
	default:
		return fmt.Errorf("unknown VM type: %s", sc.VM)
	}

	// Check RPC version compatibility
	if sc.VM != models.CustomVM {
		// Use app-aware status checker to detect the correct running network endpoint
		nc := localnetworkinterface.NewStatusCheckerWithApp(app)
		nodeVersion, err = checkDeployCompatibility(nc, sc.RPCVersion)
		if err != nil {
			return fmt.Errorf("RPC version check failed: %w", err)
		}
	}

	// Create deployer with network-aware gRPC client
	// This ensures we connect to the correct gRPC server for the running network
	deployer := chain.NewLocalDeployerForNetwork(app, nodeVersion, vmBin, networkState.NetworkType)

	// Get genesis path
	genesisPath := app.GetGenesisPath(chainName)

	// Deploy to locally-running network (works for local, testnet, mainnet started via CLI)
	subnetID, blockchainID, err := deployer.DeployToLocalNetwork(chainName, chainGenesis, genesisPath)
	if err != nil {
		// Check if this is a DeploymentError (chain-specific failure)
		var deployErr *chain.DeploymentError
		if errors.As(err, &deployErr) {
			// Deployment failed but we can provide useful feedback
			ux.Logger.PrintError("\nChain deployment failed: %s", deployErr.Cause)
			if deployErr.NetworkHealthy {
				ux.Logger.PrintToUser("\nThe primary network is still running. You can:")
				ux.Logger.PrintToUser("  1. Fix the issue and retry: lux chain deploy %s", chainName)
				ux.Logger.PrintToUser("  2. Check logs: lux network status")
				ux.Logger.PrintToUser("  3. Stop the network: lux network stop")
			} else {
				ux.Logger.PrintError("\nThe network may have crashed. Check logs and restart:")
				ux.Logger.PrintToUser("  1. lux network stop")
				ux.Logger.PrintToUser("  2. lux network start --%s", network.String())
			}
			return err
		}
		// Non-deployment error (gRPC connection issue, etc)
		if deployer.BackendStartedHere() {
			if innerErr := binutils.KillgRPCServerProcessForNetwork(app, networkState.NetworkType); innerErr != nil {
				app.Log.Warn("failed to kill gRPC server", zap.Error(innerErr))
			}
		}
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Update sidecar with deployment info (using the target network)
	if err := app.UpdateSidecarNetworks(sc, network, subnetID, blockchainID); err != nil {
		return fmt.Errorf("failed to update sidecar: %w", err)
	}
	return nil
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
