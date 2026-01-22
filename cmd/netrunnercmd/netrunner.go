// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package netrunnercmd implements the netrunner command and subcommands.
package netrunnercmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates and returns the netrunner command
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp

	cmd := &cobra.Command{
		Use:   "netrunner",
		Short: "Manage the network runner",
		Long: `Commands for managing the Lux network runner.

The netrunner is used for local network testing and development.`,
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(newLinkCmd())

	return cmd
}
