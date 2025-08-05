// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux l1
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "l1",
		Short: "Create and manage sovereign L1 blockchains",
		Long: `The l1 command suite provides tools for creating and managing sovereign L1 blockchains.

L1s are independent blockchains that can be deployed on Lux infrastructure with their own
validator sets, tokenomics, and consensus mechanisms. They support both Proof of Authority (PoA)
and Proof of Stake (PoS) validator management.

To get started, use the l1 create command to configure your L1, then deploy it with l1 deploy.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp

	// l1 create
	cmd.AddCommand(newCreateCmd())
	// l1 deploy
	cmd.AddCommand(newDeployCmd())
	// l1 list
	cmd.AddCommand(newListCmd())
	// l1 describe
	cmd.AddCommand(newDescribeCmd())
	// l1 validator
	cmd.AddCommand(newValidatorCmd())
	// l1 upgrade
	cmd.AddCommand(newUpgradeCmd())
	// l1 import
	cmd.AddCommand(newImportCmd())
	// l1 migrate
	cmd.AddCommand(newMigrateCmd())

	return cmd
}
