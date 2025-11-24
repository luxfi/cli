// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

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
	exportID    string
	exportRPC   string
	exportStart uint64
	exportEnd   uint64
	exportOut   string
)

// lux blockchain export
func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export blockchain data via RPC",
		Long: `Export blockchain blocks from a running node via RPC to JSONL format.

Each block is written as a single JSON line for efficient streaming and processing.

Example:
  lux blockchain export --id=dnmzhuf6... --output=blocks.jsonl`,
		RunE: exportFunc,
	}

	cmd.Flags().StringVar(&exportID, "id", "", "Blockchain ID (required)")
	cmd.Flags().StringVar(&exportRPC, "rpc", "", "RPC endpoint (auto-discovered from ID)")
	cmd.Flags().Uint64Var(&exportStart, "start-block", 0, "Start block")
	cmd.Flags().Uint64Var(&exportEnd, "end-block", 0, "End block (0=current)")
	cmd.Flags().StringVarP(&exportOut, "output", "o", "blocks.jsonl", "Output file (JSONL format)")

	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func exportFunc(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Discover RPC if not provided
	if exportRPC == "" {
		exportRPC = discoverRPC(exportID)
		ux.Logger.PrintToUser("🔍 RPC: %s", exportRPC)
	}

	// Get current block if end not specified
	if exportEnd == 0 {
		current, err := getCurrentBlock(ctx, exportRPC)
		if err != nil {
			return fmt.Errorf("failed to get current block: %w", err)
		}
		exportEnd = current
	}

	ux.Logger.PrintToUser("📤 Exporting blocks %d-%d to %s", exportStart, exportEnd, exportOut)

	// Open output file
	f, err := os.Create(exportOut)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	defer writer.Flush()

	// Export in batches, write as JSONL
	batchSize := uint64(100)
	exported := 0

	for start := exportStart; start <= exportEnd; start += batchSize {
		end := start + batchSize - 1
		if end > exportEnd {
			end = exportEnd
		}

		ux.Logger.PrintToUser("  Blocks %d-%d...", start, end)
		blocks, err := getBlocks(ctx, exportRPC, start, end)
		if err != nil {
			return fmt.Errorf("failed to get blocks: %w", err)
		}

		// Write each block as a JSON line
		for _, block := range blocks {
			data, err := json.Marshal(block)
			if err != nil {
				return fmt.Errorf("failed to marshal block: %w", err)
			}
			if _, err := writer.Write(data); err != nil {
				return fmt.Errorf("failed to write block: %w", err)
			}
			if _, err := writer.WriteString("\n"); err != nil {
				return fmt.Errorf("failed to write newline: %w", err)
			}
			exported++
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}

	ux.Logger.PrintToUser("✅ Exported %d blocks to %s", exported, exportOut)
	return nil
}
