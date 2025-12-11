// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/migrate"
	"github.com/luxfi/migrate/jsonl"
	"github.com/spf13/cobra"
)

var (
	// Import flags
	importID  string
	importRPC string
)

// lux network import
func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <path> [paths...]",
		Short: "Import blockchain blocks from JSONL files",
		Long: `Import blockchain blocks from JSONL file(s) to a running node via RPC.

USAGE:
  lux network import blocks.jsonl                    # Single JSONL file
  lux network import /path/to/blocks/               # Directory of JSONL files
  lux network import file1.jsonl file2.jsonl       # Multiple JSONL files

The blocks are imported to the target blockchain via RPC. By default imports to
C-Chain. Use --id to specify a different blockchain.

OPTIONS:
  --id      Blockchain ID (C, P, X, or blockchain ID)  [default: C]
  --rpc     RPC endpoint (auto-discovered from ID if not provided)

EXAMPLES:
  # Import ZOO mainnet blocks to subnet
  lux network import zoo-blocks.jsonl --id=2p4rdG...

  # Import multiple files in order
  lux network import blocks-000.jsonl blocks-001.jsonl blocks-002.jsonl

  # Import all JSONL files in a directory
  lux network import /home/z/exports/zoo/

For importing blockchain configurations, use:
  lux network import config <file>
  lux network import public --blockchain-id=...`,
		RunE: importFunc,
		Args: cobra.MinimumNArgs(1),
	}

	cmd.Flags().StringVar(&importID, "id", "C", "Blockchain ID (C, P, X, or blockchain ID)")
	cmd.Flags().StringVar(&importRPC, "rpc", "", "RPC endpoint (auto-discovered from ID if not provided)")

	// Add subcommands for config and public imports
	cmd.AddCommand(newImportConfigCmd())
	cmd.AddCommand(newImportPublicCmd())

	return cmd
}

func importFunc(_ *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	// Collect all JSONL files from arguments
	var files []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", arg, err)
		}

		if info.IsDir() {
			// Directory: find all .jsonl files
			dirFiles, err := findJSONLFiles(arg)
			if err != nil {
				return fmt.Errorf("failed to scan directory %s: %w", arg, err)
			}
			if len(dirFiles) == 0 {
				return fmt.Errorf("no .jsonl files found in %s", arg)
			}
			files = append(files, dirFiles...)
		} else {
			// Single file
			files = append(files, arg)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no JSONL files found")
	}

	ux.Logger.PrintToUser("Found %d JSONL file(s) to import", len(files))

	// Discover RPC if not provided
	if importRPC == "" {
		importRPC = discoverRPC(importID)
	}
	ux.Logger.PrintToUser("Target: %s (RPC: %s)", importID, importRPC)

	// Determine VM type from ID
	vmType := vmTypeFromID(importID)

	// Create importer using migrate package
	importer, err := migrate.NewImporter(migrate.ImporterConfig{
		VMType: vmType,
		RPCURL: importRPC,
	})
	if err != nil {
		return fmt.Errorf("failed to create importer: %w", err)
	}
	defer importer.Close()

	// Process each file
	totalImported := 0
	for i, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ux.Logger.PrintToUser("\n[%d/%d] Importing %s...", i+1, len(files), filepath.Base(file))

		imported, err := importFile(ctx, importer, file)
		if err != nil {
			return fmt.Errorf("failed to import %s: %w", file, err)
		}
		totalImported += imported
	}

	ux.Logger.PrintToUser("\nTotal: imported %d blocks from %d file(s)", totalImported, len(files))
	return nil
}

func importFile(ctx context.Context, importer migrate.Importer, file string) (int, error) {
	// Use streaming reader for memory efficiency with large files
	streamReader := jsonl.NewStreamReader(file)
	blocks, errs := streamReader.ReadBlocks()

	// Import blocks as they stream in, batching for efficiency
	batchSize := 100
	batch := make([]*migrate.BlockData, 0, batchSize)
	imported := 0
	lastProgress := time.Now()
	var firstBlock, lastBlock *migrate.BlockData

	for {
		select {
		case <-ctx.Done():
			return imported, ctx.Err()

		case err := <-errs:
			if err != nil {
				return imported, fmt.Errorf("stream error: %w", err)
			}

		case block, ok := <-blocks:
			if !ok {
				// Channel closed, import remaining batch
				if len(batch) > 0 {
					if err := importer.ImportBlocks(batch); err != nil {
						return imported, fmt.Errorf("failed to import final batch: %w", err)
					}
					imported += len(batch)
				}
				if firstBlock != nil && lastBlock != nil {
					ux.Logger.PrintToUser("  %d blocks (height %d to %d)", imported, firstBlock.Number, lastBlock.Number)
				}
				return imported, nil
			}

			if firstBlock == nil {
				firstBlock = block
			}
			lastBlock = block
			batch = append(batch, block)

			// Flush batch when full
			if len(batch) >= batchSize {
				if err := importer.ImportBlocks(batch); err != nil {
					return imported, fmt.Errorf("failed to import blocks at %d: %w", batch[0].Number, err)
				}
				imported += len(batch)
				batch = batch[:0]

				// Progress every 5 seconds or 10000 blocks
				if time.Since(lastProgress) > 5*time.Second || imported%10000 == 0 {
					ux.Logger.PrintToUser("  imported %d blocks (latest: %d)", imported, lastBlock.Number)
					lastProgress = time.Now()
				}
			}
		}
	}
}

// findJSONLFiles finds all .jsonl files in a directory, sorted by name
func findJSONLFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".jsonl") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	// Files are already sorted by ReadDir
	return files, nil
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

// discoverRPC auto-discovers the RPC endpoint for a given blockchain ID
func discoverRPC(id string) string {
	switch id {
	case "C":
		return "http://127.0.0.1:9630/ext/bc/C/rpc"
	case "P":
		return "http://127.0.0.1:9630/ext/bc/P"
	case "X":
		return "http://127.0.0.1:9630/ext/bc/X"
	default:
		// Assume it's a blockchain ID
		return fmt.Sprintf("http://127.0.0.1:9630/ext/bc/%s/rpc", id)
	}
}
