package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// StatusService handles status probing and reporting
type StatusService struct {
	concurrencyLimit int
	timeout         time.Duration
}

// NewStatusService creates a new status service
func NewStatusService() *StatusService {
	return &StatusService{
		concurrencyLimit: 32, // Global concurrency limit
		timeout:         2 * time.Second,
	}
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

// probeNode probes a single node
func (s *StatusService) probeNode(ctx context.Context, node Node) (*Node, error) {
	// TODO: Implement actual node probing
	// This should include:
	// - HTTP health check
	// - Version detection
	// - Peer count
	// - Uptime
	
	// For now, return the node as-is with placeholder values
	node.OK = true
	node.LatencyMS = 10
	node.Version = "luxd/1.22.75"
	node.PeerCount = 12
	node.Uptime = "01:22:10"
	
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
					Alias:    endpoint.ChainAlias,
					Kind:     "unknown",
					RPC_OK:   false,
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
	// TODO: Implement actual network configuration loading
	// This should load from config files, environment, etc.
	
	// For now, return a placeholder configuration
	return []Network{
		{
			Name: "mainnet",
			Nodes: []Node{
				{ID: "1", HTTPURL: "http://127.0.0.1:9630"},
				{ID: "2", HTTPURL: "http://127.0.0.1:9632"},
			},
			Metadata: NetworkMetadata{
				GRPCPort:   8369,
				NodesCount: 5,
				VMsCount:   1,
				Controller: "on",
				Status:     "up",
			},
		},
		{
			Name: "testnet",
			Nodes: []Node{
				{ID: "1", HTTPURL: "http://127.0.0.1:9631"},
				{ID: "2", HTTPURL: "http://127.0.0.1:9633"},
			},
			Metadata: NetworkMetadata{
				GRPCPort:   8368,
				NodesCount: 5,
				VMsCount:   1,
				Controller: "on",
				Status:     "up",
			},
		},
	}, nil
}

// getChainEndpoints returns the chain endpoints for a network
func (s *StatusService) getChainEndpoints(network Network) ([]EndpointStatus, error) {
	// TODO: Implement actual endpoint discovery
	// This should discover endpoints from node APIs, config, etc.
	
	// For now, return placeholder endpoints
	endpoints := []EndpointStatus{
		{ChainAlias: "p", URL: "http://127.0.0.1:9630/ext/bc/P"},
		{ChainAlias: "x", URL: "http://127.0.0.1:9630/ext/bc/X"},
		{ChainAlias: "c", URL: "http://127.0.0.1:9630/ext/bc/C/rpc"},
		{ChainAlias: "a", URL: "http://127.0.0.1:9630/ext/bc/a/rpc"},
		{ChainAlias: "b", URL: "http://127.0.0.1:9630/ext/bc/b/rpc"},
		{ChainAlias: "d", URL: "http://127.0.0.1:9630/ext/bc/d/rpc"},
		{ChainAlias: "g", URL: "http://127.0.0.1:9630/ext/bc/g/rpc"},
		{ChainAlias: "k", URL: "http://127.0.0.1:9630/ext/bc/k/rpc"},
		{ChainAlias: "q", URL: "http://127.0.0.1:9630/ext/bc/q/rpc"},
		{ChainAlias: "t", URL: "http://127.0.0.1:9630/ext/bc/t/rpc"},
		{ChainAlias: "z", URL: "http://127.0.0.1:9630/ext/bc/z/rpc"},
		{ChainAlias: "dex", URL: "http://127.0.0.1:9630/ext/bc/dex/rpc"},
	}
	
	return endpoints, nil
}