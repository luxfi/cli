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

OVERVIEW:

  Imports historical blockchain data from RLP files into a running chain.
  This is useful for bootstrapping chains with existing state or syncing
  from canonical snapshots.

  Uses the admin_importChain RPC method. The network must be running and
  the admin API must be enabled (default when started via CLI).

CHAIN IDENTIFIERS:

  c, C         C-Chain (primary EVM chain)
  <name>       Chain name (looks up blockchain ID from sidecar)
  <blockchainID>  Direct blockchain ID

NETWORK FLAGS (auto-detects port):

  --mainnet, -m    Import to mainnet chain (port 9630)
  --testnet, -t    Import to testnet chain (port 9640)
  --devnet, -d     Import to devnet chain (port 9650)

  Default: auto-detects running network or uses custom (port 9660)

OPTIONS:

  --rpc <url>      Custom RPC endpoint (overrides network flag)

PREREQUISITES:

  1. Network must be running:
     lux network start --mainnet

  2. RLP file must exist and be readable by the node

EXAMPLES:

  # Import C-Chain mainnet blocks
  lux chain import c ~/work/lux/state/rlp/lux-mainnet-96369.rlp --mainnet

  # Import to custom chain on devnet
  lux chain import zoo ~/work/lux/state/rlp/zoo-mainnet-200200.rlp --devnet

  # Import with custom RPC endpoint
  lux chain import c blocks.rlp --rpc http://localhost:9630/ext/bc/C/rpc

  # Import to blockchain by ID
  lux chain import 2ebCneCbwthjQ1rYT41nhd7M76Hc6YmosMAQrTFhBq8qeqh6tt blocks.rlp --mainnet

RLP FILE LOCATIONS:

  Canonical RLP files are stored in:
    ~/work/lux/state/rlp/<network>/<chain>-<chainid>.rlp

  Examples:
    ~/work/lux/state/rlp/lux-mainnet/lux-mainnet-96369.rlp
    ~/work/lux/state/rlp/zoo-mainnet/zoo-mainnet-200200.rlp

IMPORT PROCESS:

  1. Validates file exists
  2. Detects or connects to RPC endpoint
  3. Gets current block height
  4. Calls admin_importChain with file path
  5. Monitors import progress
  6. Reports final block height and import rate

OUTPUT:

  Import complete!
    Blocks imported: 1082780
    Final height: 1082780
    Time: 45m12s
    Rate: 399.2 blocks/sec

TROUBLESHOOTING:

  "Network not running" → Start network first:
    lux network start --mainnet

  "RPC connection refused" → Check network is running:
    lux network status

  "File not found" → Use absolute path or verify file exists

  "Import timeout" → Import continues in background, check node logs

NOTES:

  - Import runs asynchronously - RPC may timeout but import continues
  - Large imports (1M+ blocks) can take 30min - 2hrs depending on hardware
  - The node must have read access to the RLP file
  - Genesis config must match the RLP file exactly for successful import
  - Use 'lux chain export' to create RLP files from running chains`,
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

	// Determine base URL based on network target
	// Port mapping: mainnet=9630, testnet=9640, devnet=9650, custom=9660
	baseURL := "http://localhost:9660" // Default for custom
	target := GetNetworkTarget()
	switch target {
	case NetworkMainnet:
		baseURL = "http://localhost:9630" // Mainnet ports
	case NetworkTestnet:
		baseURL = "http://localhost:9640" // Testnet ports
	case NetworkDevnet:
		baseURL = "http://localhost:9650" // Devnet ports
	case NetworkCustom:
		baseURL = "http://localhost:9660" // Custom network ports
	}

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

	// Call admin_importChain on the RPC endpoint (Coreth/geth-style API)
	// The admin API is exposed on the main RPC endpoint, not a separate admin endpoint
	success, err := callAdminImportChain(rpcEndpoint, absFilePath)
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
	// Coreth/geth-style RPC uses underscore method format: admin_importChain
	// The file path is passed as a single string parameter (not a struct)
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "admin_importChain",
		"params":  []string{filePath},
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
		errMap, _ := errObj.(map[string]interface{})
		if msg, ok := errMap["message"].(string); ok && strings.Contains(msg, "timed out") {
			// The RPC timed out but the import continues in background
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Note: RPC response timed out, but import is running in background.")
			ux.Logger.PrintToUser("The node is processing blocks asynchronously.")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Check status with:")
			ux.Logger.PrintToUser("  lux chain import-status c --mainnet")
			ux.Logger.PrintToUser("  # or check node logs")
			return true, nil // Don't error out - import is running
		}
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
