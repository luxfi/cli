// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vmcmd provides commands for managing VM plugins.
package vmcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the vm command suite.
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Manage VM plugins",
		Long: `Commands for installing, linking, and managing VM plugins.

VM plugins are stored as symlinks in ~/.lux/plugins/<vmid>.
The VMID is calculated from the VM name (padded to 32 bytes, CB58 encoded).

Examples:
  lux vm link lux-evm --path ~/work/lux/evm/build/evm
  lux vm status
  lux vm unlink lux-evm
  lux vm reload`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newLinkCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newUnlinkCmd())
	cmd.AddCommand(newReloadCmd())

	return cmd
}
