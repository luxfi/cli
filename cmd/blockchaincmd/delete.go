// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/v2/pkg/cobrautils"
	"github.com/luxfi/cli/v2/pkg/subnet"
	"github.com/luxfi/cli/v2/sdk/utils"
	"github.com/spf13/cobra"
)

// lux blockchain delete
func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [blockchainName]",
		Short: "Delete a blockchain configuration",
		Long:  "The blockchain delete command deletes an existing blockchain configuration.",
		RunE:  deleteBlockchain,
		Args:  cobrautils.ExactArgs(1),
	}
}

func deleteBlockchain(_ *cobra.Command, args []string) error {
	return CallDeleteBlockchain(args[0])
}

func CallDeleteBlockchain(blockchainName string) error {
	if err := checkInvalidSubnetNames(blockchainName); err != nil {
		return fmt.Errorf("invalid blockchain name '%s': %w", blockchainName, err)
	}

	dataFound := false

	// rm airdrop key if exists
	airdropKeyName, _, _, err := subnet.GetDefaultSubnetAirdropKeyInfo(app, blockchainName)
	if err != nil {
		return err
	}
	if airdropKeyName != "" {
		airdropKeyPath := app.GetKeyPath(airdropKeyName)
		if utils.FileExists(airdropKeyPath) {
			dataFound = true
			if err := os.Remove(airdropKeyPath); err != nil {
				return err
			}
		}
	}

	// remove custom vm if exists
	customVMPath := app.GetCustomVMPath(blockchainName)
	if utils.FileExists(customVMPath) {
		dataFound = true
		if err := os.Remove(customVMPath); err != nil {
			return err
		}
	}

	// TODO this method does not delete the imported VM binary if this
	// is an LPM subnet. We can't naively delete the binary because it
	// may be used by multiple subnets. We should delete this binary,
	// but only if no other subnet is using it.
	// More info: https://github.com/luxfi/cli/issues/246

	// rm blockchain conf dir
	subnetDir := filepath.Join(app.GetSubnetDir(), blockchainName)
	if utils.DirExists(subnetDir) {
		return os.RemoveAll(subnetDir)
	}

	if !dataFound {
		return fmt.Errorf("blockchain %s does not exists", blockchainName)
	}

	return nil
}
