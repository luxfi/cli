// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"github.com/spf13/cobra"
)

// lux network import
func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import blockchain configuration",
		Long: `Import blockchain configuration from file, repository, or running network.

For importing chain data (blocks) from RLP files, use 'lux chain import'.`,
	}

	cmd.AddCommand(newImportConfigCmd())
	cmd.AddCommand(newImportPublicCmd())
	return cmd
}
