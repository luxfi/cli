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

var ErrNoAvagoVersion = errors.New("unable to find a compatible luxd version")

func GetLatestLuxGoByProtocolVersion(app *application.Lux, rpcVersion int) (string, error) {
	useVersion, err := GetAvailableLuxGoVersions(app, rpcVersion, constants.LuxGoCompatibilityURL)
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
	case constants.LuxGoRepoName:
		// if the user is using RPC that is lower than the latest RPC supported by CLI, user will get latest LuxGo version for that RPC
		// based on "https://raw.githubusercontent.com/luxfi/luxd/master/version/compatibility.json"
		if rpcVersion != nil && parsedDependency.RPC > *rpcVersion {
			return GetLatestLuxGoByProtocolVersion(
				app,
				*rpcVersion,
			)
		}
		return parsedDependency.LuxGo[network.Name()].LatestVersion, nil
	case constants.SubnetEVMRepoName:
		return parsedDependency.SubnetEVM, nil
	default:
		return "", fmt.Errorf("unsupported dependency: %s", dependencyName)
	}
}

// GetLuxGoVersionsForRPC returns list of compatible lux go versions for a specified rpcVersion
func GetLuxGoVersionsForRPC(app *application.Lux, rpcVersion int, url string) ([]string, error) {
	compatibilityBytes, err := app.Downloader.Download(url)
	if err != nil {
		return nil, err
	}

	var parsedCompat models.AvagoCompatiblity
	if err = json.Unmarshal(compatibilityBytes, &parsedCompat); err != nil {
		return nil, err
	}

	eligibleVersions, ok := parsedCompat[strconv.Itoa(rpcVersion)]
	if !ok {
		return nil, ErrNoAvagoVersion
	}

	// versions are not necessarily sorted, so we need to sort them, tho this puts them in ascending order
	semver.Sort(eligibleVersions)
	return eligibleVersions, nil
}

// GetAvailableLuxGoVersions returns list of only available for download lux go versions,
// with latest version in first index
func GetAvailableLuxGoVersions(app *application.Lux, rpcVersion int, url string) ([]string, error) {
	eligibleVersions, err := GetLuxGoVersionsForRPC(app, rpcVersion, url)
	if err != nil {
		return nil, ErrNoAvagoVersion
	}
	// get latest avago release to make sure we're not picking a release currently in progress but not available for download
	latestAvagoVersion, err := app.Downloader.GetLatestReleaseVersion(
		constants.LuxOrg,
		constants.LuxGoRepoName,
		"",
	)
	if err != nil {
		return nil, err
	}
	var availableVersions []string
	for i := len(eligibleVersions) - 1; i >= 0; i-- {
		versionComparison := semver.Compare(eligibleVersions[i], latestAvagoVersion)
		if versionComparison != 1 {
			availableVersions = append(availableVersions, eligibleVersions[i])
		}
	}
	if len(availableVersions) == 0 {
		return nil, ErrNoAvagoVersion
	}
	return availableVersions, nil
}

type LuxGoVersionSettings struct {
	UseCustomLuxgoVersion           string
	UseLatestLuxgoReleaseVersion    bool
	UseLatestLuxgoPreReleaseVersion bool
	UseLuxgoVersionFromSubnet       string
}

// GetLuxGoVersion asks users whether they want to install the newest Lux Go version
// or if they want to use the newest Lux Go Version that is still compatible with Subnet EVM
// version of their choice
func GetLuxGoVersion(app *application.Lux, avagoVersion LuxGoVersionSettings, network models.Network) (string, error) {
	// skip this logic if custom-luxd-version flag is set
	if avagoVersion.UseCustomLuxgoVersion != "" {
		return avagoVersion.UseCustomLuxgoVersion, nil
	}
	latestReleaseVersion, err := GetLatestCLISupportedDependencyVersion(app, constants.LuxGoRepoName, network, nil)
	if err != nil {
		return "", err
	}
	latestPreReleaseVersion, err := app.Downloader.GetLatestPreReleaseVersion(
		constants.LuxOrg,
		constants.LuxGoRepoName,
		"",
	)
	if err != nil {
		return "", err
	}

	if !avagoVersion.UseLatestLuxgoReleaseVersion && !avagoVersion.UseLatestLuxgoPreReleaseVersion && avagoVersion.UseCustomLuxgoVersion == "" && avagoVersion.UseLuxgoVersionFromSubnet == "" {
		avagoVersion, err = promptLuxGoVersionChoice(app, latestReleaseVersion, latestPreReleaseVersion)
		if err != nil {
			return "", err
		}
	}

	var version string
	switch {
	case avagoVersion.UseLatestLuxgoReleaseVersion:
		version = latestReleaseVersion
	case avagoVersion.UseLatestLuxgoPreReleaseVersion:
		version = latestPreReleaseVersion
	case avagoVersion.UseCustomLuxgoVersion != "":
		version = avagoVersion.UseCustomLuxgoVersion
	case avagoVersion.UseLuxgoVersionFromSubnet != "":
		sc, err := app.LoadSidecar(avagoVersion.UseLuxgoVersionFromSubnet)
		if err != nil {
			return "", err
		}
		version, err = GetLatestCLISupportedDependencyVersion(app, constants.LuxGoRepoName, network, &sc.RPCVersion)
		if err != nil {
			return "", err
		}
	}
	return version, nil
}

// promptLuxGoVersionChoice sets flags for either using the latest Lux Go
// version or using the latest Lux Go version that is still compatible with the subnet that user
// wants the cloud server to track
func promptLuxGoVersionChoice(app *application.Lux, latestReleaseVersion string, latestPreReleaseVersion string) (LuxGoVersionSettings, error) {
	versionComments := map[string]string{
		"v1.11.0-fuji": " (recommended for fuji durango)",
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
		return LuxGoVersionSettings{}, err
	}

	switch versionOption {
	case latestReleaseVersionOption:
		return LuxGoVersionSettings{UseLatestLuxgoReleaseVersion: true}, nil
	case latestPreReleaseVersionOption:
		return LuxGoVersionSettings{UseLatestLuxgoPreReleaseVersion: true}, nil
	case customOption:
		useCustomLuxgoVersion, err := app.Prompt.CaptureVersion("Which version of LuxGo would you like to install? (Use format v1.10.13)")
		if err != nil {
			return LuxGoVersionSettings{}, err
		}
		return LuxGoVersionSettings{UseCustomLuxgoVersion: useCustomLuxgoVersion}, nil
	default:
		useLuxgoVersionFromSubnet := ""
		for {
			useLuxgoVersionFromSubnet, err = app.Prompt.CaptureString("Which Subnet would you like to use to choose the lux go version?")
			if err != nil {
				return LuxGoVersionSettings{}, err
			}
			_, err = subnet.ValidateSubnetNameAndGetChains(app, []string{useLuxgoVersionFromSubnet})
			if err == nil {
				break
			}
			ux.Logger.PrintToUser(fmt.Sprintf("no blockchain named as %s found", useLuxgoVersionFromSubnet))
		}
		return LuxGoVersionSettings{UseLuxgoVersionFromSubnet: useLuxgoVersionFromSubnet}, nil
	}
}
