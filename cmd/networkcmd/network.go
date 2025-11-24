// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var (
	app         *application.Lux
	luxdVersion string
	numNodes    uint32
)

func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:     "network",
		Aliases: []string{"net"},
		Short:   "Manage locally deployed subnets",
		Long: `The network command suite provides a collection of tools for managing local Subnet
deployments.

When you deploy a Subnet locally, it runs on a local, multi-node Lux network. The
subnet deploy command starts this network in the background. This command suite allows you
to shutdown, restart, and clear that network.

This network currently supports multiple, concurrently deployed Subnets.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
		Args: cobra.ExactArgs(0),
	}
	// network start
	cmd.AddCommand(newStartCmd())
	// network stop
	cmd.AddCommand(newStopCmd())
	// network clean
	cmd.AddCommand(newCleanCmd())
	// network status
	cmd.AddCommand(newStatusCmd())
	// network quickstart
	cmd.AddCommand(newQuickstartCmd())
	// network import-genesis
	cmd.AddCommand(newImportGenesisCmd())
	// network export
	cmd.AddCommand(newExportCmd())
	// network import
	cmd.AddCommand(newImportCmd())
	return cmd
}
