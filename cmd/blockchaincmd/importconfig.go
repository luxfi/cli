// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

// lux blockchain import-config
func newImportConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-config",
		Short: "Import blockchain configurations into lux-cli",
		Long: `Import blockchain configurations into lux-cli.

This command suite supports importing from a file created on another computer,
or importing from blockchains running public networks
(e.g. created manually or with the deprecated subnet-cli)`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	// blockchain import file
	cmd.AddCommand(newImportFileCmd())
	// blockchain import public
	cmd.AddCommand(newImportPublicCmd())
	return cmd
}
