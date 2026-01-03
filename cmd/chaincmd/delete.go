// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chaincmd

import (
	"fmt"
	"path/filepath"

	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/safety"
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

	cmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip confirmation prompt (required in non-interactive mode)")
	return cmd
}

func deleteChain(cmd *cobra.Command, args []string) error {
	chainName := args[0]

	// Basic validation (detailed validation in safety package)
	if chainName == "" || filepath.Base(chainName) != chainName {
		return fmt.Errorf("invalid chain name: %s", chainName)
	}

	if !app.ChainConfigExists(chainName) {
		return fmt.Errorf("chain %s not found", chainName)
	}

	if !forceDelete {
		// In non-interactive mode, require --force flag
		if !prompts.IsInteractive() {
			return fmt.Errorf("confirmation required: use --force to delete without confirmation")
		}
		confirm, err := app.Prompt.CaptureYesNo(fmt.Sprintf("Delete chain %s?", chainName))
		if err != nil {
			return err
		}
		if !confirm {
			ux.Logger.PrintToUser("Cancelled")
			return nil
		}
	}

	// Use safety package for protected path deletion
	if err := safety.RemoveChainConfig(app.GetBaseDir(), chainName); err != nil {
		return fmt.Errorf("failed to delete chain: %w", err)
	}

	ux.Logger.PrintToUser("Deleted chain %s", chainName)
	return nil
}
