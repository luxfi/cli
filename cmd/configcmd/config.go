// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package configcmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Modify configuration for Lux CLI",
		Long:  `Customize configuration for Lux CLI`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	// set user metrics collection preferences cmd
	cmd.AddCommand(newMetricsCmd())

	return cmd
}
