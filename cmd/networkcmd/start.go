// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/subnet"
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
	// BadgerDB flags
	dbEngine      string
	archiveDir    string
	archiveShared bool
	genesisImport string
)

const latest = "latest"

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
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", constants.DefaultSnapshotName, "name of snapshot to use to start the network from")
	cmd.Flags().BoolVar(&mainnet, "mainnet", false, "start mainnet network")
	cmd.Flags().BoolVar(&testnet, "testnet", false, "start testnet network")
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

	sd := subnet.NewLocalDeployer(app, luxVersion, "")

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

	// Auto-track deployed subnets - eliminates the track-subnets gotcha
	subnetIDs, trackErr := subnet.GetLocallyDeployedSubnetIDs(app)
	if trackErr == nil && len(subnetIDs) > 0 {
		trackSubnetsStr := strings.Join(subnetIDs, ",")
		ux.Logger.PrintToUser("Auto-tracking %d deployed subnet(s): %s", len(subnetIDs), trackSubnetsStr)
		// Add track-subnets to node config
		nodeConfig["track-subnets"] = trackSubnetsStr
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

	clusterInfo, err := subnet.WaitForHealthy(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %w", err)
	}

	fmt.Println()
	if subnet.HasEndpoints(clusterInfo) {
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
	locallyDeployedSubnets, err := subnet.GetLocallyDeployedSubnetsFromFile(app)
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
func StartMainnet() error {
	if numValidators < 1 {
		numValidators = constants.LocalNetworkNumNodes
	}
	ux.Logger.PrintToUser("Starting Lux mainnet with %d validators...", numValidators)

	// Check if local luxd binary exists
	localLuxdPath := "/home/z/work/lux/node/build/luxd"
	if _, err := os.Stat(localLuxdPath); os.IsNotExist(err) {
		return fmt.Errorf("luxd binary not found at %s. Please run 'make build-node' first", localLuxdPath)
	}

	// Use local binary instead of downloading
	sd := subnet.NewLocalDeployer(app, "", "")

	// Start netrunner server
	if err := sd.StartServer(); err != nil {
		return err
	}

	// Use local binary path
	nodeBinPath := localLuxdPath

	// Get gRPC client
	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	// Historic chaindata setup
	historicChainData := "/home/z/work/lux/mainnet-data/chainData/C/db"
	if _, err := os.Stat(historicChainData); err == nil {
		ux.Logger.PrintToUser("Found historic C-Chain data at %s", historicChainData)
		// Copy to first validator's data directory will be handled post-launch
	}

	// Build mainnet configuration for single-node local development
	// Use real mainnet with staking keys and k=1 consensus parameters
	// Note: http-port and staking-port are managed by netrunner, don't set them here
	// skip-bootstrap=true is now safe because the node's createDAG() has been fixed
	// to properly initialize X-Chain and C-Chain in skip-bootstrap mode
	globalNodeConfig := `{
		"log-level": "info",
		"network-id": 96369,
		"sybil-protection-enabled": false,
		"network-health-min-conn-peers": 0
	}`

	// C-Chain runtime config (not genesis)
	chainConfigs := map[string]string{
		"C": `{
			"pruning-enabled": false,
			"local-txs-enabled": true,
			"allow-unprotected-txs": true,
			"state-sync-enabled": false,
			"eth-apis": ["eth", "personal", "admin", "debug", "web3", "net", "txpool"]
		}`,
	}

	// Build start options
	rootDataDir := path.Join(app.GetRunDir(), "mainnet-"+time.Now().Format("20060102-150405"))

	// Don't pass networkID to netrunner - let --dev flag handle the genesis internally
	// The --dev flag creates a proper single-node development network

	opts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithNumNodes(uint32(numValidators)),
		// Don't use WithNetworkId - use --dev mode for local testing
		client.WithGlobalNodeConfig(globalNodeConfig),
		client.WithRootDataDir(rootDataDir),
		client.WithReassignPortsIfUsed(true),
		client.WithDynamicPorts(false), // Use fixed ports starting from 9630
		client.WithChainConfigs(chainConfigs),
	}

	// Add plugin directory if it exists
	pluginDir := path.Join(app.GetPluginsDir(), "evm")
	if _, err := os.Stat(pluginDir); err == nil {
		opts = append(opts, client.WithPluginDir(pluginDir))
	}

	ctx := binutils.GetAsyncContext()

	ux.Logger.PrintToUser("Starting network with %d validators...", numValidators)
	ux.Logger.PrintToUser("Network ID: 96369")
	ux.Logger.PrintToUser("Root data directory: %s", rootDataDir)

	// Start the network - first parameter is execPath (luxd binary)
	startResp, err := cli.Start(ctx, nodeBinPath, opts...)
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// Wait for healthy network
	ux.Logger.PrintToUser("Waiting for all validators to become healthy...")
	healthCheckStart := time.Now()
	healthy := false

	for !healthy && time.Since(healthCheckStart) < 5*time.Minute {
		statusResp, err := cli.Status(ctx)
		if err == nil && statusResp != nil && statusResp.ClusterInfo != nil {
			// Check if cluster itself is healthy
			if statusResp.ClusterInfo.Healthy && len(statusResp.ClusterInfo.NodeInfos) == numValidators {
				healthy = true
				break
			}
			ux.Logger.PrintToUser("Waiting for cluster to become healthy... (%d nodes)", len(statusResp.ClusterInfo.NodeInfos))
		}
		time.Sleep(5 * time.Second)
	}

	if !healthy {
		return fmt.Errorf("network failed to become healthy after 5 minutes")
	}

	// Copy historic chaindata to first validator
	if _, err := os.Stat(historicChainData); err == nil && len(startResp.ClusterInfo.NodeNames) > 0 {
		firstNodeName := startResp.ClusterInfo.NodeNames[0]
		if firstNodeInfo, ok := startResp.ClusterInfo.NodeInfos[firstNodeName]; ok && firstNodeInfo != nil {
			firstNodeDataDir := firstNodeInfo.DbDir
			targetCChainDir := path.Join(firstNodeDataDir, "C")

			if _, err := os.Stat(targetCChainDir); os.IsNotExist(err) {
				ux.Logger.PrintToUser("Copying historic C-Chain data to first validator...")
				cmd := exec.Command("cp", "-r", historicChainData, targetCChainDir)
				if err := cmd.Run(); err != nil {
					ux.Logger.PrintToUser("Warning: Failed to copy historic chaindata: %v", err)
				} else {
					ux.Logger.PrintToUser("Historic chaindata copied successfully")
				}
			}
		}
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

		// Get first node's URI
		if firstNodeInfo, ok := startResp.ClusterInfo.NodeInfos[startResp.ClusterInfo.NodeNames[0]]; ok && firstNodeInfo != nil {
			ux.Logger.PrintToUser("\nPrimary RPC endpoint: %s", firstNodeInfo.Uri)
		}
	}

	ux.Logger.PrintToUser("\nData directory: %s", rootDataDir)
	ux.Logger.PrintToUser("Network is ready for use!")

	// Save local network metadata so deploy commands can find the network
	// The networkDir is in the format rootDataDir/network_timestamp
	// Find the actual network directory
	networkDir := ""
	entries, err := os.ReadDir(rootDataDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "network_") {
				networkDir = path.Join(rootDataDir, entry.Name())
				break
			}
		}
	}
	if networkDir != "" {
		// Create tmpnet-compatible config.json for blockchain deploy commands
		if err := writeTmpnetConfig(networkDir, startResp.ClusterInfo, nodeBinPath); err != nil {
			ux.Logger.PrintToUser("Warning: Failed to write tmpnet config: %v", err)
		} else {
			ux.Logger.PrintToUser("Network config written to %s/config.json", networkDir)
		}

		if err := localnet.SaveLocalNetworkMeta(app, networkDir); err != nil {
			ux.Logger.PrintToUser("Warning: Failed to save network metadata: %v", err)
		} else {
			ux.Logger.PrintToUser("Network metadata saved to %s", networkDir)
		}
	}

	return nil
}

// tmpnetNetworkConfig represents the config.json format expected by tmpnet.ReadNetwork
type tmpnetNetworkConfig struct {
	UUID                 string                 `json:"UUID"`
	NetworkID            uint32                 `json:"NetworkID"`
	Owner                string                 `json:"Owner"`
	Genesis              json.RawMessage        `json:"Genesis,omitempty"`
	DefaultFlags         map[string]string      `json:"DefaultFlags"`
	DefaultRuntimeConfig tmpnetRuntimeConfig    `json:"DefaultRuntimeConfig,omitempty"`
	Nodes                []tmpnetNodeConfig     `json:"Nodes"`
}

type tmpnetRuntimeConfig struct {
	Process *tmpnetProcessConfig `json:"process,omitempty"`
}

type tmpnetProcessConfig struct {
	LuxPath string `json:"luxPath,omitempty"`
}

type tmpnetNodeConfig struct {
	DataDir string            `json:"DataDir"`
	Flags   map[string]string `json:"Flags,omitempty"`
}

// tmpnetNodeFileConfig represents the per-node config.json format (with flags and runtimeConfig)
type tmpnetNodeFileConfig struct {
	Flags         map[string]string   `json:"flags"`
	RuntimeConfig tmpnetRuntimeConfig `json:"runtimeConfig"`
}

// writeTmpnetConfig creates a config.json file compatible with tmpnet.ReadNetwork
func writeTmpnetConfig(networkDir string, clusterInfo *rpcpb.ClusterInfo, luxdPath string) error {
	// Build nodes list from cluster info
	nodes := make([]tmpnetNodeConfig, 0)
	if clusterInfo != nil {
		for _, nodeName := range clusterInfo.NodeNames {
			// Node directories are named node1, node2, etc.
			nodes = append(nodes, tmpnetNodeConfig{
				DataDir: nodeName, // Relative path like "node1"
				Flags:   map[string]string{},
			})
		}
	}

	// Read genesis from first node if available
	var genesisData json.RawMessage
	if len(nodes) > 0 {
		genesisPath := path.Join(networkDir, nodes[0].DataDir, "genesis.json")
		if data, err := os.ReadFile(genesisPath); err == nil {
			genesisData = data
		}
	}

	config := tmpnetNetworkConfig{
		UUID:      fmt.Sprintf("mainnet-%d-deploy", 96369),
		NetworkID: 96369,
		Owner:     "lux-cli",
		Genesis:   genesisData,
		DefaultFlags: map[string]string{
			"network-id": "96369",
		},
		DefaultRuntimeConfig: tmpnetRuntimeConfig{
			Process: &tmpnetProcessConfig{
				LuxPath: luxdPath,
			},
		},
		Nodes: nodes,
	}

	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := path.Join(networkDir, "config.json")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Write per-node config.json files
	if clusterInfo != nil && clusterInfo.NodeInfos != nil {
		for i, nodeName := range clusterInfo.NodeNames {
			nodeInfo := clusterInfo.NodeInfos[nodeName]
			if nodeInfo == nil {
				continue
			}

			nodeDir := path.Join(networkDir, nodeName)
			genesisFile := path.Join(nodeDir, "genesis.json")

			// Calculate ports based on node index
			httpPort := 9630 + (i * 2)
			stakingPort := 9631 + (i * 2)

			// Read staking credentials
			var stakingTLSKey, stakingTLSCert, stakingSignerKey string

			// Try to read staking key from node directory (base64 encoded content)
			if keyData, err := os.ReadFile(path.Join(nodeDir, "staking.key")); err == nil {
				stakingTLSKey = base64Encode(keyData)
			}
			if certData, err := os.ReadFile(path.Join(nodeDir, "staking.crt")); err == nil {
				stakingTLSCert = base64Encode(certData)
			}
			if signerData, err := os.ReadFile(path.Join(nodeDir, "signer.key")); err == nil {
				stakingSignerKey = base64Encode(signerData)
			}

			nodeConfig := tmpnetNodeFileConfig{
				Flags: map[string]string{
					"data-dir":                       nodeDir,
					"network-id":                     "96369",
					"http-port":                      fmt.Sprintf("%d", httpPort),
					"staking-port":                   fmt.Sprintf("%d", stakingPort),
					"genesis-file":                   genesisFile,
					"staking-tls-key-file-content":   stakingTLSKey,
					"staking-tls-cert-file-content":  stakingTLSCert,
					"staking-signer-key-file-content": stakingSignerKey,
				},
				RuntimeConfig: tmpnetRuntimeConfig{
					Process: &tmpnetProcessConfig{
						LuxPath: luxdPath,
					},
				},
			}

			nodeConfigBytes, err := json.MarshalIndent(nodeConfig, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal node config: %w", err)
			}

			nodeConfigPath := path.Join(nodeDir, "config.json")
			if err := os.WriteFile(nodeConfigPath, nodeConfigBytes, 0644); err != nil {
				return fmt.Errorf("failed to write node config: %w", err)
			}
		}
	}

	return nil
}

// base64Encode encodes bytes to base64 string
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// StartTestnet starts a testnet network with configurable validator nodes
func StartTestnet() error {
	if numValidators < 1 {
		numValidators = constants.LocalNetworkNumNodes
	}
	ux.Logger.PrintToUser("Starting Lux testnet with %d validators...", numValidators)

	// Check if local luxd binary exists
	localLuxdPath := "/home/z/work/lux/node/build/luxd"
	if _, err := os.Stat(localLuxdPath); os.IsNotExist(err) {
		return fmt.Errorf("luxd binary not found at %s. Please run 'make build-node' first", localLuxdPath)
	}

	// Use local binary instead of downloading
	sd := subnet.NewLocalDeployer(app, "", "")

	// Start netrunner server
	if err := sd.StartServer(); err != nil {
		return err
	}

	// Use local binary path
	nodeBinPath := localLuxdPath

	// Get gRPC client
	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	// Build testnet configuration for local development
	// Use testnet with staking keys and sybil protection disabled
	globalNodeConfig := `{
		"log-level": "info",
		"network-id": 96368,
		"sybil-protection-enabled": false,
		"network-health-min-conn-peers": 0
	}`

	// C-Chain runtime config (not genesis)
	chainConfigs := map[string]string{
		"C": `{
			"pruning-enabled": false,
			"local-txs-enabled": true,
			"allow-unprotected-txs": true,
			"state-sync-enabled": false,
			"eth-apis": ["eth", "personal", "admin", "debug", "web3", "net", "txpool"]
		}`,
	}

	// Build start options
	rootDataDir := path.Join(app.GetRunDir(), "testnet-"+time.Now().Format("20060102-150405"))

	opts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithNumNodes(uint32(numValidators)),
		client.WithGlobalNodeConfig(globalNodeConfig),
		client.WithRootDataDir(rootDataDir),
		client.WithReassignPortsIfUsed(true),
		client.WithDynamicPorts(false), // Use fixed ports starting from 9630
		client.WithChainConfigs(chainConfigs),
	}

	// Add plugin directory if it exists
	pluginDir := path.Join(app.GetPluginsDir(), "evm")
	if _, err := os.Stat(pluginDir); err == nil {
		opts = append(opts, client.WithPluginDir(pluginDir))
	}

	ctx := binutils.GetAsyncContext()

	ux.Logger.PrintToUser("Starting network with %d validators...", numValidators)
	ux.Logger.PrintToUser("Network ID: 96368")
	ux.Logger.PrintToUser("Root data directory: %s", rootDataDir)

	// Start the network
	startResp, err := cli.Start(ctx, nodeBinPath, opts...)
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// Wait for healthy network
	ux.Logger.PrintToUser("Waiting for all validators to become healthy...")
	healthCheckStart := time.Now()
	healthy := false

	for !healthy && time.Since(healthCheckStart) < 5*time.Minute {
		statusResp, err := cli.Status(ctx)
		if err == nil && statusResp != nil && statusResp.ClusterInfo != nil {
			if statusResp.ClusterInfo.Healthy && len(statusResp.ClusterInfo.NodeInfos) == numValidators {
				healthy = true
				break
			}
			ux.Logger.PrintToUser("Waiting for cluster to become healthy... (%d nodes)", len(statusResp.ClusterInfo.NodeInfos))
		}
		time.Sleep(5 * time.Second)
	}

	if !healthy {
		return fmt.Errorf("network failed to become healthy after 5 minutes")
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

		// Get first node's URI
		if firstNodeInfo, ok := startResp.ClusterInfo.NodeInfos[startResp.ClusterInfo.NodeNames[0]]; ok && firstNodeInfo != nil {
			ux.Logger.PrintToUser("\nPrimary RPC endpoint: %s", firstNodeInfo.Uri)
		}
	}

	ux.Logger.PrintToUser("\nData directory: %s", rootDataDir)
	ux.Logger.PrintToUser("Network is ready for use!")

	return nil
}
