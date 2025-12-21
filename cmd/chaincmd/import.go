// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	importChain    string
	importFilePath string
	importRPC      string
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import blocks from RLP file to a running chain",
		Long: `Import blocks from an RLP-encoded file to a running chain.

This command uses the admin_importChain RPC endpoint to import blocks.
The admin API endpoint is on port 9630 (NOT 9650) at /ext/bc/<chain>/admin.

Supported chains:
  - C-Chain (coreth): Use --chain=c or --chain=C
  - Subnet EVMs: Use --chain=<blockchain-id> (e.g., ZOO subnet)

Examples:
  # Import to C-Chain on running local network
  lux chain import --chain=c --path=/tmp/lux-mainnet-96369.rlp

  # Import with custom RPC endpoint
  lux chain import --path=/tmp/blocks.rlp --rpc=http://localhost:9630/ext/bc/C/admin

  # Import to Zoo subnet EVM
  lux chain import --chain=bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM --path=/tmp/zoo.rlp

Requirements:
  - Network must be running (lux network start)
  - Admin API must be enabled (default when started via CLI)
  - RLP file path must be accessible from the node filesystem`,
		RunE: runChainImport,
	}

	cmd.Flags().StringVar(&importChain, "chain", "c", "Chain to import to (c for C-Chain, or blockchain ID for subnets)")
	cmd.Flags().StringVar(&importFilePath, "path", "", "Path to RLP file (required)")
	cmd.Flags().StringVar(&importRPC, "rpc", "", "Admin RPC endpoint (default: http://localhost:9630/ext/bc/C/admin)")

	cmd.MarkFlagRequired("path")

	return cmd
}

func runChainImport(_ *cobra.Command, _ []string) error {
	// Validate file exists
	if _, err := os.Stat(importFilePath); os.IsNotExist(err) {
		return fmt.Errorf("RLP file not found: %s", importFilePath)
	}

	// Get absolute path for the file
	absFilePath, err := filepath.Abs(importFilePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Determine admin RPC endpoint
	adminEndpoint := importRPC
	if adminEndpoint == "" {
		// Build default endpoint based on chain
		chainPath := getChainPath(importChain)
		adminEndpoint = fmt.Sprintf("http://localhost:9630/ext/bc/%s/admin", chainPath)
	}

	// Get chain display name
	chainDisplay := importChain
	if strings.ToLower(importChain) == "c" {
		chainDisplay = "C-Chain"
	}

	ux.Logger.PrintToUser("Importing blocks to %s...", chainDisplay)
	ux.Logger.PrintToUser("  RLP file: %s", absFilePath)
	ux.Logger.PrintToUser("  Admin endpoint: %s", adminEndpoint)

	// Get current block height before import (use RPC endpoint for eth calls)
	rpcEndpoint := strings.Replace(adminEndpoint, "/admin", "/rpc", 1)
	beforeHeight, err := getBlockHeight(rpcEndpoint)
	if err != nil {
		ux.Logger.PrintToUser("  Warning: Could not get current block height: %v", err)
	} else {
		ux.Logger.PrintToUser("  Current block height: %d", beforeHeight)
	}

	startTime := time.Now()

	// Call admin_importChain RPC
	success, err := callAdminImportChain(adminEndpoint, absFilePath)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	elapsed := time.Since(startTime)

	if !success {
		return fmt.Errorf("import returned false (check node logs for details)")
	}

	// Get block height after import
	afterHeight, err := getBlockHeight(rpcEndpoint)
	if err != nil {
		ux.Logger.PrintToUser("  Warning: Could not get final block height: %v", err)
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Import complete!")
		ux.Logger.PrintToUser("  Time: %v", elapsed.Round(time.Second))
	} else {
		blocksImported := afterHeight - beforeHeight
		rate := float64(blocksImported) / elapsed.Seconds()
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Import complete!")
		ux.Logger.PrintToUser("  Blocks imported: %d", blocksImported)
		ux.Logger.PrintToUser("  Final height: %d", afterHeight)
		ux.Logger.PrintToUser("  Time: %v", elapsed.Round(time.Second))
		if rate > 0 {
			ux.Logger.PrintToUser("  Rate: %.1f blocks/sec", rate)
		}
	}

	return nil
}

// getChainPath returns the chain path for use in URLs
func getChainPath(chain string) string {
	if strings.ToLower(chain) == "c" {
		return "C"
	}
	return chain
}

func getBlockHeight(rpcEndpoint string) (uint64, error) {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(rpcEndpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if errObj, ok := result["error"]; ok {
		return 0, fmt.Errorf("RPC error: %v", errObj)
	}

	heightHex, ok := result["result"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid result format")
	}

	var height uint64
	fmt.Sscanf(heightHex, "0x%x", &height)
	return height, nil
}

func callAdminImportChain(adminEndpoint, filePath string) (bool, error) {
	// admin_importChain takes a single string parameter: the file path
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "admin_importChain",
		"params":  []interface{}{filePath},
		"id":      1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use long timeout for large imports
	client := &http.Client{
		Timeout: 24 * time.Hour,
	}

	ux.Logger.PrintToUser("Calling admin_importChain (this may take a while for large files)...")

	resp, err := client.Post(adminEndpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body))
	}

	if errObj, ok := result["error"]; ok {
		return false, fmt.Errorf("RPC error: %v", errObj)
	}

	// Result can be bool or empty on success
	if resultVal, ok := result["result"]; ok {
		if success, ok := resultVal.(bool); ok {
			return success, nil
		}
		// Some implementations return empty result on success
		return true, nil
	}

	return true, nil
}
