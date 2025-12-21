// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	exportID    string
	exportRPC   string
	exportStart uint64
	exportEnd   uint64
	exportOut   string
)

// lux network export
func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export blockchain data via admin RPC",
		Long: `Export blockchain blocks from a running node using admin_exportChain RPC.

Exports blocks in native RLP format which can be directly imported using admin_importChain.

Example:
  lux network export --id=C --output=blocks.rlp
  lux network export --id=zoo --output=zoo-blocks.rlp
  lux network export --id=C --start-block=0 --end-block=1000 -o blocks.rlp`,
		RunE: exportFunc,
	}

	cmd.Flags().StringVar(&exportID, "id", "", "Blockchain ID (C, zoo, or full ID)")
	cmd.Flags().StringVar(&exportRPC, "rpc", "", "RPC endpoint (auto-discovered from ID)")
	cmd.Flags().Uint64Var(&exportStart, "start-block", 0, "Start block")
	cmd.Flags().Uint64Var(&exportEnd, "end-block", 0, "End block (0=current)")
	cmd.Flags().StringVarP(&exportOut, "output", "o", "blocks.rlp", "Output file")

	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func exportFunc(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	// Discover admin RPC endpoint
	adminRPC := exportRPC
	if adminRPC == "" {
		adminRPC = discoverAdminRPC(exportID)
		ux.Logger.PrintToUser("Admin RPC: %s", adminRPC)
	}

	return exportRLP(ctx, adminRPC)
}

// exportRLP exports blocks using admin_exportChain (native geth RLP format)
func exportRLP(ctx context.Context, adminRPC string) error {
	ux.Logger.PrintToUser("Exporting blocks to %s (RLP format via admin_exportChain)", exportOut)

	// Build admin_exportChain request
	params := []interface{}{exportOut}
	if exportStart > 0 || exportEnd > 0 {
		params = append(params, exportStart)
		if exportEnd > 0 {
			params = append(params, exportEnd)
		}
	}

	reqData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "admin_exportChain",
		"params":  params,
		"id":      1,
	}

	jsonData, _ := json.Marshal(reqData)
	req, err := http.NewRequestWithContext(ctx, "POST", adminRPC, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call admin_exportChain: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result["error"] != nil {
		return fmt.Errorf("admin_exportChain error: %v", result["error"])
	}

	success, ok := result["result"].(bool)
	if !ok || !success {
		return fmt.Errorf("admin_exportChain failed: %v", result["result"])
	}

	ux.Logger.PrintToUser("Exported blocks to %s", exportOut)
	return nil
}

// discoverAdminRPC returns the admin RPC endpoint for a blockchain ID
func discoverAdminRPC(blockchainID string) string {
	// Handle well-known chain IDs
	switch strings.ToLower(blockchainID) {
	case "c", "c-chain", "cchain":
		return "http://127.0.0.1:9630/ext/bc/C/admin"
	case "zoo":
		return "http://127.0.0.1:9630/ext/bc/zoo/admin"
	default:
		return fmt.Sprintf("http://127.0.0.1:9630/ext/bc/%s/admin", blockchainID)
	}
}
