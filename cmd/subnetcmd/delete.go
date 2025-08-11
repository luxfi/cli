// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

// lux subnet delete
func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete a subnet configuration",
		Long:  "The subnet delete command deletes an existing subnet configuration.",
		RunE:  deleteSubnet,
		Args:  cobra.ExactArgs(1),
	}
}

func deleteSubnet(_ *cobra.Command, args []string) error {
	// Get subnet name from args
	subnetName := args[0]
	subnetDir := filepath.Join(app.GetSubnetDir(), subnetName)

	customVMPath := app.GetCustomVMPath(subnetName)

	sidecar, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}

	if sidecar.VM == models.CustomVM {
		if _, err := os.Stat(customVMPath); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			app.Log.Warn("tried to remove custom VM path but it actually does not exist. Ignoring")
			return nil
		}

		// exists
		if err := os.Remove(customVMPath); err != nil {
			return err
		}
	}

	// Note: LPM subnet VM binaries are not deleted as they may be shared
	// across multiple subnets. Manual cleanup may be required for unused binaries.
	// Track usage in: https://github.com/luxfi/cli/issues/246

	if _, err := os.Stat(subnetDir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		app.Log.Warn("tried to remove the Subnet dir path but it actually does not exist. Ignoring")
		return nil
	}

	// exists
	if err := os.RemoveAll(subnetDir); err != nil {
		return err
	}
	return nil
}
