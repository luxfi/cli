// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package selfcmd implements CLI self-management commands.
// This provides nvm-style version management for the Lux CLI itself.
package selfcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates and returns the self command
func NewCmd(injectedApp *application.Lux, version string) *cobra.Command {
	app = injectedApp

	cmd := &cobra.Command{
		Use:   "self",
		Short: "Manage the Lux CLI installation",
		Long: `Commands for managing the Lux CLI installation.

Similar to nvm for Node.js, this allows you to:
- Link development builds to ~/.lux/bin/
- Install specific versions
- Switch between versions
- Self-update

EXAMPLES:

  # Link current binary to ~/.lux/bin/lux
  lux self link

  # Install a specific version
  lux self install v1.22.5

  # List installed versions
  lux self list

  # Use a specific version
  lux self use v1.22.5`,
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(newLinkCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newInstallCmd(version))
	cmd.AddCommand(newUseCmd())

	return cmd
}
