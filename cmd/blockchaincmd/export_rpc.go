// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	exportRPCURL      string
	exportBlockchainID string
	exportStartBlock  uint64
	exportEndBlock    uint64
	exportOutputFile  string
)

// lux blockchain export <blockchain-id>
func newExportRPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-rpc [blockchain-id]",
		Short: "Export blockchain data via RPC",
		Long: `Export blockchain blocks via RPC from a running node.
Uses the migrate_getBlocks RPC endpoint to fetch blocks in batches.

Example:
  lux blockchain export-rpc dnmzhuf6poM6... --output=blocks.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: exportRPCFunc,
	}

	cmd.Flags().StringVar(&exportRPCURL, "rpc-url", "", "RPC endpoint (auto-discovered if not specified)")
	cmd.Flags().Uint64Var(&exportStartBlock, "start-block", 0, "Start block number")
	cmd.Flags().Uint64Var(&exportEndBlock, "end-block", 0, "End block number (0 = current)")
	cmd.Flags().StringVar(&exportOutputFile, "output", "blocks.json", "Output file path")

	return cmd
}

func exportRPCFunc(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Get blockchain ID from arg or flag
	var blockchainID string
	if len(args) > 0 {
		blockchainID = args[0]
	} else if exportBlockchainID != "" {
		blockchainID = exportBlockchainID
	} else {
		return fmt.Errorf("blockchain ID required")
	}

	// Discover RPC endpoint if not provided
	if exportRPCURL == "" {
		exportRPCURL = discoverBlockchainRPC(blockchainID)
		ux.Logger.PrintToUser("🔍 Using RPC: %s", exportRPCURL)
	}

	// Get current block if end not specified
	if exportEndBlock == 0 {
		currentBlock, err := getCurrentBlockRPC(ctx, exportRPCURL)
		if err != nil {
			return fmt.Errorf("failed to get current block: %w", err)
		}
		exportEndBlock = currentBlock
	}

	ux.Logger.PrintToUser("📤 Exporting blocks %d to %d...", exportStartBlock, exportEndBlock)

	// Export in batches of 100
	allBlocks := []interface{}{}
	batchSize := uint64(100)

	for start := exportStartBlock; start <= exportEndBlock; start += batchSize {
		end := start + batchSize - 1
		if end > exportEndBlock {
			end = exportEndBlock
		}

		ux.Logger.PrintToUser("  Fetching blocks %d-%d...", start, end)

		blocks, err := getBlocksRPC(ctx, exportRPCURL, start, end)
		if err != nil {
			return fmt.Errorf("failed to get blocks %d-%d: %w", start, end, err)
		}

		allBlocks = append(allBlocks, blocks...)
	}

	// Write to file
	data, err := json.MarshalIndent(map[string]interface{}{
		"blockchainID": blockchainID,
		"startBlock":   exportStartBlock,
		"endBlock":     exportEndBlock,
		"blockCount":   len(allBlocks),
		"exportTime":   time.Now().Unix(),
		"blocks":       allBlocks,
	}, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(exportOutputFile, data, 0644); err != nil {
		return err
	}

	ux.Logger.PrintToUser("✅ Exported %d blocks to %s", len(allBlocks), exportOutputFile)
	return nil
}

// getBlocksRPC calls migrate_getBlocks RPC endpoint
func getBlocksRPC(ctx context.Context, rpcURL string, start, end uint64) ([]interface{}, error) {
	req := &RPCRequest{
		JSONRPC: "2.0",
		Method:  "migrate_getBlocks",
		Params:  []interface{}{start, end, 100},
		ID:      1,
	}

	var blocks []interface{}
	if err := callRPCGeneric(ctx, rpcURL, req, &blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

// getCurrentBlockRPC gets current block number
func getCurrentBlockRPC(ctx context.Context, rpcURL string) (uint64, error) {
	req := &RPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	var result string
	if err := callRPCGeneric(ctx, rpcURL, req, &result); err != nil {
		return 0, err
	}

	var blockNum uint64
	_, err := fmt.Sscanf(result, "0x%x", &blockNum)
	return blockNum, err
}

// discoverBlockchainRPC discovers RPC endpoint for a blockchain
func discoverBlockchainRPC(blockchainID string) string {
	// Internal RPC port 9630
	// Use blockchain ID in path for old nets, or C for C-Chain
	if blockchainID == "C" {
		return "http://127.0.0.1:9630/ext/bc/C/rpc"
	}
	return fmt.Sprintf("http://127.0.0.1:9630/ext/bc/%s/rpc", blockchainID)
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
	ID      int             `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// callRPCGeneric makes a generic JSON-RPC call
func callRPCGeneric(ctx context.Context, rpcURL string, req *RPCRequest, result interface{}) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return json.Unmarshal(rpcResp.Result, result)
}
