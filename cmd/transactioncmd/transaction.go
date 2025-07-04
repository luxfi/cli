// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package transactioncmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// avalanche subnet vm
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transaction",
		Short: "Sign and execute specific transactions",
		Long:  `The transaction command suite provides all of the utilities required to sign multisig transactions.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	// subnet upgrade vm
	cmd.AddCommand(newTransactionSignCmd())
	// subnet upgrade generate
	cmd.AddCommand(newTransactionCommitCmd())
	return cmd
}
