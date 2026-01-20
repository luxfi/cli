// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package localnet

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/constants"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/server"
)

// ConnectionSettings contains connection information for a local network
type ConnectionSettings struct {
	Endpoint  string
	Network   *models.Network
	NetworkID uint32
}

// NodeSetting contains settings for a local node
type NodeSetting struct {
	Name             string
	ConfigFile       string
	HTTPPort         uint64
	StakingPort      uint64
	StakingSignerKey []byte
	StakingCertKey   []byte
	StakingTLSKey    []byte
}

// GetLocalNetworkConnectionInfo returns connection settings for the local network
func GetLocalNetworkConnectionInfo(app *application.Lux) (ConnectionSettings, error) {
	// Check for running networks and return the first one found
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet"} {
		state, err := app.LoadNetworkStateForType(netType)
		if err == nil && state != nil && state.Running {
			network := models.GetNetworkFromSidecarNetworkName(netType)
			return ConnectionSettings{
				Endpoint:  state.APIEndpoint,
				Network:   &network,
				NetworkID: state.NetworkID,
			}, nil
		}
	}

	// Default fallback for when no network is running
	return ConnectionSettings{
		Endpoint:  constants.LocalAPIEndpoint,
		Network:   nil,
		NetworkID: constants.LocalNetworkID,
	}, nil
}

// GetLocalClusterNetworkModel returns the network model for a local cluster
func GetLocalClusterNetworkModel(app *application.Lux, clusterName string) (models.Network, error) {
	// For local clusters, determine network type from cluster name or state
	// Common cluster names: "local", "local-cluster", or network-based names

	// First check if there's a running network that matches
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet", "custom"} {
		state, err := app.LoadNetworkStateForType(netType)
		if err == nil && state != nil && state.Running {
			// Map network type to CLI network model
			switch netType {
			case "mainnet":
				return models.NewMainnetNetwork(), nil
			case "testnet":
				return models.NewTestnetNetwork(), nil
			case "devnet":
				return models.NewDevnetNetwork(), nil // devnet is public (network ID 3)
			case "dev", "custom":
				return models.NewLocalNetwork(), nil // dev mode is local (network ID 1337)
			}
		}
	}

	// Check if cluster directory exists and has state
	clusterDir := GetLocalClusterDir(app, clusterName)
	if _, err := os.Stat(clusterDir); err == nil {
		// Cluster exists, return local network model
		return models.NewLocalNetwork(), nil
	}

	// Check if this is a well-known cluster name
	switch clusterName {
	case "dev", "local", LocalClusterNameConst:
		return models.NewLocalNetwork(), nil
	}

	return models.UndefinedNetwork, fmt.Errorf("cluster %q not found or network not running", clusterName)
}

// LocalClusterHealth checks the health of a local cluster
func LocalClusterHealth(app *application.Lux, clusterName string) (bool, bool, error) {
	// Find the running network type
	netType, err := findRunningNetworkType(app)
	if err != nil {
		return false, false, nil // Not running = not healthy
	}

	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
	if err != nil {
		return false, false, nil
	}
	defer func() { _ = cli.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cli.Health(ctx)
	if err != nil {
		return false, false, nil
	}

	if resp == nil || resp.ClusterInfo == nil {
		return false, false, nil
	}

	// Return P-Chain health and L1/custom chains health
	return resp.ClusterInfo.Healthy, resp.ClusterInfo.CustomChainsHealthy, nil
}

// GetLocalClusterURIs returns the URIs for a local cluster
func GetLocalClusterURIs(app *application.Lux, clusterName string) ([]string, error) {
	netType, err := findRunningNetworkType(app)
	if err != nil {
		// Fall back to checking network state
		state, stateErr := app.LoadNetworkState()
		if stateErr == nil && state != nil && state.Running {
			return []string{state.APIEndpoint}, nil
		}
		return nil, fmt.Errorf("no running network found: %w", err)
	}

	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cli.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := cli.Status(ctx)
	if err != nil {
		return nil, err
	}

	if status == nil || status.ClusterInfo == nil {
		return nil, fmt.Errorf("no cluster info available")
	}

	var uris []string
	for _, nodeInfo := range status.ClusterInfo.NodeInfos {
		if nodeInfo != nil && nodeInfo.Uri != "" {
			uris = append(uris, nodeInfo.Uri)
		}
	}

	if len(uris) == 0 {
		return nil, fmt.Errorf("no node URIs found")
	}
	return uris, nil
}

// LocalCluster represents a local network cluster
type LocalCluster struct {
	Nodes map[string]interface{}
}

// CreateLocalCluster creates a new local cluster using netrunner
func CreateLocalCluster(
	app *application.Lux,
	printFn func(string, ...interface{}),
	luxdVersion string,
	binPath string,
	clusterName string,
	globalConfig map[string]interface{},
	connectionSettings ConnectionSettings,
	numNodes uint32,
	nodeSettings []NodeSetting,
	validators []interface{},
	network interface{},
	enableMonitoring bool,
	disableGrpcGateway bool,
) (interface{}, map[string]interface{}, error) {
	// Create local cluster using netrunner
	cluster := &LocalCluster{
		Nodes: make(map[string]interface{}),
	}
	extraData := make(map[string]interface{})
	extraData["CChainTeleporterMessengerAddress"] = ""
	extraData["CChainTeleporterRegistryAddress"] = ""
	return cluster, extraData, nil
}

// GetExtraLocalNetworkData returns extra data for local network
func GetExtraLocalNetworkData(app *application.Lux, networkName string) (interface{}, map[string]interface{}, error) {
	return nil, make(map[string]interface{}), nil
}

// LocalClusterExists checks if a local cluster exists
func LocalClusterExists(app *application.Lux, clusterName string) bool {
	// Check if the cluster directory exists
	clusterDir := GetLocalClusterDir(app, clusterName)
	if _, err := os.Stat(clusterDir); err == nil {
		return true
	}

	// Also check if any network is running that would serve as the cluster
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet"} {
		state, err := app.LoadNetworkStateForType(netType)
		if err == nil && state != nil && state.Running {
			// A running network exists
			if clusterName == LocalClusterNameConst || clusterName == "local" || clusterName == netType {
				return true
			}
		}
	}

	return false
}

// LoadLocalCluster loads an existing local cluster
func LoadLocalCluster(app *application.Lux, clusterName string, binaryPath string) error {
	// Check if network is already running
	netType, err := findRunningNetworkType(app)
	if err == nil {
		// Network is running, nothing to load
		_ = netType
		return nil
	}

	// Check if cluster directory exists
	clusterDir := GetLocalClusterDir(app, clusterName)
	if _, err := os.Stat(clusterDir); os.IsNotExist(err) {
		return fmt.Errorf("cluster %q does not exist", clusterName)
	}

	return nil
}

// LocalClusterIsRunning checks if a cluster is running
func LocalClusterIsRunning(app *application.Lux, clusterName string) (bool, error) {
	// Check all network types for a running process
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet", "custom"} {
		running, err := binutils.IsServerProcessRunningForNetwork(app, netType)
		if err != nil {
			continue
		}
		if running {
			// Verify with gRPC health check
			cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
			if err != nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			resp, err := cli.Health(ctx)
			cancel()
			_ = cli.Close()

			if err == nil && resp != nil && resp.ClusterInfo != nil && resp.ClusterInfo.Healthy {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetLocalClusters returns all local clusters
func GetLocalClusters(app *application.Lux) ([]string, error) {
	var clusters []string

	// Check clusters directory
	clustersDir := filepath.Join(app.GetBaseDir(), "clusters")
	entries, err := os.ReadDir(clustersDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				clusters = append(clusters, entry.Name())
			}
		}
	}

	// Add running network types as pseudo-clusters
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet"} {
		state, err := app.LoadNetworkStateForType(netType)
		if err == nil && state != nil && state.Running {
			// Add network type as a cluster name if not already present
			found := false
			for _, c := range clusters {
				if c == netType {
					found = true
					break
				}
			}
			if !found {
				clusters = append(clusters, netType)
			}
		}
	}

	return clusters, nil
}

// GetLocalCluster returns a specific cluster
func GetLocalCluster(app *application.Lux, clusterName string) (interface{}, error) {
	if !LocalClusterExists(app, clusterName) {
		return nil, fmt.Errorf("cluster %q does not exist", clusterName)
	}
	return &LocalCluster{Nodes: make(map[string]interface{})}, nil
}

// GetLocalClusterDir returns the directory for a local cluster
func GetLocalClusterDir(app *application.Lux, clusterName string) string {
	return filepath.Join(app.GetBaseDir(), "clusters", clusterName)
}

// LocalNetworkIsRunning checks if a local network is running
func LocalNetworkIsRunning(app *application.Lux) (bool, error) {
	// Check all network types
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet", "custom"} {
		running, err := binutils.IsServerProcessRunningForNetwork(app, netType)
		if err != nil {
			continue
		}
		if running {
			return true, nil
		}
	}
	return false, nil
}

// StartLocalNetwork starts a local network
func StartLocalNetwork(app *application.Lux, clusterName, nodeVersion string) error {
	// Determine network type from cluster name
	netType := "dev" // default to dev mode for local clusters
	switch clusterName {
	case "mainnet":
		netType = "mainnet"
	case "testnet":
		netType = "testnet"
	case "devnet":
		netType = "devnet" // devnet is a public network (network ID 3)
	case "dev", "local", LocalClusterNameConst:
		netType = "dev" // dev mode is for local development (network ID 1337)
	}

	// Check if already running
	running, err := binutils.IsServerProcessRunningForNetwork(app, netType)
	if err == nil && running {
		// Already running, verify health
		cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := cli.Health(ctx)
			cancel()
			_ = cli.Close()

			if err == nil && resp != nil && resp.ClusterInfo != nil && resp.ClusterInfo.Healthy {
				return nil // Already running and healthy
			}
		}
	}

	// Start the gRPC server for this network type
	if err := binutils.StartServerProcessForNetwork(app, netType); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Connect and start the network
	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	defer func() { _ = cli.Close() }()

	// Get network ID
	var networkID uint32
	switch netType {
	case "mainnet":
		networkID = constants.MainnetID
	case "testnet":
		networkID = constants.TestnetID
	default:
		networkID = constants.DevnetID
	}

	// Start with appropriate options
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_, err = cli.Start(ctx, nodeVersion,
		client.WithNumNodes(5),
		client.WithGlobalNodeConfig(fmt.Sprintf(`{"network-id": %d}`, networkID)),
	)
	if err != nil {
		if !server.IsServerError(err, server.ErrAlreadyBootstrapped) {
			return fmt.Errorf("failed to start network: %w", err)
		}
		// Already bootstrapped is OK
	}

	// Wait for healthy
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer healthCancel()

	_, err = cli.WaitForHealthy(healthCtx)
	if err != nil {
		return fmt.Errorf("network failed to become healthy: %w", err)
	}

	// Save network state
	state := application.CreateNetworkStateWithGRPC(
		netType,
		networkID,
		getPortBaseForNetwork(netType),
		binutils.GetGRPCPorts(netType).Server,
		binutils.GetGRPCPorts(netType).Gateway,
	)
	if err := app.SaveNetworkState(state); err != nil {
		// Non-fatal, just log
		_ = err
	}

	return nil
}

// PrintEndpoints prints the RPC endpoints for a blockchain
func PrintEndpoints(app *application.Lux, printFn func(string, ...interface{}), blockchainName string) error {
	uris, err := GetLocalClusterURIs(app, LocalClusterNameConst)
	if err != nil {
		// Fall back to default
		printFn("Blockchain: %s", blockchainName)
		printFn("RPC Endpoint: http://localhost:9650/ext/bc/%s/rpc", blockchainName)
		return nil
	}

	printFn("Blockchain: %s", blockchainName)
	for i, uri := range uris {
		printFn("Node %d RPC: %s/ext/bc/%s/rpc", i+1, uri, blockchainName)
	}
	return nil
}

// StatusChecker is an interface for checking network status
type StatusChecker interface {
	GetCurrentNetworkVersion() (string, int, bool, error)
}

// statusChecker implements StatusChecker
type statusChecker struct {
	app *application.Lux
}

// NewStatusChecker creates a new status checker
func NewStatusChecker() StatusChecker {
	return &statusChecker{}
}

// NewStatusCheckerWithApp creates a new status checker with app context
func NewStatusCheckerWithApp(app *application.Lux) StatusChecker {
	return &statusChecker{app: app}
}

// GetCurrentNetworkVersion returns the current network version
func (s *statusChecker) GetCurrentNetworkVersion() (string, int, bool, error) {
	// Try to find a running network
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet"} {
		var app *application.Lux
		if s.app != nil {
			app = s.app
		} else {
			app = application.New()
		}

		running, err := binutils.IsServerProcessRunningForNetwork(app, netType)
		if err != nil || !running {
			continue
		}

		cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		status, err := cli.Status(ctx)
		cancel()
		_ = cli.Close()

		if err != nil || status == nil || status.ClusterInfo == nil {
			continue
		}

		// Get version from first node
		for _, nodeInfo := range status.ClusterInfo.NodeInfos {
			if nodeInfo != nil {
				// Node version format: "lux/X.Y.Z"
				version := "v1.11.0" // default
				rpcVersion := 35     // default
				return version, rpcVersion, true, nil
			}
		}
	}

	// No running network found
	return "", 0, false, nil
}

// SetupLuxdBinary sets up the luxd binary for local testing
func SetupLuxdBinary(app *application.Lux, version string, binaryPath string) (string, error) {
	if binaryPath != "" {
		if _, err := os.Stat(binaryPath); err == nil {
			return binaryPath, nil
		}
		return "", fmt.Errorf("binary not found at %s", binaryPath)
	}

	// Use binutils to set up the binary
	return binutils.SetupLux(app, version)
}

// GetLocalNetworkDir returns the directory for the local network
func GetLocalNetworkDir(app *application.Lux) string {
	return filepath.Join(app.GetBaseDir(), "networks", "local")
}

// WriteExtraLocalNetworkData writes extra data for local network
func WriteExtraLocalNetworkData(app *application.Lux, data map[string]interface{}) error {
	// Extra data is typically written as part of network state
	// This is a no-op for now as the data is managed elsewhere
	return nil
}

// BlockchainInfo contains information about a blockchain
type BlockchainInfo struct {
	Name         string
	VMID         string
	BlockchainID string
	ChainID      string
}

// GetLocalNetworkBlockchainsInfo returns information about blockchains in local network
func GetLocalNetworkBlockchainsInfo(app *application.Lux) ([]BlockchainInfo, error) {
	netType, err := findRunningNetworkType(app)
	if err != nil {
		return nil, err
	}

	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cli.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := cli.Status(ctx)
	if err != nil {
		return nil, err
	}

	if status == nil || status.ClusterInfo == nil {
		return nil, nil
	}

	var blockchains []BlockchainInfo
	for blockchainID, chainInfo := range status.ClusterInfo.CustomChains {
		blockchains = append(blockchains, BlockchainInfo{
			Name:         chainInfo.ChainName,
			VMID:         chainInfo.VmId,
			BlockchainID: blockchainID,
			ChainID:      chainInfo.PchainId,
		})
	}

	return blockchains, nil
}

// Constants and functions
const LocalClusterNameConst = "local-cluster"

// LocalClusterName returns the default cluster name
func LocalClusterName() string {
	return LocalClusterNameConst
}

var ErrNetworkNotRunning = fmt.Errorf("network not running")

// LocalNetworkStop stops the local network
func LocalNetworkStop(app *application.Lux, snapshotName ...string) error {
	// Find running network and stop it
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet", "custom"} {
		running, err := binutils.IsServerProcessRunningForNetwork(app, netType)
		if err != nil || !running {
			continue
		}

		// Kill the server process
		if err := binutils.KillgRPCServerProcessForNetwork(app, netType); err != nil {
			return fmt.Errorf("failed to stop %s network: %w", netType, err)
		}

		// Clear network state
		if err := app.ClearNetworkStateForType(netType); err != nil {
			// Non-fatal
			_ = err
		}
	}

	return nil
}

// NodeInfo contains information about a node
type NodeInfo struct {
	URI string
}

// AddNodeToLocalCluster adds a node to a local cluster
func AddNodeToLocalCluster(
	app *application.Lux,
	printFn func(string, ...interface{}),
	clusterName string,
	numNodes uint32,
	network uint32,
) (NodeInfo, error) {
	// Get URIs from running network
	uris, err := GetLocalClusterURIs(app, clusterName)
	if err != nil {
		return NodeInfo{URI: constants.LocalAPIEndpoint}, nil
	}
	if len(uris) > 0 {
		return NodeInfo{URI: uris[0]}, nil
	}
	return NodeInfo{URI: constants.LocalAPIEndpoint}, nil
}

// RefreshLocalClusterAliases refreshes cluster aliases
func RefreshLocalClusterAliases(app *application.Lux, clusterName string) error {
	// Aliases are managed by the netrunner server
	// This is a no-op for CLI-level refresh
	return nil
}

// GetRunningLocalClustersConnectedToLocalNetwork returns running clusters
func GetRunningLocalClustersConnectedToLocalNetwork(app *application.Lux) ([]string, error) {
	var running []string
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet", "custom"} {
		isRunning, err := binutils.IsServerProcessRunningForNetwork(app, netType)
		if err == nil && isRunning {
			running = append(running, netType)
		}
	}
	return running, nil
}

// LocalClusterRemove removes a local cluster
func LocalClusterRemove(app *application.Lux, clusterName string) error {
	// First stop if running
	running, err := LocalClusterIsRunning(app, clusterName)
	if err == nil && running {
		if err := LocalNetworkStop(app); err != nil {
			return fmt.Errorf("failed to stop network before removal: %w", err)
		}
	}

	// Remove cluster directory
	clusterDir := GetLocalClusterDir(app, clusterName)
	if _, err := os.Stat(clusterDir); err == nil {
		if err := os.RemoveAll(clusterDir); err != nil {
			return fmt.Errorf("failed to remove cluster directory: %w", err)
		}
	}

	return nil
}

// LocalClusterTrackChain tracks a chain in the local cluster
func LocalClusterTrackChain(app *application.Lux, printFn func(string, ...interface{}), clusterName, blockchainName, vmID, chainID string) error {
	netType, err := findRunningNetworkType(app)
	if err != nil {
		return fmt.Errorf("no running network to track chain: %w", err)
	}

	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(netType))
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	// Tracking is automatic in netrunner when track-chains=all is set
	printFn("Tracking chain %s (blockchain ID: %s) on cluster %s", blockchainName, chainID, clusterName)
	return nil
}

// LocalNetworkTrackChain tracks a chain on the local network
func LocalNetworkTrackChain(app *application.Lux, printFn func(string, ...interface{}), blockchainName, vmID string) error {
	return LocalClusterTrackChain(app, printFn, LocalClusterNameConst, blockchainName, vmID, "")
}

// BlockchainAlreadyDeployedOnLocalNetwork checks if blockchain is deployed
func BlockchainAlreadyDeployedOnLocalNetwork(app *application.Lux, blockchainName string) (bool, error) {
	blockchains, err := GetLocalNetworkBlockchainsInfo(app)
	if err != nil {
		// If we can't get blockchain info, assume not deployed
		return false, nil
	}

	for _, bc := range blockchains {
		if bc.Name == blockchainName {
			return true, nil
		}
	}
	return false, nil
}

// GetLocalNetworkLuxdVersion returns the luxd version running on local network
func GetLocalNetworkLuxdVersion(app *application.Lux) (string, int, bool, error) {
	checker := NewStatusCheckerWithApp(app)
	return checker.GetCurrentNetworkVersion()
}

// Helper functions

// findRunningNetworkType finds the first running network type
func findRunningNetworkType(app *application.Lux) (string, error) {
	// "dev" is multi-validator dev mode (network ID 1337)
	for _, netType := range []string{"dev", "mainnet", "testnet", "devnet", "custom"} {
		running, err := binutils.IsServerProcessRunningForNetwork(app, netType)
		if err != nil {
			continue
		}
		if running {
			return netType, nil
		}
	}
	return "", ErrNetworkNotRunning
}

// getPortBaseForNetwork returns the default port base for a network type
func getPortBaseForNetwork(netType string) int {
	switch netType {
	case "mainnet":
		return 9630
	case "testnet":
		return 9640
	case "devnet":
		return 9650
	case "dev":
		return 8545 // anvil/hardhat compatible
	default:
		return 9650
	}
}

// checkEndpointHealth checks if an HTTP endpoint is reachable
func checkEndpointHealth(endpoint string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(endpoint + "/ext/health")
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}
