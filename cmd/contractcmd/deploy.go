// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package contractcmd

import (
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

// lux contract deploy
func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy smart contracts",
		Long: `The contract command suite provides a collection of tools for deploying
smart contracts on Lux networks.`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	// contract deploy erc20
	cmd.AddCommand(newDeployERC20Cmd())
	return cmd
}
