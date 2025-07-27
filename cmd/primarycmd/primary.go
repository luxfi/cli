// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package primarycmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux primary
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "primary",
		Short: "Interact with the Primary Network",
		Long: `The primary command suite provides a collection of tools for interacting with the
Primary Network`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	app = injectedApp
	// primary addValidator
	cmd.AddCommand(newAddValidatorCmd())
	// primary describe
	cmd.AddCommand(newDescribeCmd())
	return cmd
}
