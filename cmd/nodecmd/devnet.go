// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"github.com/luxfi/cli/pkg/cobrautils"

	"github.com/spf13/cobra"
)

func newDevnetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devnet",
		Short: "(ALPHA Warning) Suite of commands for a devnet cluster",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node devnet command suite provides a collection of commands related to devnets.
You can check the updated status by calling lux node status <clusterName>`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	// node devnet deploy
	cmd.AddCommand(newDeployCmd())
	// node devnet wiz
	cmd.AddCommand(newWizCmd())
	return cmd
}
