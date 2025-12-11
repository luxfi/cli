// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var forceDelete bool

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a key set",
		Long: `Delete a key set from ~/.lux/keys/

WARNING: This permanently deletes all keys! Make sure you have backed up
the mnemonic phrase before deleting.

Example:
  lux key delete validator1
  lux key delete validator1 --force  # Skip confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: runDelete,
	}

	cmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(_ *cobra.Command, args []string) error {
	name := args[0]

	// Check if key set exists
	_, err := key.LoadKeySet(name)
	if err != nil {
		return fmt.Errorf("key set '%s' not found: %w", name, err)
	}

	if !forceDelete {
		ux.Logger.PrintToUser("WARNING: This will permanently delete all keys for '%s'!", name)
		ux.Logger.PrintToUser("Make sure you have backed up the mnemonic phrase.")
		ux.Logger.PrintToUser("")

		confirm, err := app.Prompt.CaptureYesNo(fmt.Sprintf("Delete key set '%s'?", name))
		if err != nil {
			return err
		}
		if !confirm {
			ux.Logger.PrintToUser("Cancelled.")
			return nil
		}
	}

	if err := key.DeleteKeySet(name); err != nil {
		return fmt.Errorf("failed to delete key set: %w", err)
	}

	ux.Logger.PrintToUser("Key set '%s' deleted.", name)
	return nil
}
