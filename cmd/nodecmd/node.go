// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd returns a new cobra.Command for node operations
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage Lux node operations",
		Long: `The node command suite provides tools for managing Lux node operations including
development mode, automining, and advanced node configurations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newDevCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newAutominingCmd())
	cmd.AddCommand(newValidatorCmd())
	
	return cmd
}