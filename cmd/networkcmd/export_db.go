// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/migrate"
	"github.com/luxfi/migrate/jsonl"
	_ "github.com/luxfi/migrate/subnetevm" // Register SubnetEVM exporter
	"github.com/spf13/cobra"
)

var (
	exportDBPath       string
	exportDBOutput     string
	exportDBStartBlock uint64
	exportDBEndBlock   uint64
)

// lux network export db
func newExportDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db <path>",
		Short: "Export blockchain blocks from PebbleDB",
		Long: `Export blockchain blocks directly from a PebbleDB database to JSONL format.

This exports blocks without needing a running node, directly reading from the database files.

USAGE:
  lux network export db /path/to/pebbledb                    # Export all blocks
  lux network export db /path/to/pebbledb -o blocks.jsonl   # Specify output file
  lux network export db /path/to/pebbledb --start=0 --end=1000  # Export range

EXAMPLES:
  # Export ZOO mainnet blocks from chaindata
  lux network export db ~/work/lux/state/chaindata/zoo-mainnet-200200/db/pebbledb

  # Export specific block range
  lux network export db ~/work/lux/state/chaindata/zoo-mainnet-200200/db/pebbledb --start=0 --end=10000`,
		RunE: exportDBFunc,
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().StringVarP(&exportDBOutput, "output", "o", "blocks.jsonl", "Output file (JSONL format)")
	cmd.Flags().Uint64Var(&exportDBStartBlock, "start", 0, "Start block (default: 0)")
	cmd.Flags().Uint64Var(&exportDBEndBlock, "end", 0, "End block (0=latest)")

	return cmd
}

func exportDBFunc(_ *cobra.Command, args []string) error {
	dbPath := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	ux.Logger.PrintToUser("Opening database: %s", dbPath)

	// Create exporter using migrate package
	exporter, err := migrate.NewExporter(migrate.ExporterConfig{
		VMType:       migrate.VMTypeSubnetEVM,
		DatabasePath: dbPath,
		DatabaseType: "pebble",
	})
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}
	defer exporter.Close()

	// Get chain info
	info, err := exporter.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get chain info: %w", err)
	}

	ux.Logger.PrintToUser("Chain ID: %s", info.ChainID)
	ux.Logger.PrintToUser("Genesis: %s", info.GenesisHash.Hex())
	ux.Logger.PrintToUser("Head block: %d", info.CurrentHeight)

	// Determine block range
	startBlock := exportDBStartBlock
	endBlock := exportDBEndBlock
	if endBlock == 0 {
		endBlock = info.CurrentHeight
	}

	if startBlock > endBlock {
		return fmt.Errorf("start block (%d) cannot be greater than end block (%d)", startBlock, endBlock)
	}

	ux.Logger.PrintToUser("Exporting blocks %d to %d -> %s", startBlock, endBlock, exportDBOutput)

	// Create JSONL writer
	writer, err := jsonl.NewWriter(exportDBOutput)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer writer.Close()

	// Export blocks
	blocks, errs := exporter.ExportBlocks(ctx, startBlock, endBlock)

	exported := 0
	lastProgress := time.Now()

	for {
		select {
		case block, ok := <-blocks:
			if !ok {
				// Channel closed, check for errors
				select {
				case err := <-errs:
					if err != nil {
						return fmt.Errorf("export error: %w", err)
					}
				default:
				}
				ux.Logger.PrintToUser("\nExported %d blocks to %s", exported, exportDBOutput)
				return nil
			}

			if err := writer.WriteBlock(block); err != nil {
				return fmt.Errorf("failed to write block %d: %w", block.Number, err)
			}
			exported++

			// Progress every 5 seconds or 1000 blocks
			if time.Since(lastProgress) > 5*time.Second || exported%1000 == 0 {
				progress := float64(block.Number-startBlock) / float64(endBlock-startBlock) * 100
				ux.Logger.PrintToUser("  Block %d (%.1f%%)", block.Number, progress)
				lastProgress = time.Now()
			}

		case err := <-errs:
			if err != nil {
				return fmt.Errorf("export error: %w", err)
			}
		}
	}
}
