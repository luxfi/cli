// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package dependencies

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/ux"

	"golang.org/x/mod/semver"

	"github.com/luxfi/sdk/models"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

var ErrNoLuxdVersion = errors.New("unable to find a compatible luxd version")

// DefaultCLIDependencyMap provides fallback version info when remote fetch fails
var DefaultCLIDependencyMap = models.CLIDependencyMap{
	RPC: 42,
	Luxd: map[string]models.NetworkVersions{
		"Mainnet": {
			LatestVersion:  "v1.21.0",
			MinimumVersion: "v1.20.0",
		},
		"Testnet": {
			LatestVersion:  "v1.21.0",
			MinimumVersion: "v1.20.0",
		},
		"Local Network": {
			LatestVersion:  "v1.21.0",
			MinimumVersion: "v1.20.0",
		},
	},
	SubnetEVM: "v0.6.12",
}

func GetLatestLuxdByProtocolVersion(app *application.Lux, rpcVersion int) (string, error) {
	useVersion, err := GetAvailableLuxdVersions(app, rpcVersion, constants.LuxdCompatibilityURL)
	if err != nil {
		return "", err
	}
	return useVersion[0], nil
}

func GetLatestCLISupportedDependencyVersion(app *application.Lux, dependencyName string, network models.Network, rpcVersion *int) (string, error) {
	var parsedDependency models.CLIDependencyMap

	// Try to load from remote URL first
	dependencyBytes, err := app.Downloader.Download(constants.CLILatestDependencyURL)
	if err != nil {
		// Try to load from local min-version.json file (in CLI repo or executable directory)
		localPath := findLocalMinVersionFile()
		if localPath != "" {
			dependencyBytes, err = os.ReadFile(localPath)
		}
		if err != nil || localPath == "" {
			// Fall back to embedded default
			ux.Logger.PrintToUser("Using embedded dependency versions (remote fetch failed)")
			parsedDependency = DefaultCLIDependencyMap
		}
	}

	// Only parse if we don't have the default already set
	if parsedDependency.RPC == 0 && dependencyBytes != nil {
		if err = json.Unmarshal(dependencyBytes, &parsedDependency); err != nil {
			// Fall back to embedded default on parse error
			ux.Logger.PrintToUser("Using embedded dependency versions (parse error)")
			parsedDependency = DefaultCLIDependencyMap
		}
	}

	switch dependencyName {
	case constants.LuxdRepoName:
		// if the user is using RPC that is lower than the latest RPC supported by CLI, user will get latest Luxd version for that RPC
		// based on "https://raw.githubusercontent.com/luxfi/luxd/master/version/compatibility.json"
		if rpcVersion != nil && parsedDependency.RPC > *rpcVersion {
			return GetLatestLuxdByProtocolVersion(
				app,
				*rpcVersion,
			)
		}
		return parsedDependency.Luxd[network.Name()].LatestVersion, nil
	case constants.SubnetEVMRepoName:
		return parsedDependency.SubnetEVM, nil
	default:
		return "", fmt.Errorf("unsupported dependency: %s", dependencyName)
	}
}

// findLocalMinVersionFile searches for min-version.json in common locations
func findLocalMinVersionFile() string {
	// Get executable directory
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		// Check in same directory as executable
		path := filepath.Join(execDir, "..", "min-version.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// Check in parent directory (for bin/lux -> min-version.json)
		path = filepath.Join(execDir, "..", "min-version.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check current working directory
	if _, err := os.Stat("min-version.json"); err == nil {
		return "min-version.json"
	}

	// Check in ~/.lux directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(homeDir, ".lux", "min-version.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetLuxdVersionsForRPC returns list of compatible lux go versions for a specified rpcVersion
func GetLuxdVersionsForRPC(app *application.Lux, rpcVersion int, url string) ([]string, error) {
	compatibilityBytes, err := app.Downloader.Download(url)
	if err != nil {
		return nil, err
	}

	var parsedCompat models.LuxdCompatiblity
	if err = json.Unmarshal(compatibilityBytes, &parsedCompat); err != nil {
		return nil, err
	}

	eligibleVersions, ok := parsedCompat[strconv.Itoa(rpcVersion)]
	if !ok {
		return nil, ErrNoLuxdVersion
	}

	// versions are not necessarily sorted, so we need to sort them, tho this puts them in ascending order
	semver.Sort(eligibleVersions)
	return eligibleVersions, nil
}

// GetAvailableLuxdVersions returns list of only available for download lux go versions,
// with latest version in first index
func GetAvailableLuxdVersions(app *application.Lux, rpcVersion int, url string) ([]string, error) {
	eligibleVersions, err := GetLuxdVersionsForRPC(app, rpcVersion, url)
	if err != nil {
		return nil, ErrNoLuxdVersion
	}
	// get latest luxd release to make sure we're not picking a release currently in progress but not available for download
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", constants.LuxOrg, constants.LuxdRepoName)
	latestLuxdVersion, err := app.Downloader.GetLatestReleaseVersion(releaseURL)
	if err != nil {
		return nil, err
	}
	var availableVersions []string
	for i := len(eligibleVersions) - 1; i >= 0; i-- {
		versionComparison := semver.Compare(eligibleVersions[i], latestLuxdVersion)
		if versionComparison != 1 {
			availableVersions = append(availableVersions, eligibleVersions[i])
		}
	}
	if len(availableVersions) == 0 {
		return nil, ErrNoLuxdVersion
	}
	return availableVersions, nil
}

type LuxdVersionSettings struct {
	UseCustomLuxgoVersion           string
	UseLatestLuxgoReleaseVersion    bool
	UseLatestLuxgoPreReleaseVersion bool
	UseLuxgoVersionFromSubnet       string
}

// GetLuxdVersion asks users whether they want to install the newest Lux Go version
// or if they want to use the newest Lux Go Version that is still compatible with Subnet EVM
// version of their choice
func GetLuxdVersion(app *application.Lux, luxdVersion LuxdVersionSettings, network models.Network) (string, error) {
	// skip this logic if custom-luxd-version flag is set
	if luxdVersion.UseCustomLuxgoVersion != "" {
		return luxdVersion.UseCustomLuxgoVersion, nil
	}
	latestReleaseVersion, err := GetLatestCLISupportedDependencyVersion(app, constants.LuxdRepoName, network, nil)
	if err != nil {
		return "", err
	}
	preReleaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", constants.LuxOrg, constants.LuxdRepoName)
	latestPreReleaseVersion, err := app.Downloader.GetLatestPreReleaseVersion(preReleaseURL)
	if err != nil {
		return "", err
	}

	if !luxdVersion.UseLatestLuxgoReleaseVersion && !luxdVersion.UseLatestLuxgoPreReleaseVersion && luxdVersion.UseCustomLuxgoVersion == "" && luxdVersion.UseLuxgoVersionFromSubnet == "" {
		luxdVersion, err = promptLuxdVersionChoice(app, latestReleaseVersion, latestPreReleaseVersion)
		if err != nil {
			return "", err
		}
	}

	var version string
	switch {
	case luxdVersion.UseLatestLuxgoReleaseVersion:
		version = latestReleaseVersion
	case luxdVersion.UseLatestLuxgoPreReleaseVersion:
		version = latestPreReleaseVersion
	case luxdVersion.UseCustomLuxgoVersion != "":
		version = luxdVersion.UseCustomLuxgoVersion
	case luxdVersion.UseLuxgoVersionFromSubnet != "":
		sc, err := app.LoadSidecar(luxdVersion.UseLuxgoVersionFromSubnet)
		if err != nil {
			return "", err
		}
		version, err = GetLatestCLISupportedDependencyVersion(app, constants.LuxdRepoName, network, &sc.RPCVersion)
		if err != nil {
			return "", err
		}
	}
	return version, nil
}

// promptLuxdVersionChoice sets flags for either using the latest Lux Go
// version or using the latest Lux Go version that is still compatible with the subnet that user
// wants the cloud server to track
func promptLuxdVersionChoice(app *application.Lux, latestReleaseVersion string, latestPreReleaseVersion string) (LuxdVersionSettings, error) {
	versionComments := map[string]string{
		"v1.11.0-testnet": " (recommended for testnet durango)",
	}
	latestReleaseVersionOption := "Use latest Lux Go Release Version" + versionComments[latestReleaseVersion]
	latestPreReleaseVersionOption := "Use latest Lux Go Pre-release Version" + versionComments[latestPreReleaseVersion]
	subnetBasedVersionOption := "Use the deployed Subnet's VM version that the node will be validating"
	customOption := "Custom"

	txt := "What version of Lux Go would you like to install in the node?"
	versionOptions := []string{latestReleaseVersionOption, subnetBasedVersionOption, customOption}
	if latestPreReleaseVersion != latestReleaseVersion {
		versionOptions = []string{latestPreReleaseVersionOption, latestReleaseVersionOption, subnetBasedVersionOption, customOption}
	}
	versionOption, err := app.Prompt.CaptureList(txt, versionOptions)
	if err != nil {
		return LuxdVersionSettings{}, err
	}

	switch versionOption {
	case latestReleaseVersionOption:
		return LuxdVersionSettings{UseLatestLuxgoReleaseVersion: true}, nil
	case latestPreReleaseVersionOption:
		return LuxdVersionSettings{UseLatestLuxgoPreReleaseVersion: true}, nil
	case customOption:
		useCustomLuxgoVersion, err := app.Prompt.CaptureVersion("Which version of Luxd would you like to install? (Use format v1.10.13)")
		if err != nil {
			return LuxdVersionSettings{}, err
		}
		return LuxdVersionSettings{UseCustomLuxgoVersion: useCustomLuxgoVersion}, nil
	default:
		useLuxgoVersionFromSubnet := ""
		for {
			useLuxgoVersionFromSubnet, err = app.Prompt.CaptureString("Which Subnet would you like to use to choose the lux go version?")
			if err != nil {
				return LuxdVersionSettings{}, err
			}
			err = chain.ValidateSubnetNameAndGetChains(useLuxgoVersionFromSubnet)
			if err == nil {
				break
			}
			ux.Logger.PrintToUser("%s", fmt.Sprintf("no blockchain named as %s found", useLuxgoVersionFromSubnet))
		}
		return LuxdVersionSettings{UseLuxgoVersionFromSubnet: useLuxgoVersionFromSubnet}, nil
	}
}
