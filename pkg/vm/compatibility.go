// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"golang.org/x/mod/semver"
)

var ErrNoLuxVersion = errors.New("unable to find a compatible node version")

func GetRPCProtocolVersion(app *application.Lux, vmType models.VMType, vmVersion string) (int, error) {
	var url string

	switch vmType {
	case models.SubnetEvm:
		url = constants.EVMRPCCompatibilityURL
	default:
		return 0, errors.New("unknown VM type")
	}

	compatibilityBytes, err := app.Downloader.Download(url)
	if err != nil {
		return 0, err
	}

	var parsedCompat models.VMCompatibility
	if err = json.Unmarshal(compatibilityBytes, &parsedCompat); err != nil {
		return 0, err
	}

	version, ok := parsedCompat.RPCChainVMProtocolVersion[vmVersion]
	if !ok {
		return 0, errors.New("no RPC version found")
	}

	return version, nil
}

func GetLatestLuxByProtocolVersion(app *application.Lux, rpcVersion int, url string) (string, error) {
	compatibilityBytes, err := app.Downloader.Download(url)
	if err != nil {
		return "", err
	}

	var parsedCompat models.LuxCompatiblity
	if err = json.Unmarshal(compatibilityBytes, &parsedCompat); err != nil {
		return "", err
	}

	eligibleVersions, ok := parsedCompat[strconv.Itoa(rpcVersion)]
	if !ok {
		return "", ErrNoLuxVersion
	}

	// versions are not necessarily sorted, so we need to sort them, tho this puts them in ascending order
	semver.Sort(eligibleVersions)

	// get latest lux release to make sure we're not picking a release currently in progress but not available for download
	latestLuxVersion, err := app.Downloader.GetLatestReleaseVersion(binutils.GetGithubLatestReleaseURL(
		constants.LuxOrg,
		constants.LuxRepoName,
	))
	if err != nil {
		return "", err
	}

	// we need to iterate in reverse order to start with latest version
	var useVersion string
	for i := len(eligibleVersions) - 1; i >= 0; i-- {
		versionComparison := semver.Compare(eligibleVersions[i], latestLuxVersion)
		if versionComparison != 1 {
			useVersion = eligibleVersions[i]
			break
		}
	}

	if useVersion == "" {
		return "", ErrNoLuxVersion
	}

	return useVersion, nil
}
