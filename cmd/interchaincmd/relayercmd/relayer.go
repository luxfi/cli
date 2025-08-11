// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayercmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux interchain relayer
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayer",
		Short: "Manage Warp relayers",
		Long: `The relayer command suite provides a collection of tools for deploying
and configuring an Warp relayers.`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	app = injectedApp
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newFundCmd())
	return cmd
}
