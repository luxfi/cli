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
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

var (
	importRPC string
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <chain> <path>",
		Short: "Import blocks from RLP file to a running chain",
		Long: `Import blocks from an RLP-encoded file to a running chain.

This command uses the admin_importChain RPC endpoint to import blocks.
The network must be running and admin API must be enabled (default).

Chain names:
  c, C     - C-Chain (Coreth EVM)
  zoo      - Zoo subnet (will look up blockchain ID)
  <id>     - Any blockchain ID directly

Examples:
  # Import to C-Chain
  lux chain import c /path/to/blocks.rlp

  # Import to Zoo subnet
  lux chain import zoo ~/work/lux/state/rlp/zoo-mainnet/zoo-mainnet-200200.rlp

  # Import to subnet by blockchain ID
  lux chain import UNFEYEGJz3m1u5bYQw9BCgk6nqTTLqAL7a4Qi59VcD5tV5CCp /path/to/blocks.rlp

  # Import with custom RPC endpoint
  lux chain import c /path/to/blocks.rlp --rpc=http://localhost:9630/ext/bc/C/rpc

Requirements:
  - Network must be running (lux network start)
  - Admin API must be enabled (default when started via CLI)
  - RLP file path must be accessible from the node filesystem`,
		Args: cobra.ExactArgs(2),
		RunE: runChainImport,
	}

	cmd.Flags().StringVar(&importRPC, "rpc", "", "Custom RPC endpoint (default: auto-detected)")

	return cmd
}

func runChainImport(_ *cobra.Command, args []string) error {
	chainArg := args[0]
	filePath := args[1]

	// Validate file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("RLP file not found: %s", filePath)
	}

	// Get absolute path for the file
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Resolve chain name to blockchain ID/path
	chainPath, chainDisplay := resolveChain(chainArg)

	// Determine endpoints - eth RPC and admin are different
	baseURL := "http://localhost:9630"
	if importRPC != "" {
		baseURL = strings.TrimSuffix(importRPC, "/ext/bc/"+chainPath+"/rpc")
		baseURL = strings.TrimSuffix(baseURL, "/ext/bc/"+chainPath+"/admin")
		baseURL = strings.TrimSuffix(baseURL, "/")
	}
	rpcEndpoint := fmt.Sprintf("%s/ext/bc/%s/rpc", baseURL, chainPath)
	adminEndpoint := fmt.Sprintf("%s/ext/bc/%s/admin", baseURL, chainPath)

	ux.Logger.PrintToUser("Importing blocks to %s...", chainDisplay)
	ux.Logger.PrintToUser("  RLP file: %s", absFilePath)
	ux.Logger.PrintToUser("  RPC endpoint: %s", rpcEndpoint)
	ux.Logger.PrintToUser("  Admin endpoint: %s", adminEndpoint)

	// Get current block height before import
	beforeHeight, err := getBlockHeight(rpcEndpoint)
	if err != nil {
		ux.Logger.PrintToUser("  Warning: Could not get current block height: %v", err)
	} else {
		ux.Logger.PrintToUser("  Current block height: %d", beforeHeight)
	}

	startTime := time.Now()

	// Call admin_importChain on the admin endpoint, not rpc
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

// resolveChain resolves a chain name to blockchain path and display name
func resolveChain(chain string) (path, display string) {
	lower := strings.ToLower(chain)

	// C-Chain
	if lower == "c" {
		return "C", "C-Chain"
	}

	// Known chain names - try to look up blockchain ID
	if lower == "zoo" {
		if blockchainID := lookupBlockchainID("zoo"); blockchainID != "" {
			return blockchainID, "Zoo Chain"
		}
		return chain, "Zoo Chain (not deployed)"
	}

	// Assume it's a blockchain ID
	return chain, fmt.Sprintf("Chain %s", chain[:min(len(chain), 12)])
}

// lookupBlockchainID looks up a chain's blockchain ID from sidecar
func lookupBlockchainID(chainName string) string {
	// Try to load sidecar for the chain
	sc, err := app.LoadSidecar(chainName)
	if err != nil {
		return ""
	}

	// Check Local Network deployment
	if network, ok := sc.Networks[models.Local.String()]; ok {
		return network.BlockchainID.String()
	}

	return ""
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

func callAdminImportChain(rpcEndpoint, filePath string) (bool, error) {
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

	resp, err := client.Post(rpcEndpoint, "application/json", bytes.NewBuffer(data))
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
