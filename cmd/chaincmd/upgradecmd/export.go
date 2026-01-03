// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package upgradecmd

import (
	"os"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var force bool

// lux blockchain upgrade export
func newUpgradeExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [blockchainName]",
		Short: "Export the upgrade bytes file to a location of choice on disk",
		Long: `Export the upgrade bytes file to a location of choice on disk.

In non-interactive mode (CI/scripts), use --output to specify the file path
and --force to overwrite existing files without confirmation.

Examples:
  # Interactive mode (prompts for path)
  lux blockchain upgrade export mychain

  # Non-interactive mode
  lux blockchain upgrade export mychain --output ./upgrade.json --force`,
		RunE: upgradeExportCmd,
		Args: cobrautils.ExactArgs(1),
	}

	cmd.Flags().StringVarP(&upgradeBytesFilePath, "output", "o", "", "Output file path for upgrade bytes (required in non-interactive mode)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing file without confirmation")

	return cmd
}

func upgradeExportCmd(_ *cobra.Command, args []string) error {
	blockchainName := args[0]
	if !app.GenesisExists(blockchainName) {
		ux.Logger.PrintToUser("The provided blockchain name %q does not exist", blockchainName)
		return nil
	}

	// Use Validator pattern for missing output path
	v := prompts.NewValidator("lux blockchain upgrade export")
	v.Require(&upgradeBytesFilePath, prompts.MissingOpt{
		Flag:   "--output",
		Prompt: "Output file path",
		Note:   "path where upgrade bytes will be exported",
	})
	if err := v.Resolve(func(m prompts.MissingOpt) (string, error) {
		return app.Prompt.CaptureString(m.Prompt)
	}); err != nil {
		return err
	}

	// Check if file exists and handle overwrite
	if _, err := os.Stat(upgradeBytesFilePath); err == nil {
		if !force {
			if !prompts.IsInteractive() {
				ux.Logger.PrintToUser("File %q already exists. Use --force to overwrite.", upgradeBytesFilePath)
				return nil
			}
			ux.Logger.PrintToUser("The file specified with path %q already exists!", upgradeBytesFilePath)
			yes, err := app.Prompt.CaptureYesNo("Should we overwrite it?")
			if err != nil {
				return err
			}
			if !yes {
				ux.Logger.PrintToUser("Aborted by user. Nothing has been exported")
				return nil
			}
		}
	}

	fileBytes, err := app.ReadUpgradeFile(blockchainName)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("Writing the upgrade bytes file to %q...", upgradeBytesFilePath)
	err = os.WriteFile(upgradeBytesFilePath, fileBytes, constants.DefaultPerms755)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("File written successfully.")
	return nil
}
