// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package interchaincmd

import (
	"github.com/luxfi/cli/v2/cmd/interchaincmd/messengercmd"
	"github.com/luxfi/cli/v2/cmd/interchaincmd/relayercmd"
	"github.com/luxfi/cli/v2/cmd/interchaincmd/tokentransferrercmd"
	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/luxfi/cli/v2/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux interchain
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interchain",
		Short: "Set and manage interoperability between blockchains",
		Long: `The interchain command suite provides a collection of tools to
set and manage interoperability between blockchains.`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	app = injectedApp
	// interchain tokenTransferrer
	cmd.AddCommand(tokentransferrercmd.NewCmd(app))
	// interchain relayer
	cmd.AddCommand(relayercmd.NewCmd(app))
	// interchain messenger
	cmd.AddCommand(messengercmd.NewCmd(app))
	return cmd
}
