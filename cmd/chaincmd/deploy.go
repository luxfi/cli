// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chaincmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"net"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/keychain"
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
	// RemoteProbeTimeout is the timeout for probing a remote network endpoint
	RemoteProbeTimeout = 30 * time.Second
)

var (
	deployLocal   bool
	deployTestnet bool
	deployMainnet bool
	deployDevnet  bool
	nodeVersion   string
	deployTimeout time.Duration
	deployKeyName string
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

  2. For local networks, network must be running:
     lux network start --devnet

  3. For remote networks (devnet, testnet, mainnet), a funded key is needed:
     Set LUX_MNEMONIC or LUX_PRIVATE_KEY env var, or use --key flag

  4. VM must be installed (for custom VMs):
     lux vm link "Lux EVM" --path ~/work/lux/evm/build/evm

OPTIONS:

  --node-version   Specific luxd version to use (default: latest)
  --key            Key name for remote network deployment (from ~/.lux/keys/)

EXAMPLES:

  # Deploy to remote devnet (auto-detects remote endpoint)
  lux chain deploy mychain --devnet

  # Deploy to remote devnet with specific key
  lux chain deploy mychain --devnet --key mykey

  # Deploy to local devnet (if local network is running)
  lux chain deploy mychain --devnet

  # Deploy to testnet
  lux chain deploy mychain --testnet
  lux chain deploy mychain -t

  # Deploy to mainnet
  lux chain deploy mychain --mainnet
  lux chain deploy mychain -m

  # Deploy with specific node version
  lux chain deploy mychain --devnet --node-version v1.11.0

DEPLOYMENT PROCESS:

  Local network:
  1. Validates chain configuration exists
  2. Verifies local gRPC network is running
  3. Checks VM plugin is installed
  4. Creates blockchain via netrunner gRPC
  5. Updates sidecar with deployment info

  Remote network:
  1. Validates chain configuration exists
  2. Probes remote endpoint (e.g., https://api.lux-dev.network)
  3. Creates chain on P-chain via wallet transaction
  4. Creates blockchain on P-chain via wallet transaction
  5. Updates sidecar with deployment info

OUTPUT:

  On success, displays:
  - Blockchain ID
  - Chain ID
  - RPC endpoints

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
	cmd.Flags().StringVar(&deployKeyName, "key", "", "Key name for remote network deployment (from ~/.lux/keys/)")

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

// getRemoteEndpoint returns the well-known remote API endpoint for a network type.
// Returns empty string for local/custom networks that have no remote endpoint.
func getRemoteEndpoint(network models.Network) string {
	return network.Endpoint()
}

// probeRemoteEndpoint checks if a remote network endpoint is alive by hitting /ext/info.
// Returns true if the endpoint responds to a JSON-RPC request.
func probeRemoteEndpoint(endpoint string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), RemoteProbeTimeout)
	defer cancel()

	url := endpoint + "/ext/info"
	body := []byte(`{"jsonrpc":"2.0","method":"info.getNodeVersion","params":{},"id":1}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a client with short per-IP dial timeout and skip TLS verify.
	// DNS may return multiple IPs where some are unreachable.
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DialContext:        dialer.DialContext,
			ForceAttemptHTTP2:  true,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		ux.Logger.PrintToUser("  probe error: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Any response (even 4xx for missing method) means the node is alive
	return resp.StatusCode < 500
}

// isRemoteCapableNetwork returns true if the network can be deployed to via remote P-chain API
func isRemoteCapableNetwork(network models.Network) bool {
	return network == models.Devnet || network == models.Testnet || network == models.Mainnet
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
		// State file read error (not just missing) - only fail if no remote fallback
		if !isRemoteCapableNetwork(network) {
			return fmt.Errorf("failed to load network state: %w\nIs the network running? Start with: lux network start", stateErr)
		}
		app.Log.Debug("Failed to load network state, will try remote endpoint", "error", stateErr)
		networkState = nil
	}

	// For remote-capable networks, try the remote endpoint if:
	// 1. No local state exists, OR
	// 2. Local state exists but has a remote API endpoint (e.g., https://...), OR
	// 3. Local state claims running but the state file is stale (gRPC server dead)
	if isRemoteCapableNetwork(network) {
		// For devnet/testnet/mainnet, ALWAYS try the remote endpoint first.
		// These are real networks at api.lux-dev.network, api.lux-test.network,
		// api.lux.network — not local. Local state is irrelevant.
		remoteEndpoint := getRemoteEndpoint(network)
		if remoteEndpoint != "" {
			ux.Logger.PrintToUser("Probing remote %s endpoint: %s", targetType, remoteEndpoint)
			if probeRemoteEndpoint(remoteEndpoint) {
				ux.Logger.PrintToUser("Remote %s is alive at %s", targetType, remoteEndpoint)
				return deployToRemoteNetwork(chainName, chainGenesis, sc, network, remoteEndpoint)
			}
			ux.Logger.PrintToUser("Remote endpoint %s is not reachable, falling back to local network", remoteEndpoint)
		}
	}

	// Local network path - requires running gRPC netrunner
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

	return deployToLocalNetwork(chainName, chainGenesis, sc, network, networkState)
}

// deployToLocalNetwork deploys a chain to a locally-running network managed by the CLI's gRPC netrunner.
func deployToLocalNetwork(chainName string, chainGenesis []byte, sc *models.Sidecar, network models.Network, networkState *application.NetworkState) error {
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

// deployToRemoteNetwork deploys a chain to a remote network via P-chain API transactions.
// This is used when no local gRPC netrunner is running but the remote network is reachable.
func deployToRemoteNetwork(chainName string, chainGenesis []byte, sc *models.Sidecar, network models.Network, endpoint string) error {
	ux.Logger.PrintToUser("Deploying to remote %s via P-chain API at %s", network.String(), endpoint)

	// Get keychain for signing P-chain transactions
	networkID := network.ID()
	kc, err := getDeployKeychain(network, networkID)
	if err != nil {
		return fmt.Errorf("failed to get keychain for deployment: %w\n\nTo fix, set LUX_MNEMONIC or LUX_PRIVATE_KEY env var, or use --key flag", err)
	}

	// Show the deployer address
	addrs := kc.Keychain.Addresses().List()
	if len(addrs) == 0 {
		return fmt.Errorf("keychain has no addresses")
	}

	// Create the public deployer
	deployer := chain.NewPublicDeployer(app, kc.UsesLedger, kc.Keychain, network)

	// Step 1: Create chain (P-chain transaction)
	ux.Logger.PrintToUser("Creating chain on P-chain...")
	controlKeys, err := kc.PChainFormattedStrAddresses()
	if err != nil {
		return fmt.Errorf("failed to get P-chain addresses: %w", err)
	}
	ux.Logger.PrintToUser("Control keys: %v", controlKeys)

	chainID, err := deployer.DeployChain(controlKeys, uint32(len(controlKeys)))
	if err != nil {
		return fmt.Errorf("failed to create chain: %w", err)
	}
	ux.Logger.PrintToUser("Chain created: %s", chainID.String())

	// Step 2: Create blockchain (P-chain transaction)
	ux.Logger.PrintToUser("Creating blockchain on chain %s...", chainID.String())
	isFullySigned, blockchainID, _, _, err := deployer.DeployBlockchain(
		controlKeys,
		controlKeys,
		chainID,
		chainName,
		chainGenesis,
	)
	if err != nil {
		return fmt.Errorf("failed to create blockchain: %w", err)
	}
	if !isFullySigned {
		return fmt.Errorf("blockchain transaction requires additional signatures (multisig not yet supported for remote deploy)")
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Blockchain deployed successfully!")
	ux.Logger.PrintToUser("  Chain ID:      %s", chainID.String())
	ux.Logger.PrintToUser("  Blockchain ID: %s", blockchainID.String())
	ux.Logger.PrintToUser("  RPC Endpoint:  %s/ext/bc/%s/rpc", endpoint, blockchainID.String())
	ux.Logger.PrintToUser("")

	// Update sidecar with deployment info
	if err := app.UpdateSidecarNetworks(sc, network, chainID, blockchainID); err != nil {
		return fmt.Errorf("failed to update sidecar: %w", err)
	}
	return nil
}

// getDeployKeychain obtains a keychain for remote network deployment.
// Priority:
//  1. --key flag (explicit key name)
//  2. LUX_PRIVATE_KEY env var
//  3. LUX_MNEMONIC env var
//  4. Interactive prompt (if terminal available)
func getDeployKeychain(network models.Network, networkID uint32) (*keychain.Keychain, error) {
	// If --key flag specified, use that key
	if deployKeyName != "" {
		return keychain.GetKeychain(app, false, false, nil, deployKeyName, network, 0)
	}

	// Try environment variables (LUX_PRIVATE_KEY, LUX_MNEMONIC)
	sf, err := key.GetOrCreateLocalKey(networkID)
	if err == nil && sf != nil {
		kc := sf.KeyChain()
		wrappedKc := keychain.WrapSecp256k1fxKeychain(kc)
		pAddrs := sf.P()
		if len(pAddrs) > 0 {
			ux.Logger.PrintToUser("Using key with P-Chain address: %s", pAddrs[0])
		}
		return keychain.NewKeychain(network, wrappedKc, nil, nil), nil
	}

	// Fall back to interactive prompt via GetKeychainFromCmdLineFlags
	return keychain.GetKeychainFromCmdLineFlags(app, "deploy chain to "+network.String(), network, "", false, false, nil, 0)
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
