// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/rpcpb"
	"github.com/luxfi/netrunner/server"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

var (
	userProvidedLuxVersion string
	snapshotName           string
	mainnet                bool
	testnet                bool
	localNetwork           bool // Synonym for custom/default network (--local flag)
	numValidators          int
	nodePath               string // Path to custom luxd binary
	portBase               int    // Base port for nodes (each node uses 2 ports)
	// BadgerDB flags
	dbEngine      string
	archiveDir    string
	archiveShared bool
)

// StartFlags contains configuration for starting a network
type StartFlags struct {
	UserProvidedLuxdVersion string
	LuxdBinaryPath          string
	NumNodes                uint32
}

// Start starts the local network with the given flags
func Start(flags StartFlags, printEndpoints bool) error {
	// For now, just call StartNetwork with nil cmd and args
	return StartNetwork(nil, nil)
}

const nodeBinaryName = "luxd"

// findNodeBinary locates the node binary using the following priority:
// 1. User-provided --node-path flag
// 2. Node binary in PATH
// 3. Relative to CLI binary: ../node/build/<nodeBinaryName>
func findNodeBinary() (string, error) {
	// Priority 1: User-provided path via --node-path flag
	if nodePath != "" {
		if _, err := os.Stat(nodePath); os.IsNotExist(err) {
			return "", fmt.Errorf("%s binary not found at specified path: %s", nodeBinaryName, nodePath)
		}
		return nodePath, nil
	}

	// Priority 2: Check if node binary is in PATH
	if binaryPath, err := exec.LookPath(nodeBinaryName); err == nil {
		return binaryPath, nil
	}

	// Priority 3: Look relative to CLI binary location
	// Get the path of the current executable
	execPath, err := os.Executable()
	if err == nil {
		// Resolve any symlinks
		execPath, err = filepath.EvalSymlinks(execPath)
		if err == nil {
			// CLI is typically at cli/bin/lux, so node binary would be at ../node/build/<nodeBinaryName>
			cliDir := filepath.Dir(filepath.Dir(execPath)) // Go up two levels from bin/lux
			relativePath := filepath.Join(cliDir, "..", "node", "build", nodeBinaryName)
			if absPath, err := filepath.Abs(relativePath); err == nil {
				if _, err := os.Stat(absPath); err == nil {
					return absPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("%s binary not found. Please either:\n"+
		"  1. Use --node-path to specify the path to %s\n"+
		"  2. Add %s to your PATH\n"+
		"  3. Build %s in the sibling node directory (../node/build/%s)",
		nodeBinaryName, nodeBinaryName, nodeBinaryName, nodeBinaryName, nodeBinaryName)
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a local network",
		Long: `The network start command starts a local, multi-node Lux network on your machine.

By default, the command loads the default snapshot. If you provide the --snapshot-name
flag, the network loads that snapshot instead. The command fails if the local network is
already running.`,

		RunE:         StartNetwork,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&userProvidedLuxVersion, "node-version", "latest", "use this version of node (ex: v1.17.12)")
	cmd.Flags().StringVar(&nodePath, "node-path", "", "path to local luxd binary (overrides --node-version)")
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", constants.DefaultSnapshotName, "name of snapshot to use to start the network from")
	cmd.Flags().BoolVar(&mainnet, "mainnet", false, "start a mainnet node with 5 validators")
	cmd.Flags().BoolVar(&testnet, "testnet", false, "start a testnet node with 5 validators")
	cmd.Flags().BoolVar(&localNetwork, "local", false, "start a local dev network (synonym for default/custom)")
	cmd.Flags().IntVar(&numValidators, "num-validators", constants.LocalNetworkNumNodes, "number of validators to start")
	cmd.Flags().IntVar(&portBase, "port-base", 9630, "base port for node APIs (each node uses 2 ports: HTTP and staking)")
	// BadgerDB flags
	cmd.Flags().StringVar(&dbEngine, "db-backend", "", "database backend to use (pebble, leveldb, or badgerdb)")
	cmd.Flags().StringVar(&archiveDir, "archive-path", "", "path to BadgerDB archive database (enables dual-database mode)")
	cmd.Flags().BoolVar(&archiveShared, "archive-shared", false, "enable shared read-only access to archive database")

	// Add state loading flags
	AddStateFlags(cmd)

	return cmd
}

func StartNetwork(*cobra.Command, []string) error {
	// Check for conflicting flags
	flagCount := 0
	if mainnet {
		flagCount++
	}
	if testnet {
		flagCount++
	}
	if localNetwork {
		flagCount++
	}
	if flagCount > 1 {
		return fmt.Errorf("cannot use multiple network flags (--mainnet, --testnet, --local) together")
	}

	// If mainnet or testnet flag is set, delegate to the appropriate function
	if mainnet {
		return StartMainnet()
	}
	if testnet {
		return StartTestnet()
	}
	// --local flag just runs the default local network (same as no flag)
	luxVersion, err := determineLuxVersion(userProvidedLuxVersion)
	if err != nil {
		return err
	}

	// Create deployer for local network
	sd := chain.NewLocalDeployerForNetwork(app, luxVersion, "", "local")

	// Start gRPC server for local network on port 8099
	if err := sd.StartServerForNetwork("local"); err != nil {
		return err
	}

	nodeBinPath, err := sd.SetupLocalEnv()
	if err != nil {
		return err
	}

	// Connect to local network's gRPC server
	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType("local"))
	if err != nil {
		return err
	}
	defer cli.Close()

	var startMsg string
	if snapshotName == constants.DefaultSnapshotName {
		startMsg = "Starting previously deployed and stopped snapshot"
	} else {
		startMsg = fmt.Sprintf("Starting previously deployed and stopped snapshot %s...", snapshotName)
	}
	ux.Logger.PrintToUser("%s", startMsg)

	// Use stable directory with current symlink for persistence across restarts
	// This eliminates the gotcha where state is lost because each restart creates a new timestamped dir
	outputDir, err := chain.EnsureNetworkRunDir(app.GetRunDir(), "local")
	if err != nil {
		return fmt.Errorf("failed to ensure run directory: %w", err)
	}
	ux.Logger.PrintToUser("Using run directory: %s", outputDir)

	pluginDir := filepath.Join(app.GetPluginsDir(), "current")

	loadSnapshotOpts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithRootDataDir(outputDir),
		client.WithReassignPortsIfUsed(true),
		client.WithPluginDir(pluginDir),
	}

	// load global node configs if they exist
	configStr, err := app.Conf.LoadNodeConfig()
	if err != nil {
		return err
	}

	// Build node config with BadgerDB options
	nodeConfig := make(map[string]interface{})

	// Auto-track deployed nets - eliminates the track-chains gotcha
	netIDs, trackErr := chain.GetLocallyDeployedNetIDs(app)
	if trackErr == nil && len(netIDs) > 0 {
		trackNetsStr := strings.Join(netIDs, ",")
		ux.Logger.PrintToUser("Auto-tracking %d deployed net(s): %s", len(netIDs), trackNetsStr)
		// Add track-chains to node config (luxd still uses track-chains internally)
		nodeConfig["track-chains"] = trackNetsStr
	}

	// Prepare canonical chain configs directory and set it for all nodes
	// This must happen BEFORE nodes start so VMs can initialize with genesis configs
	chainConfigDir, chainConfigErr := chain.PrepareCanonicalChainConfigs(app)
	if chainConfigErr != nil {
		ux.Logger.PrintToUser("Warning: failed to prepare chain configs: %v", chainConfigErr)
	} else if chainConfigDir != "" {
		nodeConfig["chain-config-dir"] = chainConfigDir
	}
	if configStr != "" {
		if err := json.Unmarshal([]byte(configStr), &nodeConfig); err != nil {
			return fmt.Errorf("invalid node config: %w", err)
		}
	}

	// Add BadgerDB configuration if specified
	if dbEngine != "" {
		nodeConfig["db-type"] = dbEngine
	}
	if archiveDir != "" {
		nodeConfig["archive-dir"] = archiveDir
		nodeConfig["archive-shared"] = archiveShared
	}
	// NOTE: genesis-import, genesis-replay, genesis-verify flags were removed
	// as they don't exist in luxd. The --genesis-path flag is no longer functional.

	// Convert back to JSON
	if len(nodeConfig) > 0 {
		updatedConfigBytes, err := json.Marshal(nodeConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal node config: %w", err)
		}
		loadSnapshotOpts = append(loadSnapshotOpts, client.WithGlobalNodeConfig(string(updatedConfigBytes)))
	}

	ctx := binutils.GetAsyncContext()

	// Check if we have a valid snapshot with nodes (db directory with node subdirs)
	snapshotPath := path.Join(app.GetSnapshotsDir(), "anr-snapshot-"+snapshotName)
	dbPath := path.Join(snapshotPath, "db")
	hasValidSnapshot := false

	if fi, dbErr := os.Stat(dbPath); dbErr == nil && fi.IsDir() {
		entries, _ := os.ReadDir(dbPath)
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "node") {
				hasValidSnapshot = true
				break
			}
		}
	}

	var pp *rpcpb.LoadSnapshotResponse
	var loadErr error

	if hasValidSnapshot {
		// Load from existing snapshot
		pp, loadErr = cli.LoadSnapshot(
			ctx,
			snapshotName,
			loadSnapshotOpts...,
		)

		if loadErr != nil {
			if !server.IsServerError(loadErr, server.ErrAlreadyBootstrapped) {
				return fmt.Errorf("failed to start network with the persisted snapshot: %w", loadErr)
			}
			ux.Logger.PrintToUser("Network has already been booted. Wait until healthy...")
		} else {
			ux.Logger.PrintToUser("Booting Network. Wait until healthy...")
			ux.Logger.PrintToUser("Node log path: %s/node<i>/logs", pp.ClusterInfo.RootDataDir)

			// Load existing subnet state if provided
			if err := LoadExistingSubnetState(outputDir); err != nil {
				ux.Logger.PrintToUser("Warning: Failed to load existing subnet state: %v", err)
				// Continue without the state - don't fail the entire network start
			}
		}
	} else {
		// Start fresh network - no valid snapshot with nodes exists
		ux.Logger.PrintToUser("No valid snapshot found, starting fresh local network...")

		startOpts := []client.OpOption{
			client.WithExecPath(nodeBinPath),
			client.WithNumNodes(uint32(numValidators)),
			client.WithRootDataDir(outputDir),
			client.WithReassignPortsIfUsed(true),
			client.WithPluginDir(pluginDir),
		}

		// Add global node config if present
		if len(nodeConfig) > 0 {
			updatedConfigBytes, marshalErr := json.Marshal(nodeConfig)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal node config: %w", marshalErr)
			}
			startOpts = append(startOpts, client.WithGlobalNodeConfig(string(updatedConfigBytes)))
		}

		startResp, startErr := cli.Start(ctx, nodeBinPath, startOpts...)
		if startErr != nil {
			// Check if network is already bootstrapped (started via `network start --mainnet/--testnet`)
			if server.IsServerError(startErr, server.ErrAlreadyBootstrapped) {
				ux.Logger.PrintToUser("Network has already been started. Continuing with existing network...")
			} else {
				return fmt.Errorf("failed to start fresh network: %w", startErr)
			}
		} else {
			ux.Logger.PrintToUser("Fresh network started. Wait until healthy...")
			ux.Logger.PrintToUser("Node log path: %s/node<i>/logs", startResp.ClusterInfo.RootDataDir)
		}
	}

	clusterInfo, err := chain.WaitForHealthy(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %w", err)
	}

	fmt.Println()
	if chain.HasEndpoints(clusterInfo) {
		ux.Logger.PrintToUser("Network ready to use. Local network node endpoints:")
		ux.PrintTableEndpoints(clusterInfo)
	}

	// Save network state for deploy commands to find the running network
	// Local network uses default port 9630, network ID 1337, and local gRPC ports
	grpcPorts := binutils.GetGRPCPorts("local")
	networkState := application.CreateNetworkStateWithGRPC("local", constants.LocalNetworkID, 9630, grpcPorts.Server, grpcPorts.Gateway)
	if err := app.SaveNetworkState(networkState); err != nil {
		ux.Logger.PrintToUser("Warning: failed to save network state: %v", err)
	}

	return nil
}

func determineLuxVersion(userProvidedLuxVersion string) (string, error) {
	// a specific user provided version should override this calculation, so just return
	if userProvidedLuxVersion != "latest" {
		return userProvidedLuxVersion, nil
	}

	// Need to determine which chains have been deployed
	locallyDeployedChains, err := chain.GetLocallyDeployedSubnetsFromFile(app)
	if err != nil {
		return "", err
	}

	// if no chains have been deployed, use latest
	if len(locallyDeployedChains) == 0 {
		return "latest", nil
	}

	currentRPCVersion := -1

	// For each deployed chain, check RPC versions
	for _, deployedChain := range locallyDeployedChains {
		sc, err := app.LoadSidecar(deployedChain)
		if err != nil {
			return "", err
		}

		// if you have a custom vm, you must provide the version explicitly
		// if you upgrade from evm to a custom vm, the RPC version will be 0
		if sc.VM == models.CustomVM || sc.Networks[models.Local.String()].RPCVersion == 0 {
			continue
		}

		if currentRPCVersion == -1 {
			currentRPCVersion = sc.Networks[models.Local.String()].RPCVersion
		}

		if sc.Networks[models.Local.String()].RPCVersion != currentRPCVersion {
			return "", fmt.Errorf(
				"RPC version mismatch. Expected %d, got %d for chain %s. Upgrade all chains to the same RPC version to launch the network",
				currentRPCVersion,
				sc.RPCVersion,
				sc.Name,
			)
		}
	}

	// If currentRPCVersion == -1, then only custom chains have been deployed, the user must provide the version explicitly if not latest
	if currentRPCVersion == -1 {
		ux.Logger.PrintToUser("No chain RPC version found. Using latest Lux version")
		return "latest", nil
	}

	return vm.GetLatestLuxByProtocolVersion(
		app,
		currentRPCVersion,
		constants.LuxCompatibilityURL,
	)
}

// networkConfig holds configuration for starting a public network
type networkConfig struct {
	networkID   uint32
	networkName string // "mainnet" or "testnet"
	portBase    int    // Base port for APIs (defaults to 9630 for mainnet, 9640 for testnet)
}

// startPublicNetwork handles the common logic for starting mainnet/testnet
func startPublicNetwork(cfg networkConfig) error {
	if numValidators < 1 {
		numValidators = constants.LocalNetworkNumNodes
	}
	ux.Logger.PrintToUser("Starting Lux %s with %d validator nodes...", cfg.networkName, numValidators)
	ux.Logger.PrintToUser("Network ID: %d", cfg.networkID)

	localNodePath, err := findNodeBinary()
	if err != nil {
		return err
	}

	// Create deployer for the specific network type
	sd := chain.NewLocalDeployerForNetwork(app, "", "", cfg.networkName)
	if err := sd.StartServerForNetwork(cfg.networkName); err != nil {
		return err
	}

	// Connect to this network's gRPC server
	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(cfg.networkName))
	if err != nil {
		return err
	}
	defer cli.Close()

	// Build node config - auto-detect deployed subnets for tracking
	trackSubnets := ""
	netIDs, trackErr := chain.GetLocallyDeployedNetIDs(app)
	if trackErr == nil && len(netIDs) > 0 {
		trackSubnets = strings.Join(netIDs, ",")
		ux.Logger.PrintToUser("Auto-tracking %d deployed subnet(s): %s", len(netIDs), trackSubnets)
	}

	// Use "all" to auto-track all chains including newly deployed ones
	// This enables hot-loading of new chains without node restarts
	trackChainsValue := "all"
	if len(netIDs) > 0 {
		// If specific chains are configured, show them but still track all
		ux.Logger.PrintToUser("Found %d previously deployed chain(s)", len(netIDs))
	}

	// Use port base from config, default 9630 for mainnet, 9640 for testnet
	effectivePortBase := cfg.portBase
	if effectivePortBase == 0 {
		effectivePortBase = 9630
	}

	globalNodeConfig := fmt.Sprintf(`{
		"network-id": %d,
		"db-type": "badgerdb",
		"sybil-protection-enabled": true,
		"network-allow-private-ips": true,
		"health-check-frequency": "30s",
		"log-level": "info",
		"http-host": "127.0.0.1",
		"api-admin-enabled": true,
		"track-chains": %q
	}`, cfg.networkID, trackChainsValue)

	// Build per-node configs with explicit ports to avoid conflicts
	customNodeConfigs := make(map[string]string)
	for i := 0; i < numValidators; i++ {
		nodeName := fmt.Sprintf("node%d", i+1)
		httpPort := effectivePortBase + (i * 2)
		stakingPort := httpPort + 1
		customNodeConfigs[nodeName] = fmt.Sprintf(`{"http-port": %d, "staking-port": %d}`, httpPort, stakingPort)
	}

	rootDataDir, err := chain.EnsureNetworkRunDir(app.GetRunDir(), cfg.networkName)
	if err != nil {
		return fmt.Errorf("failed to ensure %s run directory: %w", cfg.networkName, err)
	}

	// Check for existing data or user-provided state
	if statePath != "" {
		ux.Logger.PrintToUser("Resuming from user-provided state: %s", statePath)
	} else {
		entries, _ := os.ReadDir(rootDataDir)
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "node") {
				ux.Logger.PrintToUser("Resuming from existing %s data: %s", cfg.networkName, rootDataDir)
				break
			}
		}
	}

	opts := []client.OpOption{
		client.WithNumNodes(uint32(numValidators)),
		client.WithGlobalNodeConfig(globalNodeConfig),
		client.WithRootDataDir(rootDataDir),
		client.WithReassignPortsIfUsed(true),
		client.WithDynamicPorts(false),
		client.WithCustomNodeConfigs(customNodeConfigs),
	}

	// Build chain configs (mainnet-specific feature, but harmless for testnet)
	cfgMgr := chain.NewManager(app)
	if err := cfgMgr.LoadDeployedChains(); err != nil {
		ux.Logger.PrintToUser("Warning: failed to load deployed chains: %v", err)
	}
	cfgMgr.EnableAdminAll()

	for chainID, cfg := range cfgMgr.ToNetrunnerMap() {
		if chainID != "C" {
			ux.Logger.PrintToUser("Configured chain: %s (admin API enabled)", chainID[:min(len(chainID), 12)])
			_ = cfg
		}
	}
	opts = append(opts, client.WithChainConfigs(cfgMgr.ToNetrunnerMap()))

	pluginDir := filepath.Join(app.GetPluginsDir(), "current")
	// Always ensure plugin dir exists and pass it to nodes
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory %s: %w", pluginDir, err)
	}
	opts = append(opts, client.WithPluginDir(pluginDir))

	ctx := binutils.GetAsyncContext()

	ux.Logger.PrintToUser("Starting network with genesis from luxfi/genesis package...")
	ux.Logger.PrintToUser("Using luxd binary: %s", localNodePath)
	ux.Logger.PrintToUser("Root data directory: %s", rootDataDir)

	startResp, err := cli.Start(ctx, localNodePath, opts...)
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	ux.Logger.PrintToUser("Waiting for all validators to become healthy...")
	clusterInfo, err := chain.WaitForHealthy(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %w", err)
	}

	// Capitalize first letter of network name
	displayName := strings.ToUpper(cfg.networkName[:1]) + cfg.networkName[1:]
	ux.Logger.PrintToUser("\n%s started successfully with %d validators!", displayName, numValidators)
	ux.Logger.PrintToUser("\nRPC Endpoints:")

	if startResp.ClusterInfo != nil && len(startResp.ClusterInfo.NodeNames) > 0 {
		for i, nodeName := range startResp.ClusterInfo.NodeNames {
			if nodeInfo, ok := startResp.ClusterInfo.NodeInfos[nodeName]; ok && nodeInfo != nil && nodeInfo.Uri != "" {
				ux.Logger.PrintToUser("  Validator %d: %s", i+1, nodeInfo.Uri)
			}
		}
	}

	if chain.HasEndpoints(clusterInfo) {
		ux.PrintTableEndpoints(clusterInfo)
	}

	ux.Logger.PrintToUser("\nData directory: %s", rootDataDir)
	ux.Logger.PrintToUser("C-Chain RPC: http://localhost:%d/ext/bc/C/rpc", effectivePortBase)
	ux.Logger.PrintToUser("Network is ready for use!")

	// Save network state for deploy commands to find the running network
	grpcPorts := binutils.GetGRPCPorts(cfg.networkName)
	networkState := application.CreateNetworkStateWithGRPC(cfg.networkName, cfg.networkID, effectivePortBase, grpcPorts.Server, grpcPorts.Gateway)
	if err := app.SaveNetworkState(networkState); err != nil {
		ux.Logger.PrintToUser("Warning: failed to save network state: %v", err)
	}
	ux.Logger.PrintToUser("gRPC server: localhost:%d", grpcPorts.Server)

	return nil
}

// StartMainnet starts a mainnet network with configurable validator nodes
func StartMainnet() error {
	// Use --port-base flag if provided, otherwise default to 9630
	pb := portBase
	if pb == 9630 && !isPortBaseFlagSet() {
		pb = 9630 // mainnet default
	}
	return startPublicNetwork(networkConfig{
		networkID:   constants.MainnetID, // P-Chain network ID (1)
		networkName: "mainnet",
		portBase:    pb,
	})
}

// StartTestnet starts a testnet network with configurable validator nodes
func StartTestnet() error {
	// Use --port-base flag if provided, otherwise default to 9640
	pb := portBase
	if pb == 9630 && !isPortBaseFlagSet() {
		pb = 9640 // testnet default (separate from mainnet)
	}
	return startPublicNetwork(networkConfig{
		networkID:   constants.TestnetID, // P-Chain network ID (2)
		networkName: "testnet",
		portBase:    pb,
	})
}

// isPortBaseFlagSet checks if --port-base was explicitly set by user
func isPortBaseFlagSet() bool {
	// If portBase != default, it was explicitly set
	return portBase != 9630
}
