// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

var (
	exportOutput        string
	customVMRepoURL     string
	customVMBranch      string
	customVMBuildScript string
)

// lux blockchain export-config
func newExportConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-config [blockchainName]",
		Short: "Export deployment configuration details",
		Long: `The blockchain export-config command writes the configuration details of an existing Blockchain deployment to a file.

The command prompts for an output path. You can also provide one with
the --output flag.`,
		RunE: exportSubnet,
		Args: cobrautils.ExactArgs(1),
	}

	cmd.Flags().StringVarP(
		&exportOutput,
		"output",
		"o",
		"",
		"write the export data to the provided file path",
	)
	cmd.Flags().StringVar(&customVMRepoURL, "custom-vm-repo-url", "", "custom vm repository url")
	cmd.Flags().StringVar(&customVMBranch, "custom-vm-branch", "", "custom vm branch")
	cmd.Flags().StringVar(&customVMBuildScript, "custom-vm-build-script", "", "custom vm build-script")
	return cmd
}

func CallExportSubnet(blockchainName, exportPath string) error {
	exportOutput = exportPath
	return exportSubnet(nil, []string{blockchainName})
}

func exportSubnet(_ *cobra.Command, args []string) error {
	var err error
	if exportOutput == "" {
		pathPrompt := "Enter file path to write export data to"
		exportOutput, err = app.Prompt.CaptureString(pathPrompt)
		if err != nil {
			return err
		}
	}

	blockchainName := args[0]

	if !app.SidecarExists(blockchainName) {
		return fmt.Errorf("invalid blockchain %q", blockchainName)
	}

	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}

	if sc.VM == models.CustomVM {
		if sc.CustomVMRepoURL == "" {
			ux.Logger.PrintToUser("Custom VM source code repository, branch and build script not defined for subnet. Filling in the details now.")
			if customVMRepoURL != "" {
				ux.Logger.PrintToUser("Checking source code repository URL %s", customVMRepoURL)
				if err := prompts.ValidateURL(customVMRepoURL, true); err != nil {
					ux.Logger.PrintToUser("Invalid repository url %s: %s", customVMRepoURL, err)
					customVMRepoURL = ""
				}
			}
			if customVMRepoURL == "" {
				customVMRepoURL, err = app.Prompt.CaptureURL("Source code repository URL")
				if err != nil {
					return err
				}
			}
			if customVMBranch != "" {
				ux.Logger.PrintToUser("Checking branch %s", customVMBranch)
				if err := prompts.ValidateRepoBranch(customVMBranch); err != nil {
					ux.Logger.PrintToUser("Invalid repository branch %s: %s", customVMBranch, err)
					customVMBranch = ""
				}
			}
			if customVMBranch == "" {
				customVMBranch, err = app.Prompt.CaptureString("Branch")
				if err != nil {
					return err
				}
			}
			if customVMBuildScript != "" {
				ux.Logger.PrintToUser("Checking build script %s", customVMBuildScript)
				if err := prompts.ValidateRepoFile(customVMBuildScript); err != nil {
					ux.Logger.PrintToUser("Invalid repository build script %s: %s", customVMBuildScript, err)
					customVMBuildScript = ""
				}
			}
			if customVMBuildScript == "" {
				customVMBuildScript, err = app.Prompt.CaptureString("Build script")
				if err != nil {
					return err
				}
			}
			sc.CustomVMRepoURL = customVMRepoURL
			sc.CustomVMBranch = customVMBranch
			sc.CustomVMBuildScript = customVMBuildScript
			if err := app.UpdateSidecar(&sc); err != nil {
				return err
			}
		}
	}

	gen, err := app.LoadRawGenesis(blockchainName)
	if err != nil {
		return err
	}

	// Node configuration and chain configs are handled separately from the export
	// These are managed through the deployment configuration
	// var chainConfig, subnetConfig, networkUpgrades []byte
	// var nodeConfig []byte
	// if app.LuxdNodeConfigExists(blockchainName) {
	// 	nodeConfig, err = app.LoadRawLuxdNodeConfig(blockchainName)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if app.ChainConfigExists(blockchainName) {
	// 	chainConfig, err = app.LoadRawChainConfig(blockchainName)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if app.LuxdSubnetConfigExists(blockchainName) {
	// 	subnetConfig, err = app.LoadRawLuxdSubnetConfig(blockchainName)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if app.NetworkUpgradeExists(blockchainName) {
	// 	networkUpgrades, err = app.LoadRawNetworkUpgrades(blockchainName)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// The Exportable struct contains the essential configuration
	// Additional configs are handled through the deployment process
	exportData := models.Exportable{
		Sidecar: sc,
		Genesis: gen,
	}
	// Additional configs would need to be handled separately:
	// chainConfig, subnetConfig, networkUpgrades

	exportBytes, err := json.Marshal(exportData)
	if err != nil {
		return err
	}
	return os.WriteFile(exportOutput, exportBytes, constants.WriteReadReadPerms)
}
