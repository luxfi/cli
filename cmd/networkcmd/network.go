// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"github.com/luxfi/cli/cmd/networkcmd/upgradecmd"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var (
	app         *application.Lux
	luxdVersion string
	numNodes    uint32
)

// lux network (alias: blockchain, net)
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:     "network",
		Aliases: []string{"blockchain", "net"},
		Short:   "Create and deploy blockchains/networks",
		Long: `The network command suite provides tools for developing and deploying blockchains.

In Lux, a blockchain IS a network - every blockchain can have sub-networks.

This command suite includes:
- Blockchain lifecycle: create, deploy, describe, configure
- Network management: start, stop, clean, status
- Data operations: export, import
- Validator management: add, remove, change weight

Use 'lux network', 'lux blockchain', or 'lux net' interchangeably.`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	// Blockchain lifecycle commands
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newDescribeCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newJoinCmd())
	cmd.AddCommand(newPublishCmd())
	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newConfigureCmd())
	cmd.AddCommand(vmidCmd())
	cmd.AddCommand(newConvertCmd())

	// Validator commands
	cmd.AddCommand(newAddValidatorCmd())
	cmd.AddCommand(newRemoveValidatorCmd())
	cmd.AddCommand(newValidatorsCmd())
	cmd.AddCommand(newChangeOwnerCmd())
	cmd.AddCommand(newChangeWeightCmd())

	// Data operations
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newSetHeadCmd())

	// Upgrade commands
	cmd.AddCommand(upgradecmd.NewCmd(app))

	// Local network operations
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newCleanCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}
