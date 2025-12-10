// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/lpmintegration"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

var (
	overwriteImport bool
	repoOrURL       string
	subnetAlias     string
	branch          string
)

// lux network import config
func newImportConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [path]",
		Short: "Import blockchain configuration from file or repository",
		Long: `Import a blockchain configuration from a file or a git repository.

USAGE:
  lux network import config /path/to/config.json     # From file
  lux network import config --repo=luxfi/plugins-core --blockchain=mychain

OPTIONS:
  --force        Overwrite existing configuration
  --repo         Repository to import from (e.g., luxfi/plugins-core)
  --branch       Repository branch to use
  --blockchain   Blockchain configuration to import from repo

EXAMPLES:
  # Import from exported config file
  lux network import config ~/exports/mychain.json

  # Import from repository
  lux network import config --repo=luxfi/plugins-core --blockchain=defi-chain`,
		RunE: importConfigFunc,
		Args: cobrautils.MaximumNArgs(1),
	}

	cmd.Flags().BoolVarP(&overwriteImport, "force", "f", false, "Overwrite existing configuration")
	cmd.Flags().StringVar(&repoOrURL, "repo", "", "Repository to import from (e.g., luxfi/plugins-core)")
	cmd.Flags().StringVar(&branch, "branch", "", "Repository branch to use")
	cmd.Flags().StringVar(&subnetAlias, "blockchain", "", "Blockchain configuration to import from repo")

	return cmd
}

func importConfigFunc(_ *cobra.Command, args []string) error {
	// If path provided as argument, use that
	if len(args) == 1 {
		return importFromFile(args[0])
	}

	// If repo flags provided, use LPM
	if repoOrURL != "" || branch != "" || subnetAlias != "" {
		return importFromLPM()
	}

	// Interactive mode
	fileOption := "File"
	lpmOption := "Repository"
	typeOptions := []string{fileOption, lpmOption}
	promptStr := "Would you like to import your blockchain from a file or a repository?"
	result, err := app.Prompt.CaptureList(promptStr, typeOptions)
	if err != nil {
		return err
	}

	if result == fileOption {
		return importFromFile("")
	}

	return importFromLPM()
}

func importFromFile(importPath string) error {
	var err error
	if importPath == "" {
		promptStr := "Select the file to import your blockchain from"
		importPath, err = app.Prompt.CaptureExistingFilepath(promptStr)
		if err != nil {
			return err
		}
	}

	importFileBytes, err := os.ReadFile(importPath)
	if err != nil {
		return err
	}

	importable := models.Exportable{}
	err = json.Unmarshal(importFileBytes, &importable)
	if err != nil {
		return err
	}

	blockchainName := importable.Sidecar.Name
	if blockchainName == "" {
		return errors.New("export data is malformed: missing blockchain name")
	}

	if app.GenesisExists(blockchainName) && !overwriteImport {
		return errors.New("blockchain already exists. Use --force parameter to overwrite")
	}

	if importable.Sidecar.VM == models.CustomVM {
		if importable.Sidecar.CustomVMRepoURL == "" {
			return fmt.Errorf("repository url must be defined for custom vm import")
		}
		if importable.Sidecar.CustomVMBranch == "" {
			return fmt.Errorf("repository branch must be defined for custom vm import")
		}
		if importable.Sidecar.CustomVMBuildScript == "" {
			return fmt.Errorf("build script must be defined for custom vm import")
		}

		vmPath := app.GetCustomVMPath(blockchainName)
		rpcVersion, err := vm.GetVMBinaryProtocolVersion(vmPath)
		if err != nil {
			return fmt.Errorf("unable to get custom binary RPC version: %w", err)
		}
		if rpcVersion != importable.Sidecar.RPCVersion {
			return fmt.Errorf("RPC version mismatch between sidecar and vm binary (%d vs %d)", importable.Sidecar.RPCVersion, rpcVersion)
		}
	}

	if err := app.WriteGenesisFile(blockchainName, importable.Genesis); err != nil {
		return err
	}

	_ = os.RemoveAll(app.GetLuxdNodeConfigPath(blockchainName))
	_ = os.RemoveAll(app.GetChainConfigPath(blockchainName))
	_ = os.RemoveAll(app.GetLuxdSubnetConfigPath(blockchainName))
	_ = os.RemoveAll(app.GetUpgradeBytesFilepath(blockchainName))

	if err := app.CreateSidecar(&importable.Sidecar); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Blockchain imported successfully")

	return nil
}

func importFromLPM() error {
	usr, err := user.Current()
	if err != nil {
		return err
	}
	lpmBaseDir := filepath.Join(usr.HomeDir, constants.LPMDir)
	if err = lpmintegration.SetupLpm(app, lpmBaseDir); err != nil {
		return err
	}
	installedRepos, err := lpmintegration.GetRepos(app)
	if err != nil {
		return err
	}

	var repoAlias string
	var repoURL *url.URL
	customRepo := "Download new repo"

	if repoOrURL != "" {
		for _, installedRepo := range installedRepos {
			if repoOrURL == installedRepo {
				repoAlias = installedRepo
				break
			}
		}
		if repoAlias == "" {
			repoAlias = customRepo
			repoURL, err = url.ParseRequestURI(repoOrURL)
			if err != nil {
				return fmt.Errorf("invalid url in flag: %w", err)
			}
		}
	}

	if repoAlias == "" {
		installedRepos = append(installedRepos, customRepo)

		promptStr := "What repo would you like to import from"
		repoAlias, err = app.Prompt.CaptureList(promptStr, installedRepos)
		if err != nil {
			return err
		}
	}

	if repoAlias == customRepo {
		if repoURL == nil {
			promptStr := "Enter your repo URL"
			repoURL, err = app.Prompt.CaptureGitURL(promptStr)
			if err != nil {
				return err
			}
		}

		if branch == "" {
			mainBranch := "main"
			masterBranch := "master"
			customBranch := "custom"
			branchList := []string{mainBranch, masterBranch, customBranch}
			promptStr := "What branch would you like to import from"
			branch, err = app.Prompt.CaptureList(promptStr, branchList)
			if err != nil {
				return err
			}
		}

		repoAlias, err = lpmintegration.AddRepo(app, repoURL, branch)
		if err != nil {
			return err
		}

		err = lpmintegration.UpdateRepos(app)
		if err != nil {
			return err
		}
	}

	subnets, err := lpmintegration.GetSubnets(app, repoAlias)
	if err != nil {
		return err
	}

	var subnet string
	if subnetAlias != "" {
		for _, availableSubnet := range subnets {
			if subnetAlias == availableSubnet {
				subnet = subnetAlias
				break
			}
		}
		if subnet == "" {
			return fmt.Errorf("unable to find blockchain %s", subnetAlias)
		}
	} else {
		promptStr := "Select a blockchain to import"
		subnet, err = app.Prompt.CaptureList(promptStr, subnets)
		if err != nil {
			return err
		}
	}

	subnetKey := lpmintegration.MakeKey(repoAlias, subnet)

	subnetDescr, err := lpmintegration.LoadSubnetFile(app, subnetKey)
	if err != nil {
		return err
	}

	var vmType models.VMType = models.CustomVM

	if len(subnetDescr.VMs) == 0 {
		return errors.New("no vms found in the given blockchain")
	}

	vmDescr, err := lpmintegration.LoadVMFile(app, repoAlias, subnetDescr.VMs[0])
	if err != nil {
		return err
	}

	rpcVersion := 0

	sidecar := models.Sidecar{
		Name:            subnetDescr.Alias,
		VM:              vmType,
		RPCVersion:      rpcVersion,
		Subnet:          subnetDescr.Alias,
		TokenName:       constants.DefaultTokenName,
		TokenSymbol:     "TEST",
		Version:         constants.SidecarVersion,
		ImportedFromLPM: true,
		ImportedVMID:    vmDescr.ID,
	}

	ux.Logger.PrintToUser("Selected blockchain, installing %s", subnetKey)

	if err = lpmintegration.InstallVM(app, subnetKey); err != nil {
		return err
	}

	err = app.CreateSidecar(&sidecar)
	if err != nil {
		return err
	}

	return app.WriteGenesisFile(subnetDescr.Alias, []byte{})
}
