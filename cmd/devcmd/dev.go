// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package devcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

// NewCmd creates the dev command for local development
func NewCmd(_ *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Development environment commands",
		Long: `The dev command provides local development environment tools.

This runs a single-node Lux network with K=1 consensus for instant
block finality. All chains (C/P/X) are enabled with full validator
signing capabilities.

Commands:
  start   - Start local dev node (default port 8545)
  stop    - Stop the dev node

Features:
  • K=1 consensus (instant finality, no validator sampling)
  • Full validator signing for all chains
  • Compatible with Hardhat/Foundry/Anvil tooling
  • Test accounts pre-funded in genesis`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())

	return cmd
}
