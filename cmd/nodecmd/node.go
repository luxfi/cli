// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd returns a new cobra.Command for node operations
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage Lux node operations",
		Long: `The node command suite provides tools for managing Lux node operations including:

• Join existing networks (mainnet/testnet) as a validator
• Run local development nodes
• Manage validator operations and staking
• Configure automining and advanced settings
• Monitor node status and health

Examples:
  # Join mainnet as a validator
  lux node join --mainnet --stake

  # Join testnet 
  lux node join --testnet

  # Run local development node
  lux node dev

  # Check node status
  lux node status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newJoinCmd())      // Join existing networks
	cmd.AddCommand(newDevCmd())       // Development mode
	cmd.AddCommand(newStartCmd())     // Start node
	cmd.AddCommand(newAutominingCmd()) // Automining config
	cmd.AddCommand(newValidatorCmd()) // Validator operations
	cmd.AddCommand(newVersionCmd())   // Version info
	cmd.AddCommand(newStatusCmd())    // Node status
	
	return cmd
}