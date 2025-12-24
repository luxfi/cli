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

// NetworkTarget represents the target network for chain operations
type NetworkTarget string

const (
	NetworkMainnet NetworkTarget = "mainnet"
	NetworkTestnet NetworkTarget = "testnet"
	NetworkDevnet  NetworkTarget = "devnet"
	NetworkCustom  NetworkTarget = "custom"
)

// Network target flags
var (
	mainnet bool
	testnet bool
	devnet  bool
	custom  bool
)

// GetNetworkTarget returns the selected network target based on flags
func GetNetworkTarget() NetworkTarget {
	switch {
	case mainnet:
		return NetworkMainnet
	case testnet:
		return NetworkTestnet
	case devnet:
		return NetworkDevnet
	case custom:
		return NetworkCustom
	default:
		return NetworkCustom
	}
}

// addNetworkFlags adds network target flags to a command
func addNetworkFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&mainnet, "mainnet", "m", false, "Target mainnet")
	cmd.Flags().BoolVarP(&testnet, "testnet", "t", false, "Target testnet")
	cmd.Flags().BoolVarP(&devnet, "devnet", "d", false, "Target devnet")
	cmd.Flags().BoolVar(&custom, "custom", false, "Target custom network")
}

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
	createCmd := newCreateCmd()
	addNetworkFlags(createCmd)
	cmd.AddCommand(createCmd)

	deployCmd := newDeployCmd()
	// Note: deploy already has network flags, skip adding duplicates
	cmd.AddCommand(deployCmd)

	listCmd := newListCmd()
	addNetworkFlags(listCmd)
	cmd.AddCommand(listCmd)

	describeCmd := newDescribeCmd()
	addNetworkFlags(describeCmd)
	cmd.AddCommand(describeCmd)

	deleteCmd := newDeleteCmd()
	addNetworkFlags(deleteCmd)
	cmd.AddCommand(deleteCmd)

	// Data operations
	importCmd := newImportCmd()
	addNetworkFlags(importCmd)
	cmd.AddCommand(importCmd)

	exportCmd := newExportCmd()
	addNetworkFlags(exportCmd)
	cmd.AddCommand(exportCmd)

	// Validator management
	validatorsCmd := newValidatorsCmd()
	addNetworkFlags(validatorsCmd)
	cmd.AddCommand(validatorsCmd)

	addValidatorCmd := newAddValidatorCmd()
	addNetworkFlags(addValidatorCmd)
	cmd.AddCommand(addValidatorCmd)

	removeValidatorCmd := newRemoveValidatorCmd()
	addNetworkFlags(removeValidatorCmd)
	cmd.AddCommand(removeValidatorCmd)

	// Upgrade and migration
	cmd.AddCommand(upgradecmd.NewCmd(app))

	migrateCmd := newMigrateCmd()
	addNetworkFlags(migrateCmd)
	cmd.AddCommand(migrateCmd)

	return cmd
}
