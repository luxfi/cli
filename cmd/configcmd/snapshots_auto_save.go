// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package configcmd

import (
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/spf13/cobra"
)

// lux config snapshotsAutoSave command
func newSnapshotsAutoSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshotsAutoSave [enable | disable]",
		Short: "opt in or out of auto saving local network snapshots",
		Long:  "set user preference between auto saving local network snapshots or not",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleBooleanSetting(cmd, constants.ConfigSnapshotsAutoSaveKey, args)
		},
		Args: cobrautils.MaximumNArgs(1),
	}

	return cmd
}
