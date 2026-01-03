// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package upgradecmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var upgradeBytesFilePath string

const upgradeBytesFilePathKey = "upgrade-filepath"

// lux blockchain upgrade import
func newUpgradeImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [blockchainName]",
		Short: "Import the upgrade bytes file into the local environment",
		Long:  `Import the upgrade bytes file into the local environment`,
		RunE:  upgradeImportCmd,
		Args:  cobrautils.ExactArgs(1),
	}

	cmd.Flags().StringVar(&upgradeBytesFilePath, upgradeBytesFilePathKey, "", "Import upgrade bytes file into local environment")

	return cmd
}

func upgradeImportCmd(_ *cobra.Command, args []string) error {
	blockchainName := args[0]
	if !app.GenesisExists(blockchainName) {
		ux.Logger.PrintToUser("The provided blockchain name %q does not exist", blockchainName)
		return nil
	}

	if upgradeBytesFilePath == "" {
		if !prompts.IsInteractive() {
			return fmt.Errorf("--%s is required in non-interactive mode", upgradeBytesFilePathKey)
		}
		var err error
		upgradeBytesFilePath, err = app.Prompt.CaptureExistingFilepath("Provide the path to the upgrade file to import")
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(upgradeBytesFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("the upgrade file specified with path %q does not exist", upgradeBytesFilePath)
		}
		return err
	}

	upgradeBytes, err := os.ReadFile(upgradeBytesFilePath) //nolint:gosec // G304: User-specified upgrade file
	if err != nil {
		return fmt.Errorf("failed to read the provided upgrade file: %w", err)
	}

	return writeUpgrade(blockchainName, upgradeBytes)
}
