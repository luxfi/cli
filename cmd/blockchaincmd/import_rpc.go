// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	importRPCURL      string
	importBlockchainID string
	importInputFile   string
)

// lux blockchain import-rpc <blockchain-id>
func newImportRPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-rpc [blockchain-id]",
		Short: "Import blockchain data via RPC",
		Long: `Import blockchain blocks via RPC to a running node.
Uses the migrate_importBlocks RPC endpoint to import blocks.

Example:
  lux blockchain import-rpc C --input=blocks.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: importRPCFunc,
	}

	cmd.Flags().StringVar(&importRPCURL, "rpc-url", "", "RPC endpoint (auto-discovered if not specified)")
	cmd.Flags().StringVar(&importInputFile, "input", "blocks.json", "Input file path")

	return cmd
}

func importRPCFunc(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Get blockchain ID from arg
	var blockchainID string
	if len(args) > 0 {
		blockchainID = args[0]
	} else if importBlockchainID != "" {
		blockchainID = importBlockchainID
	} else {
		blockchainID = "C" // Default to C-Chain
	}

	// Discover RPC endpoint if not provided
	if importRPCURL == "" {
		importRPCURL = discoverBlockchainRPC(blockchainID)
		ux.Logger.PrintToUser("🔍 Using RPC: %s", importRPCURL)
	}

	// Read input file
	data, err := os.ReadFile(importInputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var exportData map[string]interface{}
	if err := json.Unmarshal(data, &exportData); err != nil {
		return fmt.Errorf("failed to parse input file: %w", err)
	}

	blocks, ok := exportData["blocks"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid export format: missing blocks array")
	}

	ux.Logger.PrintToUser("📥 Importing %d blocks...", len(blocks))

	// Import in batches of 100
	batchSize := 100
	imported := 0

	for i := 0; i < len(blocks); i += batchSize {
		end := i + batchSize
		if end > len(blocks) {
			end = len(blocks)
		}

		batch := blocks[i:end]
		ux.Logger.PrintToUser("  Importing batch %d-%d...", i, end-1)

		count, err := importBlocksRPC(ctx, importRPCURL, batch)
		if err != nil {
			return fmt.Errorf("failed to import batch %d-%d: %w", i, end-1, err)
		}

		imported += count
	}

	ux.Logger.PrintToUser("✅ Imported %d blocks", imported)
	return nil
}

// importBlocksRPC calls migrate_importBlocks RPC endpoint
func importBlocksRPC(ctx context.Context, rpcURL string, blocks []interface{}) (int, error) {
	req := &RPCRequest{
		JSONRPC: "2.0",
		Method:  "migrate_importBlocks",
		Params:  []interface{}{blocks},
		ID:      1,
	}

	var count int
	if err := callRPCGeneric(ctx, rpcURL, req, &count); err != nil {
		return 0, err
	}

	return count, nil
}
