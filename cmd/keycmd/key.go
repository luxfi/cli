// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"fmt"

	"github.com/luxdefi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp

	cmd := &cobra.Command{
		Use:   "key",
		Short: "Create and manage testnet signing keys",
		Long: `The key command suite provides a collection of tools for creating and managing
signing keys. You can use these keys to deploy Subnets to the Fuji Testnet,
but these keys are NOT suitable to use in production environments. DO NOT use
these keys on Mainnet.

To get started, use the key create command.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	// lux key create
	cmd.AddCommand(newCreateCmd())

	// lux key list
	cmd.AddCommand(newListCmd())

	// lux key delete
	cmd.AddCommand(newDeleteCmd())

	// lux key export
	cmd.AddCommand(newExportCmd())

	// lux key transfer
	cmd.AddCommand(newTransferCmd())

	return cmd
}
