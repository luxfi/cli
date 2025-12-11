// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all key sets",
		Long: `List all key sets stored in ~/.lux/keys/

Shows the name of each key set. Use 'lux key show <name>' for details.

Example:
  lux key list
  lux key ls`,
		Args: cobra.NoArgs,
		RunE: runList,
	}

	return cmd
}

func runList(_ *cobra.Command, _ []string) error {
	keys, err := key.ListKeySets()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		ux.Logger.PrintToUser("No key sets found.")
		ux.Logger.PrintToUser("Use 'lux key create <name>' to create one.")
		return nil
	}

	ux.Logger.PrintToUser("Key sets:")
	ux.Logger.PrintToUser("")
	for _, k := range keys {
		ux.Logger.PrintToUser("  %s", k)
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Use 'lux key show <name>' for details.")

	return nil
}
