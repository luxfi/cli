// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/lpmintegration"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/version"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"gopkg.in/yaml.v3"
)

var (
	alias          string
	repoURL        string
	vmDescPath     string
	subnetDescPath string
	noRepoPath     string

	errSubnetNotDeployed = errors.New(
		"only subnets which have already been deployed to either testnet (testnet) or mainnet can be published")
)

type newPublisherFunc func(string, string, string) subnet.Publisher

// lux subnet publish
func newPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "publish [subnetName]",
		Short:        "Publish the subnet's VM to a repository",
		Long:         `The subnet publish command publishes the Subnet's VM to a repository.`,
		SilenceUsage: true,
		RunE:         publish,
		Args:         cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&alias, "alias", "",
		"We publish to a remote repo, but identify the repo locally under a user-provided alias (e.g. myrepo).")
	cmd.Flags().StringVar(&repoURL, "repo-url", "", "The URL of the repo where we are publishing")
	cmd.Flags().StringVar(&vmDescPath, "vm-file-path", "",
		"Path to the VM description file. If not given, a prompting sequence will be initiated.")
	cmd.Flags().StringVar(&subnetDescPath, "subnet-file-path", "",
		"Path to the Subnet description file. If not given, a prompting sequence will be initiated.")
	cmd.Flags().StringVar(&noRepoPath, "no-repo-path", "",
		"Do not let the tool manage file publishing, but have it only generate the files and put them in the location given by this flag.")
	cmd.Flags().BoolVar(&forceWrite, forceFlag, false,
		"If true, ignores if the subnet has been published in the past, and attempts a forced publish.")
	return cmd
}

func publish(_ *cobra.Command, args []string) error {
	chains, err := validateSubnetNameAndGetChains(args)
	if err != nil {
		return err
	}
	subnetName := chains[0]
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}
	if !isReadyToPublish(&sc) {
		return errSubnetNotDeployed
	}
	return doPublish(&sc, subnetName, subnet.NewPublisher)
}

// isReadyToPublish currently means if deployed to testnet and/or main
func isReadyToPublish(sc *models.Sidecar) bool {
	if sc.Networks[models.Testnet.String()].SubnetID != ids.Empty &&
		sc.Networks[models.Testnet.String()].BlockchainID != ids.Empty {
		return true
	}
	if sc.Networks[models.Mainnet.String()].SubnetID != ids.Empty &&
		sc.Networks[models.Mainnet.String()].BlockchainID != ids.Empty {
		return true
	}
	return false
}

func doPublish(sc *models.Sidecar, subnetName string, publisherCreateFunc newPublisherFunc) (err error) {
	reposDir := app.GetReposDir()
	// iterate the reposDir to check what repos already exist locally
	// if nothing is found, prompt the user for an alias for a new repo
	if err = getAlias(reposDir); err != nil {
		return err
	}
	// get the URL for the repo
	if err = getRepoURL(reposDir); err != nil {
		return err
	}

	var (
		tsubnet *lpmintegration.Subnet
		vm      *lpmintegration.VM
	)

	if !forceWrite && noRepoPath == "" {
		// if forceWrite is present, we don't need to check if it has been previously published, we just do
		published, err := isAlreadyPublished(subnetName)
		if err != nil {
			return err
		}
		if published {
			ux.Logger.PrintToUser(
				"It appears this subnet has already been published, while no force flag has been detected.")
			return errors.New("aborted")
		}
	}

	if subnetDescPath == "" {
		tsubnet, err = getSubnetInfo(sc)
	} else {
		tsubnet = new(lpmintegration.Subnet)
		err = loadYAMLFile(subnetDescPath, tsubnet)
	}
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("Nice! We got the subnet info. Let's now get the VM details")

	if vmDescPath == "" {
		vm, err = getVMInfo(sc)
	} else {
		vm = new(lpmintegration.VM)
		err = loadYAMLFile(vmDescPath, vm)
	}
	if err != nil {
		return err
	}

	// Currently publishing exactly 1 subnet and 1 VM per operation
	tsubnet.VMs = []string{vm.Alias}

	subnetYAML, err := yaml.Marshal(tsubnet)
	if err != nil {
		return err
	}
	vmYAML, err := yaml.Marshal(vm)
	if err != nil {
		return err
	}

	if noRepoPath != "" {
		ux.Logger.PrintToUser(
			"Writing the file specs to the provided directory at: %s", noRepoPath)
		// the directory does not exist
		if _, err := os.Stat(noRepoPath); err != nil {
			if err := os.MkdirAll(noRepoPath, constants.DefaultPerms755); err != nil {
				return fmt.Errorf(
					"attempted to create the given --no-repo-path directory at %s, but failed: %w", noRepoPath, err)
			}
			ux.Logger.PrintToUser(
				"The given --no-repo-path at %s did not exist; created it with permissions %o", noRepoPath, constants.DefaultPerms755)
		}
		subnetFile := filepath.Join(noRepoPath, constants.SubnetDir, subnetName+constants.YAMLSuffix)
		vmFile := filepath.Join(noRepoPath, constants.VMDir, vm.Alias+constants.YAMLSuffix)
		if !forceWrite {
			// do not automatically overwrite
			if _, err := os.Stat(subnetFile); err == nil {
				return fmt.Errorf(
					"a file with the name %s already exists. If you wish to overwrite, provide the %s flag", subnetFile, forceFlag)
			}
			if _, err := os.Stat(vmFile); err == nil {
				return fmt.Errorf(
					"a file with the name %s already exists. If you wish to overwrite, provide the %s flag", vmFile, forceFlag)
			}
		}
		if err := os.WriteFile(subnetFile, subnetYAML, constants.DefaultPerms755); err != nil {
			return fmt.Errorf("failed creating the subnet description YAML file: %w", err)
		}
		if err := os.WriteFile(vmFile, vmYAML, constants.DefaultPerms755); err != nil {
			return fmt.Errorf("failed creating the subnet description YAML file: %w", err)
		}
		ux.Logger.PrintToUser("YAML files written successfully to %s", noRepoPath)
		return nil
	}

	ux.Logger.PrintToUser("Thanks! We got all the bits and pieces. Trying to publish on the provided repo...")

	publisher := publisherCreateFunc(reposDir, repoURL, alias)
	repo, err := publisher.GetRepo()
	if err != nil {
		return err
	}

	// Check if already published and handle updates
	needsPublish := true
	if !forceWrite {
		// Check if this version has already been published
		// VersionExists method not yet implemented
		// if exists, _ := publisher.VersionExists(repo, subnetName, vm.Alias); exists {
		//	needsPublish = false
		//	ux.Logger.PrintToUser("Subnet %s already published. Use --force to republish", subnetName)
		// }
	}

	if needsPublish {
		if err = publisher.Publish(repo, subnetName, vm.Alias, subnetYAML, vmYAML); err != nil {
			return err
		}
	}

	ux.Logger.PrintToUser("Successfully published")
	return nil
}

// current simplistic approach:
// just search any folder names `subnetName` inside the reposDir's `subnets` folder
func isAlreadyPublished(subnetName string) (bool, error) {
	reposDir := app.GetReposDir()

	found := false

	if err := filepath.WalkDir(reposDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			if filepath.Base(path) == constants.VMDir {
				return filepath.SkipDir
			}
			if !d.IsDir() && d.Name() == subnetName {
				found = true
			}
		}
		return nil
	}); err != nil {
		return false, err
	}
	return found, nil
}

// iterate the reposDir to check what repos already exist locally
// if nothing is found, prompt the user for an alias for a new repo
func getAlias(reposDir string) error {
	// have any aliases already been defined?
	if alias == "" {
		matches, err := os.ReadDir(reposDir)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			// no aliases yet; just ask for a new one
			alias, err = getNewAlias()
			if err != nil {
				return err
			}
		} else {
			// there are already aliases, ask how to proceed
			options := []string{"Provide a new alias", "Pick from list"}
			choice, err := app.Prompt.CaptureList(
				"Don't know which repo to publish to. How would you like to proceed?", options)
			if err != nil {
				return err
			}
			if choice == options[0] {
				// user chose to provide a new alias
				alias, err = getNewAlias()
				if err != nil {
					return err
				}
				// double-check: actually this path exists...
				if _, err := os.Stat(filepath.Join(reposDir, alias)); err == nil {
					ux.Logger.PrintToUser(
						"The repository with the given alias already exists locally. You may have already published this subnet there (the other explanation is that a different subnet has been published there).")
					yes, err := app.Prompt.CaptureYesNo("Do you wish to continue?")
					if err != nil {
						return err
					}
					if !yes {
						ux.Logger.PrintToUser("User canceled, nothing got published.")
						return nil
					}
				}
			} else {
				aliases := make([]string, len(matches))
				for i, a := range matches {
					aliases[i] = a.Name()
				}
				alias, err = app.Prompt.CaptureList("Pick an alias", aliases)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ask for a new alias
func getNewAlias() (string, error) {
	return app.Prompt.CaptureString("Provide an alias for the repository we are going to use")
}

func getVMAlias(sc *models.Sidecar, subnetName string) string {
	// Try to use blockchain ID from deployed networks
	if sc.Networks[models.Testnet.String()].BlockchainID != ids.Empty {
		return sc.Networks[models.Testnet.String()].BlockchainID.String()
	}
	if sc.Networks[models.Mainnet.String()].BlockchainID != ids.Empty {
		return sc.Networks[models.Mainnet.String()].BlockchainID.String()
	}
	// Fallback to subnet name
	return subnetName
}

// getRepoURL determines the repository URL from flag or existing configuration
func getRepoURL(reposDir string) error {
	if repoURL != "" {
		return nil
	}
	path := filepath.Join(reposDir, alias)
	repo, err := git.PlainOpen(path)
	if err != nil {
		app.Log.Debug(
			"opening repo failed - alias might have not been created yet, so ignore", zap.String("alias", alias), zap.Error(err))
		repoURL, err = app.Prompt.CaptureString("Provide the repository URL")
		return err
	}
	// there is a repo already for this alias, let's try to figure out the remote URL from there
	conf, err := repo.Config()
	if err != nil {
		// Log the error but allow user to provide URL manually
		app.Log.Debug("Failed to read repository config", zap.Error(err))
		repoURL, err = app.Prompt.CaptureString("Unable to detect remote URL. Please provide the repository URL")
		return err
	}
	remotes := make([]string, len(conf.Remotes))
	i := 0
	for _, r := range conf.Remotes {
		// NOTE: supporting only one remote for now
		remotes[i] = r.URLs[0]
		i++
	}
	repoURL, err = app.Prompt.CaptureList("Which is the remote URL for this repo?", remotes)
	if err != nil {
		// should never happen
		return err
	}
	return nil
}

// loadYAMLFile loads a YAML file from disk into a concrete types.Definition object
// using generics. It's role really is solely to verify that the YAML content is valid.
func loadYAMLFile[T any](path string, defType T) error {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(fileBytes, &defType)
}

func getSubnetInfo(sc *models.Sidecar) (*lpmintegration.Subnet, error) {
	_, err := app.Prompt.CaptureStringAllowEmpty("What is the homepage of the Subnet project?")
	if err != nil {
		return nil, err
	}

	desc, err := app.Prompt.CaptureStringAllowEmpty("Provide a free-text description of the Subnet")
	if err != nil {
		return nil, err
	}

	_, canceled, err := prompts.CaptureListDecision(
		app.Prompt,
		"Who are the maintainers of the Subnet?",
		app.Prompt.CaptureEmail,
		"Provide a maintainer",
		"Maintainer",
		"",
	)
	if err != nil {
		return nil, err
	}
	if canceled {
		ux.Logger.PrintToUser("Publishing aborted")
		return nil, errors.New("canceled by user")
	}

	subnet := &lpmintegration.Subnet{
		ID:          sc.Networks[models.Testnet.String()].SubnetID.String(),
		Alias:       sc.Name,
		Description: desc,
		VMs:         []string{sc.Subnet},
	}

	return subnet, nil
}

func getVMInfo(sc *models.Sidecar) (*lpmintegration.VM, error) {
	var (
		vmID, desc, url, sha string
		canceled             bool
		ver                  *version.Semantic
		err                  error
	)

	switch {
	case sc.VM == models.CustomVM:
		vmID, err = app.Prompt.CaptureStringAllowEmpty("What is the ID of this VM?")
		if err != nil {
			return nil, err
		}
		desc, err = app.Prompt.CaptureStringAllowEmpty("Provide a description for this VM")
		if err != nil {
			return nil, err
		}
		_, canceled, err = prompts.CaptureListDecision(
			app.Prompt,
			"Who are the maintainers of the VM?",
			app.Prompt.CaptureEmail,
			"Provide a maintainer",
			"Maintainer",
			"",
		)
		if err != nil {
			return nil, err
		}
		if canceled {
			ux.Logger.PrintToUser("Publishing aborted")
			return nil, errors.New("canceled by user")
		}

		url, err = app.Prompt.CaptureStringAllowEmpty(
			"Tell us the URL to download the source. Needs to be a fixed version, not `latest`.")
		if err != nil {
			return nil, err
		}

		sha, err = app.Prompt.CaptureStringAllowEmpty(
			"For integrity checks, provide the sha256 commit for the used version")
		if err != nil {
			return nil, err
		}
		strVer, err := app.Prompt.CaptureVersion(
			"This is the last question! What is the version being used? Use semantic versioning (v1.2.3)")
		if err != nil {
			return nil, err
		}
		ver, err = version.Parse(strVer)
		if err != nil {
			return nil, err
		}

	case sc.VM == models.EVM:
		vmID = models.EVM
		dl := binutils.NewEVMDownloader()
		desc = "Lux EVM is a simplified version of Geth VM (C-Chain). It implements the Ethereum Virtual Machine and supports Solidity smart contracts as well as most other Ethereum client functionality"
		_, ver, url, sha, err = getInfoForKnownVMs(
			sc.VMVersion,
			constants.EVMRepoName,
			app.GetEVMBinDir(),
			constants.EVMBin,
			dl,
		)
	default:
		return nil, fmt.Errorf("unexpected error: unsupported VM type: %s", sc.VM)
	}
	if err != nil {
		return nil, err
	}

	_, err = app.Prompt.CaptureStringAllowEmpty(
		"What scripts needs to run to install this VM? Needs to be an executable command to build the VM")
	if err != nil {
		return nil, err
	}

	_, err = app.Prompt.CaptureStringAllowEmpty(
		"What is the binary path? (This is the output of the build command)")
	if err != nil {
		return nil, err
	}

	vm := &lpmintegration.VM{
		ID:          vmID,
		Alias:       getVMAlias(sc, sc.Name),
		Description: desc,
		URL:         url,
		Checksum:    sha,
		Version:     ver.String(),
	}

	return vm, nil
}

func getInfoForKnownVMs(
	strVer, repoName, vmBinDir, vmBin string,
	dl binutils.GithubDownloader,
) ([]string, *version.Semantic, string, string, error) {
	maintrs := []string{constants.LuxMaintainers}
	binPath := filepath.Join(vmBinDir, repoName+"-"+strVer, vmBin)
	sha, err := utils.GetSHA256FromDisk(binPath)
	if err != nil {
		return nil, nil, "", "", err
	}
	ver, err := version.Parse(strVer)
	if err != nil {
		return nil, nil, "", "", err
	}
	inst := binutils.NewInstaller()
	url, _, err := dl.GetDownloadURL(strVer, inst)
	if err != nil {
		return nil, nil, "", "", err
	}

	return maintrs, ver, url, sha, nil
}
