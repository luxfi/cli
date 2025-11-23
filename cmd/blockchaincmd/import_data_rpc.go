// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	importDataID    string
	importDataRPC   string
	importDataInput string
)

// lux blockchain import data
func newImportDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data",
		Short: "Import blockchain data via RPC",
		Long: `Import blockchain blocks to a running node via RPC from JSONL format.

Reads blocks from a JSONL file (one JSON object per line) and imports them.

Example:
  lux blockchain import data --id=C --input=blocks.jsonl`,
		RunE: importDataFunc,
	}

	cmd.Flags().StringVar(&importDataID, "id", "C", "Blockchain ID")
	cmd.Flags().StringVar(&importDataRPC, "rpc", "", "RPC endpoint (auto-discovered from ID)")
	cmd.Flags().StringVarP(&importDataInput, "input", "i", "blocks.jsonl", "Input file (JSONL format)")

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

	// Open input file
	f, err := os.Open(importDataInput)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Increase buffer size for large JSON lines
	const maxCapacity = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	blocks := []interface{}{}
	lineNum := 0

	// Read all blocks from JSONL
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var block map[string]interface{}
		if err := json.Unmarshal(line, &block); err != nil {
			return fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}
		blocks = append(blocks, block)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
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
