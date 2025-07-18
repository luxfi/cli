// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package upgradecmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux subnet vm
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade your Subnets",
		Long: `The subnet upgrade command suite provides a collection of tools for
updating your developmental and deployed Subnets.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	// subnet upgrade vm
	cmd.AddCommand(newUpgradeVMCmd())
	// subnet upgrade generate
	cmd.AddCommand(newUpgradeGenerateCmd())
	// subnet upgrade import
	cmd.AddCommand(newUpgradeImportCmd())
	// subnet upgrade export
	cmd.AddCommand(newUpgradeExportCmd())
	// subnet upgrade print
	cmd.AddCommand(newUpgradePrintCmd())
	// subnet upgrade apply
	cmd.AddCommand(newUpgradeApplyCmd())
	return cmd
}
