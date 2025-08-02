// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package contractcmd

import (
	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux contract
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract",
		Short: "Manage smart contracts",
		Long: `The contract command suite provides a collection of tools for deploying
and interacting with smart contracts on Lux networks.`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	app = injectedApp
	// contract deploy
	cmd.AddCommand(newDeployCmd())
	// contract initValidatorManager
	cmd.AddCommand(newInitValidatorManagerCmd())
	return cmd
}