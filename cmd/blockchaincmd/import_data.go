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
	importDataID     string
	importDataRPC    string
	importDataDir    string
	importDataInput  string
)

// lux blockchain import --data-dir=...
func newImportDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-data",
		Short: "Import blockchain data via RPC",
		Long: `Import blockchain blocks to a running node via RPC.

Example:
  lux blockchain import-data --id=C --input=blocks.json`,
		RunE: importDataFunc,
	}

	cmd.Flags().StringVar(&importDataID, "id", "C", "Blockchain ID")
	cmd.Flags().StringVar(&importDataRPC, "rpc", "", "RPC endpoint (auto-discovered from ID)")
	cmd.Flags().StringVar(&importDataDir, "data-dir", "", "Data directory (for discovery)")
	cmd.Flags().StringVar(&importDataInput, "input", "blocks.json", "Input file")

	return cmd
}

func importDataFunc(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Discover RPC if not provided
	if importDataRPC == "" {
		importDataRPC = discoverRPC(importDataID)
		ux.Logger.PrintToUser("🔍 RPC: %s", importDataRPC)
	}

	// Read input file
	data, err := os.ReadFile(importDataInput)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	var exportData map[string]interface{}
	if err := json.Unmarshal(data, &exportData); err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	blocks, ok := exportData["blocks"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid format: missing blocks array")
	}

	ux.Logger.PrintToUser("📥 Importing %d blocks...", len(blocks))

	// Import in batches
	batchSize := 100
	imported := 0

	for i := 0; i < len(blocks); i += batchSize {
		end := i + batchSize
		if end > len(blocks) {
			end = len(blocks)
		}

		batch := blocks[i:end]
		ux.Logger.PrintToUser("  Batch %d-%d...", i, end-1)

		count, err := importBlocks(ctx, importDataRPC, batch)
		if err != nil {
			return fmt.Errorf("failed to import batch: %w", err)
		}
		imported += count
	}

	ux.Logger.PrintToUser("✅ Imported %d blocks", imported)
	return nil
}

func importBlocks(ctx context.Context, rpcURL string, blocks []interface{}) (int, error) {
	req := &rpcRequest{
		JSONRPC: "2.0",
		Method:  "migrate_importBlocks",
		Params:  []interface{}{blocks},
		ID:      1,
	}

	var count int
	if err := callRPC(ctx, rpcURL, req, &count); err != nil {
		return 0, err
	}
	return count, nil
}
