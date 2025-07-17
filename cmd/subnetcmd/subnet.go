// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"fmt"

	"github.com/luxfi/cli/cmd/subnetcmd/upgradecmd"
	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux l2 (alias: subnet for backward compatibility)
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "l2",
		Aliases: []string{"subnet"},
		Short:   "Create and deploy L2s",
		Long: `The l2 command suite provides tools for creating and deploying L2s.

L2s (formerly subnets) support multiple sequencing models:
- Lux: Based rollup, 100ms blocks, lowest cost
- Ethereum: Based rollup, 12s blocks, highest security  
- Lux: Based rollup, 2s blocks, fast finality
- OP: OP Stack compatible for Optimism ecosystem
- External: Traditional centralized sequencer

Features:
- EIP-4844 blob support for data availability
- Pre-confirmations for <100ms transaction acknowledgment
- IBC/Teleport for cross-chain messaging
- Ringtail post-quantum signatures

To get started, use 'lux l2 create' to configure your L2.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	app = injectedApp
	// subnet create
	cmd.AddCommand(newCreateCmd())
	// subnet delete
	cmd.AddCommand(newDeleteCmd())
	// subnet deploy
	cmd.AddCommand(newDeployCmd())
	// subnet describe
	cmd.AddCommand(newDescribeCmd())
	// subnet list
	cmd.AddCommand(newListCmd())
	// subnet join
	cmd.AddCommand(newJoinCmd())
	// subnet addValidator
	cmd.AddCommand(newAddValidatorCmd())
	// subnet export
	cmd.AddCommand(newExportCmd())
	// subnet import
	cmd.AddCommand(newImportCmd())
	// subnet publish
	cmd.AddCommand(newPublishCmd())
	// subnet upgrade
	cmd.AddCommand(upgradecmd.NewCmd(app))
	// subnet stats
	cmd.AddCommand(newStatsCmd())
	// subnet configure
	cmd.AddCommand(newConfigureCmd())
	// subnet import-running
	cmd.AddCommand(newImportFromNetworkCmd())
	// subnet import-historic
	cmd.AddCommand(newImportHistoricCmd())
	// subnet VMID
	cmd.AddCommand(vmidCmd())
	// subnet removeValidator
	cmd.AddCommand(newRemoveValidatorCmd())
	// subnet elastic
	cmd.AddCommand(newElasticCmd())
	// subnet validators
	cmd.AddCommand(newValidatorsCmd())
	// subnet migrate-base
	cmd.AddCommand(newMigrateBaseCmd())
	return cmd
}
