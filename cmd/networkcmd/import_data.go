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
	"github.com/spf13/cobra"
)

var (
	importID    string
	importRPC   string
	importInput string
)

// lux blockchain import data
func newImportDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data",
		Short: "Import blockchain blocks via RPC",
		Long: `Import blockchain blocks to a running node via RPC from JSONL format.

Reads blocks from a JSONL file (one JSON object per line) and imports them
using the github.com/luxfi/migrate package.

Example:
  lux blockchain import data --id=C --input=blocks.jsonl`,
		RunE: importDataFunc,
	}

	cmd.Flags().StringVar(&importID, "id", "C", "Blockchain ID")
	cmd.Flags().StringVar(&importRPC, "rpc", "", "RPC endpoint (auto-discovered from ID)")
	cmd.Flags().StringVarP(&importInput, "input", "i", "blocks.jsonl", "Input file (JSONL format)")

	return cmd
}

func importDataFunc(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Discover RPC if not provided
	if importRPC == "" {
		importRPC = discoverRPC(importID)
		ux.Logger.PrintToUser("RPC: %s", importRPC)
	}

	// Determine VM type from ID
	vmType := vmTypeFromID(importID)
	ux.Logger.PrintToUser("VM Type: %s", vmType)

	// Create importer using migrate package
	importer, err := migrate.NewImporter(migrate.ImporterConfig{
		VMType: vmType,
		RPCURL: importRPC,
	})
	if err != nil {
		return fmt.Errorf("failed to create importer: %w", err)
	}
	defer importer.Close()

	// Open JSONL reader
	reader, err := jsonl.NewReader(importInput)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer reader.Close()

	// Read all blocks
	blocks, err := reader.ReadAllBlocks()
	if err != nil {
		return fmt.Errorf("failed to read blocks: %w", err)
	}

	ux.Logger.PrintToUser("Importing %d blocks...", len(blocks))

	// Import in batches
	batchSize := 100
	imported := 0

	for i := 0; i < len(blocks); i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(blocks) {
			end = len(blocks)
		}

		batch := blocks[i:end]
		ux.Logger.PrintToUser("  Batch %d-%d...", i, end-1)

		if err := importer.ImportBlocks(batch); err != nil {
			return fmt.Errorf("failed to import batch %d-%d: %w", i, end-1, err)
		}
		imported += len(batch)
	}

	// Finalize and verify
	if len(blocks) > 0 {
		lastBlock := blocks[len(blocks)-1]
		if err := importer.FinalizeImport(lastBlock.Number); err != nil {
			ux.Logger.PrintToUser("Warning: finalization check failed: %v", err)
		}
	}

	ux.Logger.PrintToUser("Imported %d blocks", imported)
	return nil
}

// vmTypeFromID converts a blockchain ID to a VMType
func vmTypeFromID(id string) migrate.VMType {
	switch id {
	case "C":
		return migrate.VMTypeCChain
	case "P":
		return migrate.VMTypePChain
	case "X":
		return migrate.VMTypeXChain
	default:
		// Default to C-Chain for subnet imports
		return migrate.VMTypeCChain
	}
}
