// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l3cmd

import (
	"github.com/luxfi/cli/v2/pkg/ux"
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
	// TODO: Implement L3 listing logic
	return nil
}