// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gpucmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

// NewCmd creates the gpu command and its subcommands.
func NewCmd(_ *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gpu",
		Short: "Manage GPU acceleration",
		Long: `The gpu command provides utilities for managing GPU acceleration
in the Lux node. Use subcommands to check GPU status, availability,
and configuration.

GPU acceleration is used for:
  - NTT operations in Ringtail consensus
  - FHE operations in ThresholdVM
  - Lattice cryptography operations`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	cmd.AddCommand(newStatusCmd())
	return cmd
}
