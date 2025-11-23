// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

// lux blockchain import
func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import blockchains into lux-cli",
		Long: `Import blockchain configurations into lux-cli.

This command suite supports importing from a file created on another computer,
or importing from blockchains running public networks
(e.g. created manually or with the deprecated subnet-cli)`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	// blockchain import data
	cmd.AddCommand(newImportDataCmd())
	// blockchain import file
	cmd.AddCommand(newImportFileCmd())
	// blockchain import public
	cmd.AddCommand(newImportPublicCmd())
	return cmd
}
