// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package l3cmd

import (
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured L3s",
		RunE:  listL3s,
	}

	return cmd
}

func listL3s(cmd *cobra.Command, args []string) error {
	ux.Logger.PrintToUser("📋 Configured L3s:")
	ux.Logger.PrintToUser("==================")

	// List all L3 chains
	l3Chains := []string{}

	// Check for L3 configurations in the base directory
	baseDir := app.GetBaseDir()
	l3Dir := filepath.Join(baseDir, "l3")

	if entries, err := os.ReadDir(l3Dir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				l3Chains = append(l3Chains, entry.Name())
			}
		}
	}

	if len(l3Chains) == 0 {
		ux.Logger.PrintToUser("  (No L3 chains found)")
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("💡 Create your first L3 with: lux l3 create <name>")
	} else {
		for _, chain := range l3Chains {
			ux.Logger.PrintToUser("  • %s", chain)
		}
	}

	return nil
}
