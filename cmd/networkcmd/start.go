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
	"time"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/net"
	"github.com/luxfi/cli/pkg/ux"
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
	numValidators          int
	nodePath               string // Path to custom luxd binary
	// BadgerDB flags
	dbEngine      string
	archiveDir    string
	archiveShared bool
	genesisImport string
)

// StartFlags contains configuration for starting a network
type StartFlags struct {
	UserProvidedAvagoVersion string
	AvagoBinaryPath          string
	UserProvidedLuxdVersion  string
	LuxdBinaryPath           string
	NumNodes                 uint32
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

	cmd.Flags().StringVar(&userProvidedLuxVersion, "node-version", latest, "use this version of node (ex: v1.17.12)")
	cmd.Flags().StringVar(&nodePath, "node-path", "", "path to local luxd binary (overrides --node-version)")
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", constants.DefaultSnapshotName, "name of snapshot to use to start the network from")
	cmd.Flags().BoolVar(&mainnet, "mainnet", false, "start a mainnet node with 5 validators")
	cmd.Flags().BoolVar(&testnet, "testnet", false, "start a testnet node with 5 validators")
	cmd.Flags().IntVar(&numValidators, "num-validators", constants.LocalNetworkNumNodes, "number of validators to start")
	// BadgerDB flags
	cmd.Flags().StringVar(&dbEngine, "db-backend", "", "database backend to use (pebble, leveldb, or badgerdb)")
	cmd.Flags().StringVar(&archiveDir, "archive-path", "", "path to BadgerDB archive database (enables dual-database mode)")
	cmd.Flags().BoolVar(&archiveShared, "archive-shared", false, "enable shared read-only access to archive database")
	cmd.Flags().StringVar(&genesisImport, "genesis-path", "", "path to genesis database to import (PebbleDB or LevelDB)")

	// Add state loading flags
	AddStateFlags(cmd)

	return cmd
}

func StartNetwork(*cobra.Command, []string) error {
	// Check for conflicting flags
	if mainnet && testnet {
		return fmt.Errorf("cannot use both --mainnet and --testnet flags")
	}

	// If mainnet or testnet flag is set, delegate to the appropriate function
	if mainnet {
		return StartMainnet()
	}
	if testnet {
		return StartTestnet()
	}
	luxVersion, err := determineLuxVersion(userProvidedLuxVersion)
	if err != nil {
		return err
	}

	sd := net.NewLocalDeployer(app, luxVersion, "")

	if err := sd.StartServer(); err != nil {
		return err
	}

	nodeBinPath, err := sd.SetupLocalEnv()
	if err != nil {
		return err
	}

	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	var startMsg string
	if snapshotName == constants.DefaultSnapshotName {
		startMsg = "Starting previously deployed and stopped snapshot"
	} else {
		startMsg = fmt.Sprintf("Starting previously deployed and stopped snapshot %s...", snapshotName)
	}
	ux.Logger.PrintToUser("%s", startMsg)

	// Use stable directory path for persistence across restarts
	// This eliminates the gotcha where state is lost because each restart creates a new timestamped dir
	outputDir := path.Join(app.GetRunDir(), "local_network")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	pluginDir := app.GetPluginsDir()

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

	// Auto-track deployed nets - eliminates the track-subnets gotcha
	netIDs, trackErr := net.GetLocallyDeployedNetIDs(app)
	if trackErr == nil && len(netIDs) > 0 {
		trackNetsStr := strings.Join(netIDs, ",")
		ux.Logger.PrintToUser("Auto-tracking %d deployed net(s): %s", len(netIDs), trackNetsStr)
		// Add track-subnets to node config (luxd still uses track-subnets internally)
		nodeConfig["track-subnets"] = trackNetsStr
	}

	// Prepare canonical chain configs directory and set it for all nodes
	// This must happen BEFORE nodes start so VMs can initialize with genesis configs
	chainConfigDir, chainConfigErr := net.PrepareCanonicalChainConfigs(app)
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
		nodeConfig["db-engine"] = dbEngine
	}
	if archiveDir != "" {
		nodeConfig["archive-dir"] = archiveDir
		nodeConfig["archive-shared"] = archiveShared
	}
	if genesisImport != "" {
		nodeConfig["genesis-import"] = genesisImport
		nodeConfig["genesis-replay"] = true
		nodeConfig["genesis-verify"] = true
	}

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

	clusterInfo, err := net.WaitForHealthy(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %w", err)
	}

	fmt.Println()
	if net.HasEndpoints(clusterInfo) {
		ux.Logger.PrintToUser("Network ready to use. Local network node endpoints:")
		ux.PrintTableEndpoints(clusterInfo)
	}

	return nil
}

func determineLuxVersion(userProvidedLuxVersion string) (string, error) {
	// a specific user provided version should override this calculation, so just return
	if userProvidedLuxVersion != latest {
		return userProvidedLuxVersion, nil
	}

	// Need to determine which subnets have been deployed
	locallyDeployedSubnets, err := net.GetLocallyDeployedSubnetsFromFile(app)
	if err != nil {
		return "", err
	}

	// if no subnets have been deployed, use latest
	if len(locallyDeployedSubnets) == 0 {
		return latest, nil
	}

	currentRPCVersion := -1

	// For each deployed subnet, check RPC versions
	for _, deployedSubnet := range locallyDeployedSubnets {
		sc, err := app.LoadSidecar(deployedSubnet)
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
				"RPC version mismatch. Expected %d, got %d for Subnet %s. Upgrade all subnets to the same RPC version to launch the network",
				currentRPCVersion,
				sc.RPCVersion,
				sc.Name,
			)
		}
	}

	// If currentRPCVersion == -1, then only custom subnets have been deployed, the user must provide the version explicitly if not latest
	if currentRPCVersion == -1 {
		ux.Logger.PrintToUser("No Subnet RPC version found. Using latest Lux version")
		return latest, nil
	}

	return vm.GetLatestLuxByProtocolVersion(
		app,
		currentRPCVersion,
		constants.LuxCompatibilityURL,
	)
}

// StartMainnet starts a mainnet network with configurable validator nodes
// Uses netrunner internally with genesis configuration from luxfi/genesis package
func StartMainnet() error {
	if numValidators < 1 {
		numValidators = constants.LocalNetworkNumNodes
	}
	ux.Logger.PrintToUser("Starting Lux mainnet with %d validator nodes...", numValidators)
	ux.Logger.PrintToUser("Network ID: 96369")

	localNodePath, err := findNodeBinary()
	if err != nil {
		return err
	}

	// Use local binary instead of downloading
	sd := net.NewLocalDeployer(app, "", "")

	// Start netrunner server
	if err := sd.StartServer(); err != nil {
		return err
	}

	// Get gRPC client
	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	// Build mainnet configuration - netrunner will use luxfi/genesis for network ID 96369
	// This triggers local.NewMainnetConfig() in netrunner which uses configs.GetGenesis(96369)
	// Note: staking is enabled by default in luxd, no flag needed
	// http-host is required for RPC to be accessible
	globalNodeConfig := `{
		"network-id": 96369,
		"db-type": "pebbledb",
		"sybil-protection-enabled": true,
		"network-allow-private-ips": true,
		"health-check-frequency": "30s",
		"log-level": "info",
		"http-host": "127.0.0.1",
		"api-admin-enabled": true
	}`

	// Build start options
	rootDataDir := path.Join(app.GetRunDir(), "mainnet-"+time.Now().Format("20060102-150405"))

	// Don't pass networkID to netrunner - let --dev flag handle the genesis internally
	// The --dev flag creates a proper single-node development network

	opts := []client.OpOption{
		client.WithNumNodes(uint32(numValidators)),
		client.WithGlobalNodeConfig(globalNodeConfig),
		client.WithRootDataDir(rootDataDir),
		client.WithReassignPortsIfUsed(true),
		client.WithDynamicPorts(false), // Use fixed ports starting from 9630
	}

	// Add plugin directory if it exists
	pluginDir := app.GetPluginsDir()
	if _, err := os.Stat(pluginDir); err == nil {
		opts = append(opts, client.WithPluginDir(pluginDir))
	}

	ctx := binutils.GetAsyncContext()

	ux.Logger.PrintToUser("Starting network with genesis from luxfi/genesis package...")
	ux.Logger.PrintToUser("Using luxd binary: %s", localNodePath)
	ux.Logger.PrintToUser("Root data directory: %s", rootDataDir)

	// Start the network - netrunner handles genesis via luxfi/genesis
	// First arg is exec path (luxd binary), not log path
	startResp, err := cli.Start(ctx, localNodePath, opts...)
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// Wait for healthy network
	ux.Logger.PrintToUser("Waiting for all validators to become healthy...")
	clusterInfo, err := net.WaitForHealthy(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %w", err)
	}

	// Display endpoints
	ux.Logger.PrintToUser("\nMainnet started successfully with %d validators!", numValidators)
	ux.Logger.PrintToUser("\nRPC Endpoints:")

	if startResp.ClusterInfo != nil && len(startResp.ClusterInfo.NodeNames) > 0 {
		for i, nodeName := range startResp.ClusterInfo.NodeNames {
			if nodeInfo, ok := startResp.ClusterInfo.NodeInfos[nodeName]; ok && nodeInfo != nil && nodeInfo.Uri != "" {
				ux.Logger.PrintToUser("  Validator %d: %s", i+1, nodeInfo.Uri)
			}
		}
	}

	// Show table of endpoints
	if net.HasEndpoints(clusterInfo) {
		ux.PrintTableEndpoints(clusterInfo)
	}

	ux.Logger.PrintToUser("\nData directory: %s", rootDataDir)
	ux.Logger.PrintToUser("C-Chain RPC: http://localhost:9630/ext/bc/C/rpc")
	ux.Logger.PrintToUser("Network is ready for use!")

	return nil
}

// StartTestnet starts a testnet network with configurable validator nodes
// Uses netrunner internally with genesis configuration from luxfi/genesis package
func StartTestnet() error {
	if numValidators < 1 {
		numValidators = constants.LocalNetworkNumNodes
	}
	ux.Logger.PrintToUser("Starting Lux testnet with %d validator nodes...", numValidators)
	ux.Logger.PrintToUser("Network ID: 96368")

	localNodePath, err := findNodeBinary()
	if err != nil {
		return err
	}

	// Use local binary instead of downloading
	sd := net.NewLocalDeployer(app, "", "")

	// Start netrunner server
	if err := sd.StartServer(); err != nil {
		return err
	}

	// Get gRPC client
	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	// Build testnet configuration - netrunner will use luxfi/genesis for network ID 96368
	// This triggers local.NewTestnetConfig() in netrunner which uses configs.GetGenesis(96368)
	// Note: staking is enabled by default in luxd, no flag needed
	// http-host is required for RPC to be accessible
	globalNodeConfig := `{
		"network-id": 96368,
		"db-type": "pebbledb",
		"sybil-protection-enabled": true,
		"health-check-frequency": "30s",
		"log-level": "info",
		"http-host": "127.0.0.1",
		"api-admin-enabled": true
	}`

	// Build start options
	rootDataDir := path.Join(app.GetRunDir(), "testnet-"+time.Now().Format("20060102-150405"))

	opts := []client.OpOption{
		client.WithNumNodes(uint32(numValidators)),
		client.WithGlobalNodeConfig(globalNodeConfig),
		client.WithRootDataDir(rootDataDir),
		client.WithReassignPortsIfUsed(true),
		client.WithDynamicPorts(false), // Use fixed ports starting from 9630
	}

	// Add plugin directory if it exists
	pluginDir := app.GetPluginsDir()
	if _, err := os.Stat(pluginDir); err == nil {
		opts = append(opts, client.WithPluginDir(pluginDir))
	}

	ctx := binutils.GetAsyncContext()

	ux.Logger.PrintToUser("Starting network with genesis from luxfi/genesis package...")
	ux.Logger.PrintToUser("Using luxd binary: %s", localNodePath)
	ux.Logger.PrintToUser("Root data directory: %s", rootDataDir)

	// Start the network - netrunner handles genesis via luxfi/genesis
	// First arg is exec path (luxd binary), not log path
	startResp, err := cli.Start(ctx, localNodePath, opts...)
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// Wait for healthy network
	ux.Logger.PrintToUser("Waiting for all validators to become healthy...")
	clusterInfo, err := net.WaitForHealthy(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %w", err)
	}

	// Display endpoints
	ux.Logger.PrintToUser("\nTestnet started successfully with %d validators!", numValidators)
	ux.Logger.PrintToUser("\nRPC Endpoints:")

	if startResp.ClusterInfo != nil && len(startResp.ClusterInfo.NodeNames) > 0 {
		for i, nodeName := range startResp.ClusterInfo.NodeNames {
			if nodeInfo, ok := startResp.ClusterInfo.NodeInfos[nodeName]; ok && nodeInfo != nil && nodeInfo.Uri != "" {
				ux.Logger.PrintToUser("  Validator %d: %s", i+1, nodeInfo.Uri)
			}
		}
	}

	// Show table of endpoints
	if net.HasEndpoints(clusterInfo) {
		ux.PrintTableEndpoints(clusterInfo)
	}

	ux.Logger.PrintToUser("\nData directory: %s", rootDataDir)
	ux.Logger.PrintToUser("C-Chain RPC: http://localhost:9630/ext/bc/C/rpc")
	ux.Logger.PrintToUser("Network is ready for use!")

	return nil
}
