// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"github.com/luxfi/cli/cmd/chaincmd/upgradecmd"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the unified chain command suite for all blockchain operations
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "chain",
		Short: "Manage blockchain lifecycle - create, deploy, import, export, validate",
		Long: `The chain command provides unified operations for all blockchain types.

Chain Types:
  L1 (Sovereign)  - Independent validator set, own tokenomics
  L2 (Rollup)     - Based on L1 sequencing (Lux, Ethereum, etc.)
  L3 (App Chain)  - Built on L2 for application-specific use

Common Operations:
  create    Create a new blockchain configuration
  deploy    Deploy to local network, testnet, or mainnet
  list      List all configured blockchains
  describe  Show detailed blockchain information
  import    Import blocks from RLP file
  export    Export blocks to RLP file

Examples:
  # Create a new L2 based rollup
  lux chain create mychain --type=l2 --sequencer=lux

  # Create a sovereign L1
  lux chain create mychain --type=l1

  # Deploy to local network
  lux chain deploy mychain --local

  # Import historical blocks
  lux chain import --chain=mychain --path=/tmp/blocks.rlp

  # List all chains
  lux chain list`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	// Core lifecycle commands
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newDescribeCmd())
	cmd.AddCommand(newDeleteCmd())

	// Data operations
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newExportCmd())

	// Validator management
	cmd.AddCommand(newValidatorsCmd())
	cmd.AddCommand(newAddValidatorCmd())
	cmd.AddCommand(newRemoveValidatorCmd())

	// Upgrade and migration
	cmd.AddCommand(upgradecmd.NewCmd(app))
	cmd.AddCommand(newMigrateCmd())

	return cmd
}
