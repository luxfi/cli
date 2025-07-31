// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package messengercmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux interchain messenger
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messenger",
		Short: "Interact with ICM messenger contracts",
		Long: `The messenger command suite provides a collection of tools for interacting
with ICM messenger contracts.`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	app = injectedApp
	// interchain messenger sendMsg
	cmd.AddCommand(NewSendMsgCmd())
	// interchain messenger deploy
	cmd.AddCommand(NewDeployCmd())
	return cmd
}
