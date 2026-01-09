package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/constants"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var (
	// ErrNoNetwork indicates no nodes are running for a network status check.
	ErrNoNetwork = errors.New("no network running")
)

// StatusService handles status probing and reporting
type StatusService struct {
	concurrencyLimit int
	timeout          time.Duration
}

// NewStatusService creates a new status service
func NewStatusService() *StatusService {
	return &StatusService{
		concurrencyLimit: 32, // Global concurrency limit
		timeout:          2 * time.Second,
	}
}

// NewStatusServiceWithProgress creates a new status service with a progress bar (if needed)
func NewStatusServiceWithProgress(progress interface{}) *StatusService {
	// For now, we ignore the progress interface as the service doesn't use it directly yet
	return NewStatusService()
}

// GetStatus retrieves the status of all networks and chains
func (s *StatusService) GetStatus(ctx context.Context) (*StatusResult, error) {
	startTime := time.Now()

	// Get network configurations
	networks, err := s.getNetworkConfigurations()
	if err != nil {
		return nil, fmt.Errorf("failed to get network configurations: %w", err)
	}

	// Create semaphore for concurrency control
	sem := semaphore.NewWeighted(int64(s.concurrencyLimit))

	// Process each network concurrently
	errGroup, ctx := errgroup.WithContext(ctx)

	var result StatusResult
	result.Networks = make([]Network, len(networks))

	for i, network := range networks {
		i := i
		network := network

		errGroup.Go(func() error {
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			// Probe this network
			probedNetwork, err := s.probeNetwork(ctx, network)
			if err != nil {
				return err
			}

			result.Networks[i] = *probedNetwork
			return nil
		})
	}

	// Wait for all networks to be probed
	if err := errGroup.Wait(); err != nil {
		return nil, fmt.Errorf("failed to probe networks: %w", err)
	}

	// Calculate duration
	durationMS := int(time.Since(startTime).Milliseconds())
	result.Timestamp = time.Now()
	result.DurationMS = durationMS

	return &result, nil
}

// probeNetwork probes a single network
func (s *StatusService) probeNetwork(ctx context.Context, network Network) (*Network, error) {
	// Create context with timeout for this network
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Probe nodes concurrently
	errGroup, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	probedNodes := make([]Node, len(network.Nodes))

	for i, node := range network.Nodes {
		i := i
		node := node

		errGroup.Go(func() error {
			probedNode, err := s.probeNode(ctx, node)
			if err != nil {
				return err
			}

			mu.Lock()
			probedNodes[i] = *probedNode
			mu.Unlock()
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		return nil, fmt.Errorf("failed to probe nodes: %w", err)
	}

	// Update network with probed nodes
	network.Nodes = probedNodes

	// Probe chains
	probedChains, err := s.probeChains(ctx, network)
	if err != nil {
		return nil, fmt.Errorf("failed to probe chains: %w", err)
	}
	network.Chains = probedChains

	return &network, nil
}

// probeNode probes a single node by making real API calls
func (s *StatusService) probeNode(ctx context.Context, node Node) (*Node, error) {
	startTime := time.Now()

	// 1. Get Node Version
	versionURL := fmt.Sprintf("%s/ext/info", node.HTTPURL)
	versionBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "info.getNodeVersion",
		"params":  map[string]interface{}{},
	}
	vJson, _ := json.Marshal(versionBody)

	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "POST", versionURL, bytes.NewBuffer(vJson))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		var r map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err == nil {
			if result, ok := r["result"].(map[string]interface{}); ok {
				if version, ok := result["version"].(string); ok {
					node.Version = version
				}
			}
		}
	} else {
		return &node, fmt.Errorf("node unreachable: %w", err)
	}

	// 2. Get NodeID
	idBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "info.getNodeID",
		"params":  map[string]interface{}{},
	}
	idJson, _ := json.Marshal(idBody)
	reqID, _ := http.NewRequestWithContext(ctx, "POST", versionURL, bytes.NewBuffer(idJson))
	reqID.Header.Set("Content-Type", "application/json")
	if respID, err := client.Do(reqID); err == nil {
		defer respID.Body.Close()
		var r map[string]interface{}
		if err := json.NewDecoder(respID.Body).Decode(&r); err == nil {
			if result, ok := r["result"].(map[string]interface{}); ok {
				if nodeID, ok := result["nodeID"].(string); ok {
					node.NodeID = nodeID
				}
			}
		}
	}

	// 3. Get Peers
	peersBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "info.peers",
		"params":  map[string]interface{}{},
	}
	pJson, _ := json.Marshal(peersBody)
	reqP, _ := http.NewRequestWithContext(ctx, "POST", versionURL, bytes.NewBuffer(pJson))
	reqP.Header.Set("Content-Type", "application/json")
	if respP, err := client.Do(reqP); err == nil {
		defer respP.Body.Close()
		var r map[string]interface{}
		if err := json.NewDecoder(respP.Body).Decode(&r); err == nil {
			if result, ok := r["result"].(map[string]interface{}); ok {
				if peers, ok := result["peers"].([]interface{}); ok {
					node.PeerCount = len(peers)
				}
			}
		}
	}

	node.LatencyMS = int(time.Since(startTime).Milliseconds())
	node.OK = true

	return &node, nil
}

// probeChains probes all chains for a network
func (s *StatusService) probeChains(ctx context.Context, network Network) ([]ChainStatus, error) {
	// Get all chain endpoints for this network
	endpoints, err := s.getChainEndpoints(network)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain endpoints: %w", err)
	}

	// Create semaphore for chain probing
	sem := semaphore.NewWeighted(int64(s.concurrencyLimit))
	errGroup, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	probedChains := make([]ChainStatus, len(endpoints))

	for i, endpoint := range endpoints {
		i := i
		endpoint := endpoint

		errGroup.Go(func() error {
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			probedChain, err := s.probeChainEndpoint(ctx, endpoint)
			if err != nil {
				// Store error but don't fail the whole operation
				probedChain := ChainStatus{
					Alias:     endpoint.ChainAlias,
					Kind:      "unknown",
					RPC_OK:    false,
					LastError: fmt.Sprintf("failed to probe: %v", err),
				}
				mu.Lock()
				probedChains[i] = probedChain
				mu.Unlock()
				return nil
			}

			mu.Lock()
			probedChains[i] = *probedChain
			mu.Unlock()
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		return nil, fmt.Errorf("failed to probe chain endpoints: %w", err)
	}

	return probedChains, nil
}

// probeChainEndpoint probes a single chain endpoint
func (s *StatusService) probeChainEndpoint(ctx context.Context, endpoint EndpointStatus) (*ChainStatus, error) {
	startTime := time.Now()

	// Get resolver for this chain
	resolver := GetResolverForChain(endpoint.ChainAlias)

	// Probe the endpoint
	height, meta, err := resolver.Height(ctx, endpoint.URL)

	latencyMS := int(time.Since(startTime).Milliseconds())

	if err != nil {
		return &ChainStatus{
			Alias:     endpoint.ChainAlias,
			Kind:      resolver.Kind(),
			RPC_OK:    false,
			LatencyMS: latencyMS,
			LastError: err.Error(),
			Metadata:  meta,
		}, nil
	}

	// Extract metadata
	chainStatus := ChainStatus{
		Alias:     endpoint.ChainAlias,
		Kind:      resolver.Kind(),
		Height:    height,
		RPC_OK:    true,
		LatencyMS: latencyMS,
		Metadata:  meta,
	}

	// Extract chain ID if available
	if chainID, ok := meta["chain_id"].(uint64); ok {
		chainStatus.ChainID = fmt.Sprintf("%d", chainID)
	}

	// Extract syncing status if available
	if syncing, ok := meta["syncing"]; ok {
		chainStatus.Syncing = syncing
	}

	return &chainStatus, nil
}

// getNetworkConfigurations returns the network configurations
func (s *StatusService) getNetworkConfigurations() ([]Network, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	luxDir := filepath.Join(home, ".lux")

	// Define all known network types that should be tracked
	knownNetworks := []string{"mainnet", "testnet", "devnet", "custom"}

	// Find all network state files
	matches, err := filepath.Glob(filepath.Join(luxDir, "*_network_state.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob network state files: %w", err)
	}

	var networks []Network
	foundNetworks := make(map[string]bool)

	// First, process any existing network state files
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}

		type NetworkState struct {
			NetworkType string `json:"network_type"`
			PortBase    int    `json:"port_base"`
			GRPCPort    int    `json:"grpc_port"`
			Running     bool   `json:"running"`
			ApiEndpoint string `json:"api_endpoint"`
		}

		var state NetworkState
		if err := json.Unmarshal(data, &state); err != nil {
			continue // Skip invalid JSON
		}

		foundNetworks[state.NetworkType] = true

		if !state.Running {
			// Include stopped networks but mark them
			networks = append(networks, Network{
				Name: state.NetworkType,
				Metadata: NetworkMetadata{
					Status: "stopped",
				},
			})
			continue
		}

		// Discover nodes for this network by checking the runs directory first
		runDirPattern := filepath.Join(luxDir, "runs", state.NetworkType, "run_*")
		runDirs, err := filepath.Glob(runDirPattern)
		if err != nil {
			runDirs = []string{}
		}

		var nodeDirs []string
		if len(runDirs) > 0 {
			// Use the most recent run directory to discover nodes
			latestRunDir := ""
			var latestModTime time.Time

			for _, runDir := range runDirs {
				if info, err := os.Stat(runDir); err == nil {
					if info.ModTime().After(latestModTime) {
						latestModTime = info.ModTime()
						latestRunDir = runDir
					}
				}
			}

			if latestRunDir != "" {
				nodeDirs, _ = filepath.Glob(filepath.Join(latestRunDir, "node*"))
			}
		} else {
			// Fallback to the old networks directory if no runs directory exists
			networkDir := filepath.Join(luxDir, "networks", state.NetworkType)
			nodeDirs, _ = filepath.Glob(filepath.Join(networkDir, "node*"))
		}

		var nodes []Node
		for _, nodeDir := range nodeDirs {
			nodeName := filepath.Base(nodeDir)
			nodeID := strings.TrimPrefix(nodeName, "node")

			// Try to read process.json
			procPath := filepath.Join(nodeDir, "process.json")
			procData, err := os.ReadFile(procPath)

			uri := ""
			if err == nil {
				var proc struct {
					URI string `json:"uri"`
				}
				if err := json.Unmarshal(procData, &proc); err == nil {
					uri = proc.URI
				}
			}

			// Fallback if process.json is missing or invalid
			if uri == "" {
				idx, _ := strconv.Atoi(nodeID)
				if idx > 0 {
					apiPort := state.PortBase + ((idx - 1) * 2)
					uri = fmt.Sprintf("http://127.0.0.1:%d", apiPort)
				}
			}

			if uri != "" {
				nodes = append(nodes, Node{
					ID:      nodeID,
					HTTPURL: uri,
				})
			}
		}

		// Handle single-node networks (like devnet) where node directories might not exist
		if len(nodes) == 0 && state.ApiEndpoint != "" {
			nodes = append(nodes, Node{
				ID:      "1",
				HTTPURL: state.ApiEndpoint,
			})
		} else if len(nodes) == 0 && state.PortBase > 0 {
			// Fallback to PortBase if API endpoint is missing
			nodes = append(nodes, Node{
				ID:      "1",
				HTTPURL: fmt.Sprintf("http://127.0.0.1:%d", state.PortBase),
			})
		}

		// Get gRPC port from constants if not set in state
		grpcPort := state.GRPCPort
		if grpcPort == 0 {
			ports := constants.GetGRPCPorts(state.NetworkType)
			grpcPort = ports.Server
		}

		networks = append(networks, Network{
			Name:  state.NetworkType,
			Nodes: nodes,
			Metadata: NetworkMetadata{
				GRPCPort:   grpcPort,
				NodesCount: len(nodes),
				VMsCount:   1, // Placeholder until probed
				Controller: "on",
				Status: (func() string {
					if state.Running {
						return "up"
					}
					return "stopped"
				}()),
			},
		})
	}

	// Add any known networks that weren't found in state files (they might be stopped)
	for _, netType := range knownNetworks {
		if !foundNetworks[netType] {
			// Check if this network has any runtime data
			networkDir := filepath.Join(luxDir, "networks", netType)
			if _, err := os.Stat(networkDir); err == nil {
				// Network directory exists but no state file - mark as stopped
				networks = append(networks, Network{
					Name: netType,
					Metadata: NetworkMetadata{
						Status: "stopped",
					},
				})
			}
		}
	}

	if len(networks) == 0 {
		return nil, ErrNoNetwork
	}

	return networks, nil
}

// getChainEndpoints returns the chain endpoints for a network
func (s *StatusService) getChainEndpoints(network Network) ([]EndpointStatus, error) {
	if len(network.Nodes) == 0 {
		return nil, ErrNoNetwork
	}
	baseURL := network.Nodes[0].HTTPURL

	// Try to discover actual chain endpoints from the node
	endpoints, err := s.discoverChainEndpointsFromNode(baseURL)
	if err != nil {
		// Fallback to standard chains if discovery fails
		endpoints = []EndpointStatus{
			{ChainAlias: "p", URL: fmt.Sprintf("%s/ext/bc/P", baseURL)},
			{ChainAlias: "x", URL: fmt.Sprintf("%s/ext/bc/X", baseURL)},
			{ChainAlias: "c", URL: fmt.Sprintf("%s/ext/bc/C/rpc", baseURL)},
		}
	}

	return endpoints, nil
}

// discoverChainEndpointsFromNode attempts to discover all available chain endpoints
func (s *StatusService) discoverChainEndpointsFromNode(baseURL string) ([]EndpointStatus, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Build the request URL for platform.getBlockchains
	requestURL := fmt.Sprintf("%s/ext/bc/P/rpc", baseURL)

	// Create JSON-RPC request
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "platform.getBlockchains",
		"params":  map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract blockchains from response
	var endpoints []EndpointStatus
	if result, ok := responseMap["result"].(map[string]interface{}); ok {
		if blockchains, ok := result["blockchains"].([]interface{}); ok {
			for _, bc := range blockchains {
				if blockchain, ok := bc.(map[string]interface{}); ok {
					if id, ok := blockchain["id"].(string); ok {
						// Map blockchain ID to chain alias
						chainAlias := s.mapBlockchainIDToAlias(id)
						if chainAlias != "" {
							url := fmt.Sprintf("%s/ext/bc/%s", baseURL, id)
							// Special case for C-Chain (EVM) which uses /rpc endpoint
							if chainAlias == "c" {
								url = fmt.Sprintf("%s/ext/bc/C/rpc", baseURL)
							}
							endpoints = append(endpoints, EndpointStatus{
								ChainAlias: chainAlias,
								URL:        url,
							})
						}
					}
				}
			}
		}
	}

	// Always include core chains if not found
	foundChains := make(map[string]bool)
	for _, ep := range endpoints {
		foundChains[ep.ChainAlias] = true
	}

	if !foundChains["p"] {
		endpoints = append(endpoints, EndpointStatus{
			ChainAlias: "p",
			URL:        fmt.Sprintf("%s/ext/bc/P", baseURL),
		})
	}
	if !foundChains["x"] {
		endpoints = append(endpoints, EndpointStatus{
			ChainAlias: "x",
			URL:        fmt.Sprintf("%s/ext/bc/X", baseURL),
		})
	}
	if !foundChains["c"] {
		endpoints = append(endpoints, EndpointStatus{
			ChainAlias: "c",
			URL:        fmt.Sprintf("%s/ext/bc/C/rpc", baseURL),
		})
	}

	return endpoints, nil
}

// mapBlockchainIDToAlias maps blockchain IDs to chain aliases
func (s *StatusService) mapBlockchainIDToAlias(blockchainID string) string {
	// Standard Lux blockchain IDs
	switch blockchainID {
	case "P":
		return "p"
	case "X":
		return "x"
	case "C":
		return "c"
	case "2ebCneCbwthjQ1rYT41nhd7M76Hc6YmosMAQrTFhBq8SvgU1s": // Mainnet C-Chain
		return "c"
	case "2oYMBNV4eNHyqk2fjjV5nP2rB8kJLnN57D7D77D7D7D7D7D7D": // Fuji C-Chain
		return "c"
	case "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPrRJU1J1U1J1U1J1J": // Devnet C-Chain
		return "c"
	default:
		// Custom chains - use first few characters as alias
		if len(blockchainID) >= 4 {
			return blockchainID[:4]
		}
		return blockchainID
	}
}
