// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	exportChainFlag string
	exportPath      string
	exportStart     uint64
	exportEnd       uint64
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export blocks from a running chain to RLP file",
		Long: `Export blocks from a running blockchain to an RLP file.

The chain must be running on the local network.

Examples:
  # Export all blocks from C-Chain
  lux chain export --chain=c --path=/tmp/blocks.rlp

  # Export specific block range
  lux chain export --chain=mychain --path=/tmp/blocks.rlp --start=1000 --end=2000`,
		RunE: exportChain,
	}

	cmd.Flags().StringVar(&exportChainFlag, "chain", "", "Chain to export (required)")
	cmd.Flags().StringVar(&exportPath, "path", "", "Output file path (required)")
	cmd.Flags().Uint64Var(&exportStart, "start", 0, "Start block number")
	cmd.Flags().Uint64Var(&exportEnd, "end", 0, "End block number (0 = latest)")
	cmd.MarkFlagRequired("chain")
	cmd.MarkFlagRequired("path")

	return cmd
}

func exportChain(cmd *cobra.Command, args []string) error {
	// TODO: Implement export via admin_exportChain RPC
	return fmt.Errorf("export command not yet implemented - use admin_exportChain RPC directly")
}
