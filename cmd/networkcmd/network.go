// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux network
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage local network runtime",
		Long: `The network command manages local network runtime operations.

Commands:
  start   - Start the local network
  stop    - Stop the local network
  status  - Show local network status
  clean   - Clean local network state

For blockchain/chain management, use 'lux chain' instead.`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	// Local network runtime operations only
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newCleanCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}
