// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package tokentransferrercmd

import (
	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/luxfi/cli/v2/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux interchain tokenTransferrer
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokenTransferrer",
		Short: "Manage Token Transferrers",
		Long:  `The tokenTransfer command suite provides tools to deploy and manage Token Transferrers.`,
		RunE:  cobrautils.CommandSuiteUsage,
	}
	app = injectedApp
	// tokenTransferrer deploy
	cmd.AddCommand(NewDeployCmd())
	return cmd
}
