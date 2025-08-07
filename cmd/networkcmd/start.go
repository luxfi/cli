// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/server"
	"github.com/luxfi/netrunner/utils"
	"github.com/spf13/cobra"
)

var (
	userProvidedLuxVersion string
	snapshotName           string
	mainnet                bool
	testnet                bool
	// BadgerDB flags
	dbEngine      string
	archiveDir    string
	archiveShared bool
	genesisImport string
)

const latest = "latest"

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
	cmd.Flags().BoolVar(&mainnet, "mainnet", false, "start a mainnet node with 21 validators")
	cmd.Flags().BoolVar(&testnet, "testnet", false, "start a testnet node with 11 validators")
	// BadgerDB flags
	cmd.Flags().StringVar(&dbEngine, "db-backend", "", "database backend to use (pebble, leveldb, or badgerdb)")
	cmd.Flags().StringVar(&archiveDir, "archive-path", "", "path to BadgerDB archive database (enables dual-database mode)")
	cmd.Flags().BoolVar(&archiveShared, "archive-shared", false, "enable shared read-only access to archive database")
	cmd.Flags().StringVar(&genesisImport, "genesis-path", "", "path to genesis database to import (PebbleDB or LevelDB)")

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

	outputDirPrefix := path.Join(app.GetRunDir(), "restart")
	outputDir, err := utils.MkDirWithTimestamp(outputDirPrefix)
	if err != nil {
		return err
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

	pp, err := cli.LoadSnapshot(
		ctx,
		snapshotName,
		loadSnapshotOpts...,
	)

	if err != nil {
		if !server.IsServerError(err, server.ErrAlreadyBootstrapped) {
			return fmt.Errorf("failed to start network with the persisted snapshot: %w", err)
		}
		ux.Logger.PrintToUser("Network has already been booted. Wait until healthy...")
	} else {
		ux.Logger.PrintToUser("Booting Network. Wait until healthy...")
		ux.Logger.PrintToUser("Node log path: %s/node<i>/logs", pp.ClusterInfo.RootDataDir)
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

// StartMainnet starts a mainnet network with 21 validator nodes
func StartMainnet() error {
	ux.Logger.PrintToUser("Starting Lux mainnet with 21 validator nodes...")

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

	// Build mainnet configuration with proper consensus parameters
	globalNodeConfig := `{
		"network-id": "96369",
		"db-type": "pebbledb",
		"staking-enabled": true,
		"sybil-protection-enabled": true,
		"health-check-frequency": "30s",
		"http-port": 9630,
		"staking-port": 9631,
		"consensus-sample-size": 21,
		"consensus-quorum-size": 15,
		"consensus-virtuous-commit-threshold": 21,
		"consensus-rogue-commit-threshold": 28,
		"consensus-concurrent-repolls": 3,
		"consensus-optimal-processing": 50,
		"consensus-max-processing": 3000,
		"consensus-max-time-processing": 3000000000,
		"stake-max-consumption-rate": 120000,
		"stake-min-consumption-rate": 100000,
		"stake-minted-reward": 64000000,
		"stake-supply-cap": 3000000000000000,
		"uptime-requirement": 0.8,
		"max-stake-duration": 31536000,
		"min-stake-duration": 1209600,
		"min-validator-stake": 2000000000000,
		"min-delegator-stake": 25000000000,
		"delegation-fee-cap": 200000,
		"stake-minting-period": 31536000,
		"log-level": "info"
	}`

	// Prepare chain configs with genesis files
	chainConfigs := map[string]string{}
	genesisPath := "/home/z/work/lux/genesis/configs/mainnet"

	// Read genesis files and add to chain configs
	for _, chain := range []string{"C", "P", "X"} {
		genesisFile := fmt.Sprintf("%s/%s/genesis.json", genesisPath, chain)
		if genesisData, err := os.ReadFile(genesisFile); err == nil {
			chainConfigs[chain] = string(genesisData)
		} else {
			ux.Logger.PrintToUser("Warning: Could not read %s genesis file: %v", chain, err)
		}
	}

	// Build start options
	rootDataDir := path.Join(app.GetRunDir(), "mainnet-"+time.Now().Format("20060102-150405"))

	opts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithNumNodes(21),
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

	ux.Logger.PrintToUser("Starting network with 21 validators...")
	ux.Logger.PrintToUser("Network ID: 96369")
	ux.Logger.PrintToUser("Root data directory: %s", rootDataDir)

	// Start the network
	startResp, err := cli.Start(ctx, path.Join(rootDataDir, "netrunner.log"), opts...)
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
			if statusResp.ClusterInfo.Healthy && len(statusResp.ClusterInfo.NodeInfos) == 21 {
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
	ux.Logger.PrintToUser("\nMainnet started successfully with 21 validators!")
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

// StartTestnet starts a testnet network with 11 validator nodes
func StartTestnet() error {
	ux.Logger.PrintToUser("Starting Lux testnet with 11 validator nodes...")

	// First, ensure we have validator keys generated
	keysDir := "/home/z/.luxd/keys/testnet"
	if _, err := os.Stat(keysDir + "/validator-0/staking.crt"); os.IsNotExist(err) {
		ux.Logger.PrintToUser("Generating validator keys for testnet...")
		genkeysCmd := exec.Command("/home/z/.luxd/genkeys", "testnet")
		genkeysCmd.Stdout = os.Stdout
		genkeysCmd.Stderr = os.Stderr
		if err := genkeysCmd.Run(); err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}
	}

	ux.Logger.PrintToUser("Using keys from %s", keysDir)

	// Build the command to start luxd with testnet configuration
	luxdPath := "/home/z/work/lux/node/bin/luxd"

	// Check if luxd exists
	if _, err := os.Stat(luxdPath); os.IsNotExist(err) {
		return fmt.Errorf("luxd binary not found at %s. Please build it first with 'make build-node'", luxdPath)
	}

	// Testnet configuration (faster parameters)
	args := []string{
		"--network-id=96368",
		"--staking-enabled=true",
		"--consensus-x-enabled=false",
		"--consensus-shutdown-timeout=3s",
		"--consensus-gossip-frequency=5s",
		"--consensus-k=11",
		"--consensus-alpha-preference=7",
		"--consensus-alpha-confidence=9",
		"--consensus-beta=5",
		"--consensus-concurrent-repolls=5",
		"--consensus-optimal-processing=5",
		"--consensus-max-processing-time=6300000000", // 6.3s in nanoseconds
		"--staking-tls-cert-file=/home/z/.luxd/keys/testnet/validator-0/staking.crt",
		"--staking-tls-key-file=/home/z/.luxd/keys/testnet/validator-0/staking.key",
		"--staking-signer-key-file=/home/z/.luxd/keys/testnet/validator-0/signer.key",
		"--genesis=/home/z/work/lux/genesis/configs/testnet",
		"--chain-data-dir=/home/z/.luxd/testnet-data",
		"--db-dir=/home/z/.luxd/testnet-db",
		"--log-dir=/home/z/.luxd/testnet-logs",
		"--http-port=9650",
		"--staking-port=9651",
	}

	// Execute luxd
	cmd := exec.Command(luxdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ux.Logger.PrintToUser("Starting luxd with command: %s %v", luxdPath, args)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %w", err)
	}

	ux.Logger.PrintToUser("Testnet started successfully!")
	ux.Logger.PrintToUser("RPC endpoint: http://localhost:9650")
	ux.Logger.PrintToUser("Node logs: ~/.luxd/testnet-logs/")
	return nil
}
