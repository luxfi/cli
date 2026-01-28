// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package status

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

	// Implement actual P-Chain height resolution using Lux P-Chain API
	// The P-Chain API endpoint is at /ext/bc/P (no /rpc suffix)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Use the URL as-is - P-Chain doesn't use /rpc suffix
	requestURL := url

	// Create JSON-RPC request for P-Chain height
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "platform.getHeight",
		"params":  map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return 0, meta, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, meta, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, meta, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// P-Chain endpoint might not exist, return 0 with appropriate error
		meta["error"] = "pchain_endpoint_not_found"
		return 0, meta, nil
	}

	if resp.StatusCode != http.StatusOK {
		return 0, meta, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return 0, meta, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract height from response
	if result, ok := responseMap["result"].(map[string]interface{}); ok {
		if heightStr, ok := result["height"].(string); ok {
			// Height might be in decimal or hex format
			height, err := strconv.ParseUint(heightStr, 10, 64)
			if err != nil {
				// Try hex format
				height, err = strconv.ParseUint(strings.TrimPrefix(heightStr, "0x"), 16, 64)
				if err != nil {
					return 0, meta, fmt.Errorf("failed to parse height: %w", err)
				}
			}

			meta["method"] = "platform.getHeight"
			return height, meta, nil
		}
	}

	return 0, meta, fmt.Errorf("invalid response format")
}

// XChainHeightResolver resolves heights for X-Chain
type XChainHeightResolver struct{}

func (r *XChainHeightResolver) Kind() string {
	return "xchain"
}

func (r *XChainHeightResolver) Height(ctx context.Context, url string) (uint64, map[string]any, error) {
	meta := make(map[string]any)

	// Implement actual X-Chain height resolution using Lux X-Chain API
	// The X-Chain API endpoint is at /ext/bc/X (no /rpc suffix)
	// Lux X-chain uses xvm.getHeight (not avm.getHeight)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Use the URL as-is - X-Chain doesn't use /rpc suffix
	requestURL := url

	// Create JSON-RPC request for X-Chain height
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "xvm.getHeight",
		"params":  map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return 0, meta, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, meta, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, meta, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, meta, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return 0, meta, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for error responses (e.g., "chain is not linearized" during bootstrap)
	if errObj, ok := responseMap["error"].(map[string]interface{}); ok {
		if errMsg, ok := errObj["message"].(string); ok {
			if strings.Contains(errMsg, "not linearized") {
				meta["status"] = "bootstrapping"
				meta["error"] = errMsg
				return 0, meta, nil // Return 0 height but no error - chain is bootstrapping
			}
			return 0, meta, fmt.Errorf("RPC error: %s", errMsg)
		}
	}

	// Extract height from response
	if result, ok := responseMap["result"].(map[string]interface{}); ok {
		if heightStr, ok := result["height"].(string); ok {
			// Convert height string to uint64
			height, err := strconv.ParseUint(heightStr, 10, 64)
			if err != nil {
				// Try hex format
				height, err = strconv.ParseUint(strings.TrimPrefix(heightStr, "0x"), 16, 64)
				if err != nil {
					return 0, meta, fmt.Errorf("failed to parse height: %w", err)
				}
			}

			meta["method"] = "xvm.getHeight"
			return height, meta, nil
		}
	}

	return 0, meta, fmt.Errorf("invalid response format")
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
	case "c": // Only C-Chain is EVM
		return &EVMHeightResolver{}
	case "p":
		return &PChainHeightResolver{}
	case "x":
		return &XChainHeightResolver{}
	case "a": // AI VM
		return &FallbackHeightResolver{}
	case "b": // Bridge VM
		return &FallbackHeightResolver{}
	case "d": // DEX VM
		return &FallbackHeightResolver{}
	case "g": // Graph VM
		return &FallbackHeightResolver{}
	case "k": // KMS VM
		return &FallbackHeightResolver{}
	case "q": // Quantum VM
		return &FallbackHeightResolver{}
	case "t": // Threshold VM
		return &FallbackHeightResolver{}
	case "z": // Zero-Knowledge VM
		return &FallbackHeightResolver{}
	// Removed duplicate "dex" case - "d" already covers DEX VM
	default:
		return &FallbackHeightResolver{}
	}
}
