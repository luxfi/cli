// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package dependencies

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/ux"

	"golang.org/x/mod/semver"

	"github.com/luxfi/cli/pkg/models"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

var ErrNoLuxdVersion = errors.New("unable to find a compatible luxd version")

func GetLatestLuxdByProtocolVersion(app *application.Lux, rpcVersion int) (string, error) {
	useVersion, err := GetAvailableLuxdVersions(app, rpcVersion, constants.LuxdCompatibilityURL)
	if err != nil {
		return "", err
	}
	return useVersion[0], nil
}

func GetLatestCLISupportedDependencyVersion(app *application.Lux, dependencyName string, network models.Network, rpcVersion *int) (string, error) {
	dependencyBytes, err := app.Downloader.Download(constants.CLILatestDependencyURL)
	if err != nil {
		return "", err
	}

	var parsedDependency models.CLIDependencyMap
	if err = json.Unmarshal(dependencyBytes, &parsedDependency); err != nil {
		return "", err
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
	latestLuxdVersion, err := app.Downloader.GetLatestReleaseVersion(
		constants.LuxOrg,
		constants.LuxdRepoName,
		"",
	)
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
	latestPreReleaseVersion, err := app.Downloader.GetLatestPreReleaseVersion(
		constants.LuxOrg,
		constants.LuxdRepoName,
		"",
	)
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
			_, err = subnet.ValidateSubnetNameAndGetChains(app, []string{useLuxgoVersionFromSubnet})
			if err == nil {
				break
			}
			ux.Logger.PrintToUser(fmt.Sprintf("no blockchain named as %s found", useLuxgoVersionFromSubnet))
		}
		return LuxdVersionSettings{UseLuxgoVersionFromSubnet: useLuxgoVersionFromSubnet}, nil
	}
}
