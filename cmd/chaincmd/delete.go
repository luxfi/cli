// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var forceDelete bool

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [chainName]",
		Short: "Delete a blockchain configuration",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteChain,
	}

	cmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip confirmation")
	return cmd
}

func deleteChain(cmd *cobra.Command, args []string) error {
	chainName := args[0]

	if !app.ChainConfigExists(chainName) {
		return fmt.Errorf("chain %s not found", chainName)
	}

	if !forceDelete {
		confirm, err := app.Prompt.CaptureYesNo(fmt.Sprintf("Delete chain %s?", chainName))
		if err != nil {
			return err
		}
		if !confirm {
			ux.Logger.PrintToUser("Cancelled")
			return nil
		}
	}

	chainDir := filepath.Join(app.GetChainsDir(), chainName)
	if err := os.RemoveAll(chainDir); err != nil {
		return fmt.Errorf("failed to delete chain: %w", err)
	}

	ux.Logger.PrintToUser("Deleted chain %s", chainName)
	return nil
}
