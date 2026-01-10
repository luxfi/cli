// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chaincmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/localnetworkinterface"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

// Default timeouts for chain deployment
const (
	// DefaultDeployTimeout is the maximum time to wait for chain deployment to complete.
	// For local networks, this should be fast (<30s). Longer means something is wrong.
	DefaultDeployTimeout = 30 * time.Second
	// MaxConsecutiveHealthFailures is the number of consecutive health check failures before failing fast
	MaxConsecutiveHealthFailures = 10
	// LuxEVMName is the canonical name for the Lux EVM
	LuxEVMName = "Lux EVM"
)

var (
	deployLocal   bool
	deployTestnet bool
	deployMainnet bool
	deployDevnet  bool
	nodeVersion   string
	deployTimeout time.Duration
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [chainName]",
		Short: "Deploy a blockchain to local network, testnet, or mainnet",
		Long: `Deploy a configured blockchain to the network.

OVERVIEW:

  Deploys a blockchain configuration to a running network. The blockchain
  must be created first with 'lux chain create'. The target network must
  be running before deployment.

NETWORK FLAGS (choose one):

  --mainnet, -m    Deploy to mainnet (port 9630, Network ID 1)
  --testnet, -t    Deploy to testnet (port 9640, Network ID 2)
  --devnet, -d     Deploy to devnet (port 9650, Network ID 3)
  --local, -l      Deploy to local/custom network

  Default: --local (deploys to custom/local network)

PREREQUISITES:

  1. Chain must be created:
     lux chain create mychain

  2. Network must be running:
     lux network start --devnet

  3. VM must be installed (for custom VMs):
     lux vm link "Lux EVM" --path ~/work/lux/evm/build/evm

OPTIONS:

  --node-version   Specific luxd version to use (default: latest)

EXAMPLES:

  # Deploy to local devnet (most common)
  lux chain deploy mychain --devnet
  lux chain deploy mychain -d

  # Deploy to testnet
  lux chain deploy mychain --testnet
  lux chain deploy mychain -t

  # Deploy to mainnet
  lux chain deploy mychain --mainnet
  lux chain deploy mychain -m

  # Deploy with specific node version
  lux chain deploy mychain --devnet --node-version v1.11.0

DEPLOYMENT PROCESS:

  1. Validates chain configuration exists
  2. Verifies network is running
  3. Checks VM plugin is installed
  4. Creates blockchain on the network
  5. Updates sidecar with deployment info (chain ID, blockchain ID)
  6. Returns endpoints for the deployed chain

OUTPUT:

  On success, displays:
  - Blockchain ID
  - Chain ID
  - RPC endpoints for each validator node

TROUBLESHOOTING:

  "Network not running" → Start network first:
    lux network start --devnet

  "Chain mychain not found" → Create chain first:
    lux chain create mychain

  "VM not installed" → Link VM binary:
    lux vm link "Lux EVM" --path ~/path/to/evm

  "RPC version mismatch" → Chain VM version incompatible with running node

NOTES:

  - Deployment info is saved to the chain's sidecar.json
  - Same chain can be deployed to multiple networks
  - Each deployment gets unique blockchain ID
  - Use 'lux network status' to see deployed chain endpoints`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         deployChain,
	}

	cmd.Flags().BoolVarP(&deployLocal, "local", "l", false, "Deploy to local/custom network")
	cmd.Flags().BoolVarP(&deployTestnet, "testnet", "t", false, "Deploy to testnet")
	cmd.Flags().BoolVarP(&deployMainnet, "mainnet", "m", false, "Deploy to mainnet")
	cmd.Flags().BoolVarP(&deployDevnet, "devnet", "d", false, "Deploy to devnet")
	cmd.Flags().StringVar(&nodeVersion, "node-version", "latest", "Node version to use")
	cmd.Flags().DurationVar(&deployTimeout, "timeout", DefaultDeployTimeout, "Maximum time to wait for chain deployment (e.g., 60s, 2m)")

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
	case deployDevnet:
		network = models.Devnet
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
	vmName := LuxEVMName // Default for EVM chains
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
	app.Log.Debug("Checking plugin path", "path", pluginPath, "vmid", vmIDStr, "pluginsDir", app.GetCurrentPluginsDir())

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
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("VM plugin at %s is not executable", pluginPath)
	}

	app.Log.Debug("VM plugin verified", "vmid", vmIDStr, "path", pluginPath)
	return nil
}

// getVMDisplayName returns a human-readable name for the VM type
func getVMDisplayName(vm models.VMType) string {
	switch vm {
	case models.EVM:
		return LuxEVMName
	case models.CustomVM:
		return "Custom VM"
	default:
		return string(vm)
	}
}

func deployToNetwork(chainName string, chainGenesis []byte, sc *models.Sidecar, network models.Network) error {
	app.Log.Debug("Deploy to network", "network", network.String())

	// Map deploy target to network type
	// Default is "custom" (not "local" which is ambiguous - any network can run locally)
	targetType := "custom"
	switch network {
	case models.Testnet:
		targetType = "testnet"
	case models.Mainnet:
		targetType = "mainnet"
	case models.Devnet:
		targetType = "devnet"
	case models.Local:
		targetType = "custom"
	}

	// Load network state for the specific target network type
	// Each network type (custom, testnet, mainnet) has its own state file
	networkState, stateErr := app.LoadNetworkStateForType(targetType)
	if stateErr != nil {
		return fmt.Errorf("failed to load network state: %w\nIs the network running? Start with: lux network start", stateErr)
	}
	if networkState == nil || !networkState.Running {
		startHint := "lux network start"
		switch targetType {
		case "testnet":
			startHint = "lux network start --testnet"
		case "mainnet":
			startHint = "lux network start --mainnet"
		case "devnet":
			startHint = "lux network start --devnet"
		}
		return fmt.Errorf("no %s network running. Start the network first with: %s", targetType, startHint)
	}

	// Log gRPC port being used
	app.Log.Debug("Using gRPC port from network state", "port", networkState.GRPCPort, "network", networkState.NetworkType)

	// Preflight check: verify VM is installed before any network operations
	if err := verifyVMInstalled(chainName, sc); err != nil {
		return err
	}

	// Get VM binary - prefer linked plugin over downloaded
	var vmBin string
	var err error

	// Compute VMID for plugin lookup
	vmName := LuxEVMName
	if sc.VM == models.CustomVM {
		vmName = chainName
	}
	vmID, _ := utils.VMID(vmName)
	vmIDStr := vmID.String()

	switch sc.VM {
	case models.EVM:
		// First check if EVM plugin already exists (linked or copied)
		pluginPath := filepath.Join(app.GetCurrentPluginsDir(), vmIDStr)
		if info, pluginErr := os.Stat(pluginPath); pluginErr == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			// Plugin exists and is executable, use it directly
			vmBin = pluginPath
			app.Log.Debug("Using existing EVM plugin", "path", vmBin)
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
	chainID, blockchainID, err := deployer.DeployToLocalNetwork(chainName, chainGenesis, genesisPath)
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
				app.Log.Warn("failed to kill gRPC server", "error", innerErr)
			}
		}
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Update sidecar with deployment info (using the target network)
	if err := app.UpdateSidecarNetworks(sc, network, chainID, blockchainID); err != nil {
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
