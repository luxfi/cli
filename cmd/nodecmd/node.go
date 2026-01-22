// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the node command suite.
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage luxd node binary",
		Long: `Commands for managing the luxd node binary.

The node command suite helps configure which luxd binary the CLI uses
for local network operations.

COMMANDS:

  link      Symlink a luxd binary to ~/.lux/bin/luxd

EXAMPLES:

  # Link a specific luxd binary
  lux node link /path/to/luxd

  # Auto-detect from ../node/build/luxd (relative to CLI)
  lux node link --auto

  # Show current linked binary
  ls -la ~/.lux/bin/luxd`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	cmd.AddCommand(newLinkCmd())

	return cmd
}
