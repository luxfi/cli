// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package upgradecmd

import (
	"bytes"
	"encoding/json"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// lux subnet upgrade import
func newUpgradePrintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print [subnetName]",
		Short: "Print the upgrade.json file content",
		Long:  `Print the upgrade.json file content`,
		RunE:  upgradePrintCmd,
		Args:  cobra.ExactArgs(1),
	}

	return cmd
}

func upgradePrintCmd(_ *cobra.Command, args []string) error {
	subnetName := args[0]
	if !app.GenesisExists(subnetName) {
		ux.Logger.PrintToUser("The provided subnet name %q does not exist", subnetName)
		return nil
	}

	fileBytes, err := app.ReadUpgradeFile(subnetName)
	if err != nil {
		return err
	}

	var prettyJSON bytes.Buffer
	if err = json.Indent(&prettyJSON, fileBytes, "", "  "); err != nil {
		return err
	}
	ux.Logger.PrintToUser("%s", prettyJSON.String())
	return nil
}
