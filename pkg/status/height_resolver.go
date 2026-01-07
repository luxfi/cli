package status

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/utils"
)

// HeightResolver defines the interface for resolving chain heights
type HeightResolver interface {
	Height(ctx context.Context, url string) (height uint64, meta map[string]any, err error)
	Kind() string // "evm", "pchain", "xchain", "custom", etc.
}

// EVMHeightResolver resolves heights for EVM-compatible chains
type EVMHeightResolver struct{}

func (r *EVMHeightResolver) Kind() string {
	return "evm"
}

func (r *EVMHeightResolver) Height(ctx context.Context, url string) (uint64, map[string]any, error) {
	meta := make(map[string]any)
	
	// Create a client with timeout
	client, err := utils.NewEVMClientWithTimeout(url, 2*time.Second)
	if err != nil {
		return 0, meta, fmt.Errorf("failed to create EVM client: %w", err)
	}
	
	// Get block number
	height, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, meta, fmt.Errorf("failed to get block number: %w", err)
	}
	
	// Get chain ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		meta["chain_id_error"] = err.Error()
	} else {
		meta["chain_id"] = chainID.Uint64()
	}
	
	// Get syncing status
	syncing, err := client.Syncing(ctx)
	if err != nil {
		meta["syncing_error"] = err.Error()
	} else {
		meta["syncing"] = syncing
	}
	
	// Get client version
	version, err := client.ClientVersion(ctx)
	if err != nil {
		meta["client_version_error"] = err.Error()
	} else {
		meta["client_version"] = version
	}
	
	return height, meta, nil
}

// PChainHeightResolver resolves heights for P-Chain
type PChainHeightResolver struct{}

func (r *PChainHeightResolver) Kind() string {
	return "pchain"
}

func (r *PChainHeightResolver) Height(ctx context.Context, url string) (uint64, map[string]any, error) {
	meta := make(map[string]any)
	
	// TODO: Implement actual P-Chain height resolution
	// This is a placeholder - replace with actual API calls
	meta["method"] = "pchain.getHeight (placeholder)"
	
	// For now, return a placeholder value
	return 0, meta, fmt.Errorf("pchain height resolver not yet implemented")
}

// XChainHeightResolver resolves heights for X-Chain
type XChainHeightResolver struct{}

func (r *XChainHeightResolver) Kind() string {
	return "xchain"
}

func (r *XChainHeightResolver) Height(ctx context.Context, url string) (uint64, map[string]any, error) {
	meta := make(map[string]any)
	
	// TODO: Implement actual X-Chain height resolution
	// This is a placeholder - replace with actual API calls
	meta["method"] = "xchain.getHeight (placeholder)"
	
	// For now, return a placeholder value
	return 0, meta, fmt.Errorf("xchain height resolver not yet implemented")
}

// FallbackHeightResolver tries EVM first, then falls back to unknown
type FallbackHeightResolver struct{}

func (r *FallbackHeightResolver) Kind() string {
	return "fallback"
}

func (r *FallbackHeightResolver) Height(ctx context.Context, url string) (uint64, map[string]any, error) {
	meta := make(map[string]any)
	
	// First try EVM
	evmResolver := &EVMHeightResolver{}
	height, evmMeta, err := evmResolver.Height(ctx, url)
	if err == nil {
		// Merge metadata
		for k, v := range evmMeta {
			meta[k] = v
		}
		meta["resolver"] = "evm"
		return height, meta, nil
	}
	
	// If EVM fails, mark as unknown
	meta["resolver"] = "fallback"
	meta["error"] = err.Error()
	return 0, meta, fmt.Errorf("unknown chain type: %w", err)
}

// GetResolverForChain returns the appropriate resolver for a chain alias
func GetResolverForChain(chainAlias string) HeightResolver {
	switch chainAlias {
	case "c", "a", "b", "d", "g", "k", "q", "t", "z", "dex":
		return &EVMHeightResolver{}
	case "p":
		return &PChainHeightResolver{}
	case "x":
		return &XChainHeightResolver{}
	default:
		return &FallbackHeightResolver{}
	}
}