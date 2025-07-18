// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l3cmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux l3
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "l3",
		Short: "Create and manage L3s (app-specific chains)",
		Long: `The l3 command suite provides tools for creating and deploying L3s.

L3s are application-specific chains that can be deployed on top of L2s,
providing even more customization and scalability for specific use cases.

L3s support:
- Custom execution environments
- App-specific state transitions
- Ultra-low latency (inherit from L2)
- Cross-layer messaging via IBC/Teleport

Examples:
  lux l3 create gaming-chain --l2 myL2 --vm custom
  lux l3 deploy gaming-chain --preconfirm`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	
	// l3 create
	cmd.AddCommand(newCreateCmd())
	// l3 deploy
	cmd.AddCommand(newDeployCmd())
	// l3 list
	cmd.AddCommand(newListCmd())
	// l3 describe
	cmd.AddCommand(newDescribeCmd())
	// l3 bridge
	cmd.AddCommand(newBridgeCmd())
	
	return cmd
}