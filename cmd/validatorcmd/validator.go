// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package validatorcmd

import (
	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux validator
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Manage P-Chain validator balance",
		Long: `The validator command suite provides a collection of tools for managing validator
balance on P-Chain.

Validator's balance is used to pay for continuous fee to the P-Chain. When this Balance reaches 0, 
the validator will be considered inactive and will no longer participate in validating the L1`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	app = injectedApp
	// validator list
	cmd.AddCommand(NewListCmd())
	// validator getBalance
	cmd.AddCommand(NewGetBalanceCmd())
	// validator increaseBalance
	cmd.AddCommand(NewIncreaseBalanceCmd())
	return cmd
}
