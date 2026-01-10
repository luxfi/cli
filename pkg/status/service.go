package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
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

			// Skip probing stopped networks - just copy them as-is
			if network.Metadata.Status == "stopped" || len(network.Nodes) == 0 {
				result.Networks[i] = network
				return nil
			}

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

	// Probe tracked L1 EVMs (Zoo, Hanzo, SPC)
	trackedEVMs := s.probeTrackedEVMs(ctx, result.Networks)
	result.TrackedEVMs = trackedEVMs

	// Calculate duration
	durationMS := int(time.Since(startTime).Milliseconds())
	result.Timestamp = time.Now()
	result.DurationMS = durationMS

	return &result, nil
}

// getL1ChainConfig returns the L1 chain configurations for Zoo, Hanzo, SPC
func (s *StatusService) getL1ChainConfig() []TrackedEVM {
	return []TrackedEVM{
		// Zoo - Decentralized AI network
		{Name: "zoo", Network: "mainnet", RPCs: []string{}, BlockchainID: "", VMID: ""},
		{Name: "zoo", Network: "testnet", RPCs: []string{}, BlockchainID: "", VMID: ""},
		// Hanzo - AI compute network
		{Name: "hanzo", Network: "mainnet", RPCs: []string{}, BlockchainID: "", VMID: ""},
		{Name: "hanzo", Network: "testnet", RPCs: []string{}, BlockchainID: "", VMID: ""},
		// SPC - Smart Payment Chain
		{Name: "spc", Network: "mainnet", RPCs: []string{}, BlockchainID: "", VMID: ""},
		{Name: "spc", Network: "testnet", RPCs: []string{}, BlockchainID: "", VMID: ""},
	}
}

// probeTrackedEVMs probes L1 chains (Zoo, Hanzo, SPC) based on network status
func (s *StatusService) probeTrackedEVMs(ctx context.Context, networks []Network) []EVMStatus {
	var results []EVMStatus

	// L1 chain IDs from CLAUDE.md
	l1Chains := map[string]map[string]uint64{
		"zoo":   {"mainnet": 200200, "testnet": 200201},
		"hanzo": {"mainnet": 36963, "testnet": 36962},
		"spc":   {"mainnet": 36911, "testnet": 36910},
	}

	// For each running network, try to discover L1 chains
	for _, network := range networks {
		if network.Metadata.Status != "up" || len(network.Nodes) == 0 {
			continue
		}

		baseURL := network.Nodes[0].HTTPURL
		networkType := network.Name // mainnet or testnet

		// Query for any L1 blockchains that might be running
		blockchains, err := s.getBlockchainsFromNode(ctx, baseURL)
		if err != nil {
			continue
		}

		// Check for each L1 chain
		for chainName, chainIDs := range l1Chains {
			expectedChainID := chainIDs[networkType]
			if expectedChainID == 0 {
				continue
			}

			// Look for this chain in the discovered blockchains
			for _, bc := range blockchains {
				bcName, _ := bc["name"].(string)
				bcID, _ := bc["id"].(string)

				// Match by name (case-insensitive) or check the chain ID via RPC
				if strings.EqualFold(bcName, chainName+"-chain") || strings.Contains(strings.ToLower(bcName), chainName) {
					// Found a potential L1 chain, probe it
					rpcURL := fmt.Sprintf("%s/ext/bc/%s/rpc", baseURL, bcID)

					evmStatus := s.probeL1Chain(ctx, chainName, networkType, rpcURL, expectedChainID)
					if evmStatus != nil {
						results = append(results, *evmStatus)
					}
					break
				}
			}
		}
	}

	return results
}

// getBlockchainsFromNode retrieves the list of blockchains from a node
func (s *StatusService) getBlockchainsFromNode(ctx context.Context, baseURL string) ([]map[string]interface{}, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	requestURL := fmt.Sprintf("%s/ext/bc/P", baseURL)
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "platform.getBlockchains",
		"params":  map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return nil, err
	}

	var blockchains []map[string]interface{}
	if result, ok := responseMap["result"].(map[string]interface{}); ok {
		if bcs, ok := result["blockchains"].([]interface{}); ok {
			for _, bc := range bcs {
				if bcMap, ok := bc.(map[string]interface{}); ok {
					blockchains = append(blockchains, bcMap)
				}
			}
		}
	}

	return blockchains, nil
}

// probeL1Chain probes a single L1 EVM chain
func (s *StatusService) probeL1Chain(ctx context.Context, name, network, rpcURL string, expectedChainID uint64) *EVMStatus {
	resolver := &EVMHeightResolver{}
	height, meta, err := resolver.Height(ctx, rpcURL)

	status := &EVMStatus{
		Name:    name,
		Network: network,
		Endpoints: []EndpointStatus{
			{ChainAlias: name, URL: rpcURL, OK: err == nil},
		},
	}

	if err != nil {
		return status
	}

	status.Height = height

	// Extract chain ID
	if chainID, ok := meta["chain_id"].(uint64); ok {
		status.ChainID = chainID
		if chainID != expectedChainID {
			status.ChainIDMismatch = true
		}
	}

	// Extract client version
	if version, ok := meta["client_version"].(string); ok {
		status.ClientVersion = version
	}

	// Extract syncing status
	if syncing, ok := meta["syncing"]; ok {
		status.Syncing = syncing
	}

	return status
}

// probeNetwork probes a single network
func (s *StatusService) probeNetwork(ctx context.Context, network Network) (*Network, error) {
	// Create context with timeout for this network
	networkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Probe nodes concurrently - use a separate context for errgroup to avoid cancellation issues
	nodeErrGroup, nodeCtx := errgroup.WithContext(networkCtx)

	var mu sync.Mutex
	probedNodes := make([]Node, len(network.Nodes))

	for i, node := range network.Nodes {
		i := i
		node := node

		nodeErrGroup.Go(func() error {
			probedNode, err := s.probeNode(nodeCtx, node)
			if err != nil {
				return err
			}

			mu.Lock()
			probedNodes[i] = *probedNode
			mu.Unlock()
			return nil
		})
	}

	if err := nodeErrGroup.Wait(); err != nil {
		return nil, fmt.Errorf("failed to probe nodes: %w", err)
	}

	// Update network with probed nodes
	network.Nodes = probedNodes

	// Probe chains - use the main networkCtx, not the cancelled nodeCtx
	probedChains, err := s.probeChains(networkCtx, network)
	if err != nil {
		return nil, fmt.Errorf("failed to probe chains: %w", err)
	}
	network.Chains = probedChains

	// Query balances for validators if we have any
	if len(network.Validators) > 0 && len(network.Nodes) > 0 {
		baseURL := network.Nodes[0].HTTPURL
		network.Validators = s.queryValidatorBalances(networkCtx, baseURL, network.Validators)
	}

	return &network, nil
}

// queryValidatorBalances queries P/X/C balances for all validators
func (s *StatusService) queryValidatorBalances(ctx context.Context, baseURL string, validators []ValidatorAccount) []ValidatorAccount {
	// Query balances concurrently for all validators
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range validators {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			v := &validators[idx]

			// Query P-chain balance
			if v.PChainAddress != "" {
				if balance, err := s.QueryPChainBalance(ctx, baseURL, v.PChainAddress); err == nil {
					mu.Lock()
					validators[idx].PChainBalance = balance
					mu.Unlock()
				}
			}

			// Query X-chain balance
			if v.XChainAddress != "" {
				if balance, err := s.QueryXChainBalance(ctx, baseURL, v.XChainAddress); err == nil {
					mu.Lock()
					validators[idx].XChainBalance = balance
					mu.Unlock()
				}
			}

			// Query C-chain balance
			if v.CChainAddress != "" {
				if balance, err := s.QueryCChainBalance(ctx, baseURL, v.CChainAddress); err == nil {
					mu.Lock()
					validators[idx].CChainBalance = balance
					validators[idx].CChainBalanceLUX = FormatCChainBalanceLUX(balance)
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
	return validators
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

	// 4. Get Uptime
	uptimeBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "info.uptime",
		"params":  map[string]interface{}{},
	}
	uptimeJson, _ := json.Marshal(uptimeBody)
	reqUptime, _ := http.NewRequestWithContext(ctx, "POST", versionURL, bytes.NewBuffer(uptimeJson))
	reqUptime.Header.Set("Content-Type", "application/json")
	if respUptime, err := client.Do(reqUptime); err == nil {
		defer respUptime.Body.Close()
		var r map[string]interface{}
		if err := json.NewDecoder(respUptime.Body).Decode(&r); err == nil {
			if result, ok := r["result"].(map[string]interface{}); ok {
				if uptime, ok := result["rewardingStakePercentage"].(float64); ok {
					node.Uptime = fmt.Sprintf("%.1f%%", uptime*100)
				}
			}
		}
	}

	// 5. Check GPU acceleration (via health check or custom endpoint)
	healthURL := fmt.Sprintf("%s/ext/health", node.HTTPURL)
	healthReq, _ := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if healthResp, err := client.Do(healthReq); err == nil {
		defer healthResp.Body.Close()
		var r map[string]interface{}
		if err := json.NewDecoder(healthResp.Body).Decode(&r); err == nil {
			// Check for GPU-related info in health response
			if checks, ok := r["checks"].(map[string]interface{}); ok {
				if gpuCheck, ok := checks["gpu"].(map[string]interface{}); ok {
					if msg, ok := gpuCheck["message"].(map[string]interface{}); ok {
						if device, ok := msg["device"].(string); ok {
							node.GPUDevice = device
							node.GPUAccelerated = true
						}
						if driver, ok := msg["driver"].(string); ok {
							node.GPUDriverVersion = driver
						}
					}
				}
			}
		}
	}

	// 6. Get validator addresses if this node is a validator
	if node.NodeID != "" {
		// Query platform.getCurrentValidators to get validator address
		validatorsBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "platform.getCurrentValidators",
			"params": map[string]interface{}{
				"nodeIDs": []string{node.NodeID},
			},
		}
		validatorsJson, _ := json.Marshal(validatorsBody)
		pChainURL := fmt.Sprintf("%s/ext/bc/P", node.HTTPURL)
		reqValidators, _ := http.NewRequestWithContext(ctx, "POST", pChainURL, bytes.NewBuffer(validatorsJson))
		reqValidators.Header.Set("Content-Type", "application/json")
		if respValidators, err := client.Do(reqValidators); err == nil {
			defer respValidators.Body.Close()
			var r map[string]interface{}
			if err := json.NewDecoder(respValidators.Body).Decode(&r); err == nil {
				if result, ok := r["result"].(map[string]interface{}); ok {
					if validators, ok := result["validators"].([]interface{}); ok && len(validators) > 0 {
						if validator, ok := validators[0].(map[string]interface{}); ok {
							// Get validationRewardOwner address (P-chain address)
							if rewardOwner, ok := validator["validationRewardOwner"].(map[string]interface{}); ok {
								if addrs, ok := rewardOwner["addresses"].([]interface{}); ok && len(addrs) > 0 {
									if addr, ok := addrs[0].(string); ok {
										// Address format: "11111111111111111111111111111111P-lux1..." or "...P-test1..."
										// Extract just the P-... part
										if idx := strings.Index(addr, "P-lux"); idx >= 0 {
											node.PChainAddress = addr[idx:]
											node.XChainAddress = "X-lux" + strings.TrimPrefix(addr[idx:], "P-lux")
										} else if idx := strings.Index(addr, "P-test"); idx >= 0 {
											node.PChainAddress = addr[idx:]
											node.XChainAddress = "X-test" + strings.TrimPrefix(addr[idx:], "P-test")
										} else {
											node.PChainAddress = addr
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// 7. Get C-chain address (derive from nodeID or check if node exposes it)
		// C-chain addresses are Ethereum-style (0x...) and derived differently
		// For now, try to get it from the node's keystore if available
		cChainBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "eth_accounts",
			"params":  []interface{}{},
		}
		cChainJson, _ := json.Marshal(cChainBody)
		cChainURL := fmt.Sprintf("%s/ext/bc/C/rpc", node.HTTPURL)
		reqCChain, _ := http.NewRequestWithContext(ctx, "POST", cChainURL, bytes.NewBuffer(cChainJson))
		reqCChain.Header.Set("Content-Type", "application/json")
		if respCChain, err := client.Do(reqCChain); err == nil {
			defer respCChain.Body.Close()
			var r map[string]interface{}
			if err := json.NewDecoder(respCChain.Body).Decode(&r); err == nil {
				if accounts, ok := r["result"].([]interface{}); ok && len(accounts) > 0 {
					if addr, ok := accounts[0].(string); ok {
						node.CChainAddress = addr
					}
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

		type ValidatorInfo struct {
			Index         int    `json:"index"`
			NodeID        string `json:"nodeID"`
			PChainAddress string `json:"pChainAddress"`
			XChainAddress string `json:"xChainAddress"`
			CChainAddress string `json:"cChainAddress"`
		}
		type ActiveAccountInfo struct {
			Index         int    `json:"index"`
			PChainAddress string `json:"pChainAddress"`
			XChainAddress string `json:"xChainAddress"`
			CChainAddress string `json:"cChainAddress"`
		}
		type NetworkState struct {
			NetworkType   string             `json:"network_type"`
			PortBase      int                `json:"port_base"`
			GRPCPort      int                `json:"grpc_port"`
			Running       bool               `json:"running"`
			ApiEndpoint   string             `json:"api_endpoint"`
			Validators    []ValidatorInfo    `json:"validators"`
			ActiveAccount *ActiveAccountInfo `json:"active_account"`
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

		// Convert validators from state to status model
		var validators []ValidatorAccount
		for _, v := range state.Validators {
			validators = append(validators, ValidatorAccount{
				Index:         v.Index,
				NodeID:        v.NodeID,
				PChainAddress: v.PChainAddress,
				XChainAddress: v.XChainAddress,
				CChainAddress: v.CChainAddress,
			})
		}

		// Convert active account
		var activeAccount *ActiveAccount
		if state.ActiveAccount != nil {
			activeAccount = &ActiveAccount{
				Index:         state.ActiveAccount.Index,
				PChainAddress: state.ActiveAccount.PChainAddress,
				XChainAddress: state.ActiveAccount.XChainAddress,
				CChainAddress: state.ActiveAccount.CChainAddress,
			}
		}

		networks = append(networks, Network{
			Name:          state.NetworkType,
			Nodes:         nodes,
			Validators:    validators,
			ActiveAccount: activeAccount,
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
		// Fallback to all native chains if discovery fails
		endpoints = s.getAllNativeChainEndpoints(baseURL)
	}

	return endpoints, nil
}

// getAllNativeChainEndpoints returns endpoints for all native Lux chains
// P-chain and X-chain use JSON-RPC directly (no /rpc suffix)
// EVM chains (C, Q, A, B, T, Z, G, K, D) use /rpc suffix
func (s *StatusService) getAllNativeChainEndpoints(baseURL string) []EndpointStatus {
	return []EndpointStatus{
		{ChainAlias: "p", URL: fmt.Sprintf("%s/ext/bc/P", baseURL)},     // Platform chain (JSON-RPC)
		{ChainAlias: "x", URL: fmt.Sprintf("%s/ext/bc/X", baseURL)},     // Exchange chain (JSON-RPC)
		{ChainAlias: "c", URL: fmt.Sprintf("%s/ext/bc/C/rpc", baseURL)}, // Coreth (EVM)
		{ChainAlias: "q", URL: fmt.Sprintf("%s/ext/bc/Q/rpc", baseURL)}, // Quantum (EVM)
		{ChainAlias: "a", URL: fmt.Sprintf("%s/ext/bc/A/rpc", baseURL)}, // AI (EVM)
		{ChainAlias: "b", URL: fmt.Sprintf("%s/ext/bc/B/rpc", baseURL)}, // Bridge (EVM)
		{ChainAlias: "t", URL: fmt.Sprintf("%s/ext/bc/T/rpc", baseURL)}, // Threshold (EVM)
		{ChainAlias: "z", URL: fmt.Sprintf("%s/ext/bc/Z/rpc", baseURL)}, // ZK (EVM)
		{ChainAlias: "g", URL: fmt.Sprintf("%s/ext/bc/G/rpc", baseURL)}, // Graph (EVM)
		{ChainAlias: "k", URL: fmt.Sprintf("%s/ext/bc/K/rpc", baseURL)}, // KMS (EVM)
		{ChainAlias: "d", URL: fmt.Sprintf("%s/ext/bc/D/rpc", baseURL)}, // DEX (EVM)
	}
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

	// Always include all native chains if not found
	foundChains := make(map[string]bool)
	for _, ep := range endpoints {
		foundChains[ep.ChainAlias] = true
	}

	// Add all native chains that weren't discovered
	nativeChains := s.getAllNativeChainEndpoints(baseURL)
	for _, nc := range nativeChains {
		if !foundChains[nc.ChainAlias] {
			endpoints = append(endpoints, nc)
		}
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

// QueryPChainBalance queries the P-chain balance for an address
func (s *StatusService) QueryPChainBalance(ctx context.Context, baseURL, address string) (uint64, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	requestURL := fmt.Sprintf("%s/ext/bc/P", baseURL)
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "platform.getBalance",
		"params": map[string]interface{}{
			"addresses": []string{address},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return 0, err
	}

	if result, ok := responseMap["result"].(map[string]interface{}); ok {
		if balanceStr, ok := result["balance"].(string); ok {
			balance, err := strconv.ParseUint(balanceStr, 10, 64)
			if err != nil {
				return 0, err
			}
			return balance, nil
		}
	}

	return 0, fmt.Errorf("failed to parse P-chain balance response")
}

// QueryXChainBalance queries the X-chain balance for an address
func (s *StatusService) QueryXChainBalance(ctx context.Context, baseURL, address string) (uint64, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	requestURL := fmt.Sprintf("%s/ext/bc/X", baseURL)
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "avm.getBalance",
		"params": map[string]interface{}{
			"address": address,
			"assetID": "LUX",
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return 0, err
	}

	if result, ok := responseMap["result"].(map[string]interface{}); ok {
		if balanceStr, ok := result["balance"].(string); ok {
			balance, err := strconv.ParseUint(balanceStr, 10, 64)
			if err != nil {
				return 0, err
			}
			return balance, nil
		}
	}

	return 0, fmt.Errorf("failed to parse X-chain balance response")
}

// QueryCChainBalance queries the C-chain balance for an address (0x format)
func (s *StatusService) QueryCChainBalance(ctx context.Context, baseURL, address string) (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	requestURL := fmt.Sprintf("%s/ext/bc/C/rpc", baseURL)
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getBalance",
		"params":  []interface{}{address, "latest"},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return "", err
	}

	if result, ok := responseMap["result"].(string); ok {
		return result, nil // Returns hex string like "0x1234..."
	}

	return "", fmt.Errorf("failed to parse C-chain balance response")
}

// FormatCChainBalanceLUX converts C-chain balance (wei hex) to human-readable LUX
func FormatCChainBalanceLUX(weiHex string) string {
	// Remove 0x prefix
	weiHex = strings.TrimPrefix(weiHex, "0x")
	if weiHex == "" || weiHex == "0" {
		return "0 LUX"
	}

	// Parse as big int
	wei := new(big.Int)
	wei.SetString(weiHex, 16)

	// 1 LUX = 10^18 wei
	divisor := new(big.Int)
	divisor.SetString("1000000000000000000", 10)

	// Calculate whole LUX and remainder
	luxWhole := new(big.Int).Div(wei, divisor)
	remainder := new(big.Int).Mod(wei, divisor)

	// Format with decimals if there's a remainder
	if remainder.Cmp(big.NewInt(0)) == 0 {
		return fmt.Sprintf("%s LUX", luxWhole.String())
	}

	// Show up to 4 decimal places
	remainderStr := fmt.Sprintf("%018s", remainder.String())
	remainderStr = strings.TrimRight(remainderStr[:4], "0")
	if remainderStr == "" {
		return fmt.Sprintf("%s LUX", luxWhole.String())
	}
	return fmt.Sprintf("%s.%s LUX", luxWhole.String(), remainderStr)
}

// FormatNLUXToLUX converts nLUX (nanoLUX) to human-readable LUX
func FormatNLUXToLUX(nLUX uint64) string {
	if nLUX == 0 {
		return "0 LUX"
	}

	// 1 LUX = 10^9 nLUX
	luxWhole := nLUX / 1_000_000_000
	remainder := nLUX % 1_000_000_000

	if remainder == 0 {
		return fmt.Sprintf("%d LUX", luxWhole)
	}

	// Show up to 4 decimal places
	remainderStr := fmt.Sprintf("%09d", remainder)
	remainderStr = strings.TrimRight(remainderStr[:4], "0")
	if remainderStr == "" {
		return fmt.Sprintf("%d LUX", luxWhole)
	}
	return fmt.Sprintf("%d.%s LUX", luxWhole, remainderStr)
}
