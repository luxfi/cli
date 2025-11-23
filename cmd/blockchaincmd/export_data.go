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
	exportDataDir     string
	exportDataID      string
	exportDataRPC     string
	exportDataStart   uint64
	exportDataEnd     uint64
	exportDataOut     string
)

// lux blockchain export --data-dir=...
func newExportDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-data",
		Short: "Export blockchain data via RPC",
		Long: `Export blockchain blocks from a running node via RPC.

Example:
  lux blockchain export-data --id=dnmzhuf6... --output=blocks.json`,
		RunE: exportDataFunc,
	}

	cmd.Flags().StringVar(&exportDataID, "id", "", "Blockchain ID")
	cmd.Flags().StringVar(&exportDataRPC, "rpc", "", "RPC endpoint (auto-discovered from ID)")
	cmd.Flags().StringVar(&exportDataDir, "data-dir", "", "Data directory (for discovery)")
	cmd.Flags().Uint64Var(&exportDataStart, "start-block", 0, "Start block")
	cmd.Flags().Uint64Var(&exportDataEnd, "end-block", 0, "End block (0=current)")
	cmd.Flags().StringVar(&exportDataOut, "output", "blocks.json", "Output file")

	return cmd
}

func exportDataFunc(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if exportDataID == "" {
		return fmt.Errorf("--id required")
	}

	// Discover RPC if not provided
	if exportDataRPC == "" {
		exportDataRPC = discoverRPC(exportDataID)
		ux.Logger.PrintToUser("🔍 RPC: %s", exportDataRPC)
	}

	// Get current block if end not specified
	if exportDataEnd == 0 {
		current, err := getCurrentBlock(ctx, exportDataRPC)
		if err != nil {
			return fmt.Errorf("failed to get current block: %w", err)
		}
		exportDataEnd = current
	}

	ux.Logger.PrintToUser("📤 Exporting blocks %d-%d", exportDataStart, exportDataEnd)

	// Export in batches
	allBlocks := []interface{}{}
	batchSize := uint64(100)

	for start := exportDataStart; start <= exportDataEnd; start += batchSize {
		end := start + batchSize - 1
		if end > exportDataEnd {
			end = exportDataEnd
		}

		ux.Logger.PrintToUser("  Blocks %d-%d...", start, end)
		blocks, err := getBlocks(ctx, exportDataRPC, start, end)
		if err != nil {
			return fmt.Errorf("failed to get blocks: %w", err)
		}
		allBlocks = append(allBlocks, blocks...)
	}

	// Write JSON
	data, err := json.MarshalIndent(map[string]interface{}{
		"blockchainID": exportDataID,
		"startBlock":   exportDataStart,
		"endBlock":     exportDataEnd,
		"blockCount":   len(allBlocks),
		"exportTime":   time.Now().Unix(),
		"blocks":       allBlocks,
	}, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(exportDataOut, data, 0644); err != nil {
		return err
	}

	ux.Logger.PrintToUser("✅ Exported %d blocks to %s", len(allBlocks), exportDataOut)
	return nil
}

// Helper functions

func discoverRPC(blockchainID string) string {
	if blockchainID == "C" {
		return "http://127.0.0.1:9630/ext/bc/C/rpc"
	}
	return fmt.Sprintf("http://127.0.0.1:9630/ext/bc/%s/rpc", blockchainID)
}

func getCurrentBlock(ctx context.Context, rpcURL string) (uint64, error) {
	req := &rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	var result string
	if err := callRPC(ctx, rpcURL, req, &result); err != nil {
		return 0, err
	}

	var blockNum uint64
	_, err := fmt.Sscanf(result, "0x%x", &blockNum)
	return blockNum, err
}

func getBlocks(ctx context.Context, rpcURL string, start, end uint64) ([]interface{}, error) {
	req := &rpcRequest{
		JSONRPC: "2.0",
		Method:  "migrate_getBlocks",
		Params:  []interface{}{start, end, 100},
		ID:      1,
	}

	var blocks []interface{}
	if err := callRPC(ctx, rpcURL, req, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	ID      int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func callRPC(ctx context.Context, rpcURL string, req *rpcRequest, result interface{}) error {
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

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return json.Unmarshal(rpcResp.Result, result)
}
