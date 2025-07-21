// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"sync"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/node/utils/logging"
	"golang.org/x/mod/semver"
)

var (
	binaryToVersion map[string]string
	lock            = &sync.Mutex{}

	_ VersionMapper = &versionMapper{}
)

/*
VersionMapper keys and their usage:
 * OnlyLuxKey: 					Used when running one node only (no compatibility required)

 * MultiLux1Key					Used for the update scenario where node is updated and
 * MultiLux2Key    			both node versions need to be compatible.
 * MultiLuxEVMKey	This is the Lux EVM version compatible to the above scenario.

 * LatestEVM2LuxKey 	  Latest evm version
 * LatestLux2EVMKey     while this is the latest node compatible with that evm

 * SoloEVMKey1 			This is used when we want to test evm versions where compatibility
 * SoloEVMKey2      needs to be between the two evm versions
 													(latest might not be compatible with second latest)


*/

// VersionMapper is an abstraction for retrieving version compatibility URLs
// allowing unit tests without requiring external http calls.
// The idea is to finally calculate which VM is compatible with which Lux,
// so that the e2e tests can always download and run the latest compatible versions,
// without having to manually update the e2e tests periodically.
type VersionMapper interface {
	GetCompatURL(vmType models.VMType) string
	GetLuxURL() string
	GetApp() *application.Lux
	GetLatestLuxByProtoVersion(app *application.Lux, rpcVersion int, url string) (string, error)
	GetEligibleVersions(sortedVersions []string, repoName string, app *application.Lux) ([]string, error)
}

// NewVersionMapper returns the default VersionMapper for e2e tests
func NewVersionMapper() VersionMapper {
	app := &application.Lux{
		Downloader: application.NewDownloader(),
		Log:        logging.NoLog{},
	}
	return &versionMapper{
		app: app,
	}
}

// versionMapper is the default implementation for version mapping.
// It downloads compatibility URLs from the actual github endpoints
type versionMapper struct {
	app *application.Lux
}

// GetLatestLuxByProtoVersion returns the latest Lux version which
// runs with the specified rpcVersion, or an error if it can't be found
// (or other errors occurred)
func (*versionMapper) GetLatestLuxByProtoVersion(app *application.Lux, rpcVersion int, url string) (string, error) {
	return vm.GetLatestLuxByProtocolVersion(app, rpcVersion, url)
}

// GetApp returns the Lux application instance
func (m *versionMapper) GetApp() *application.Lux {
	return m.app
}

// GetCompatURL returns the compatibility URL for the given VM type
func (*versionMapper) GetCompatURL(vmType models.VMType) string {
	switch vmType {
	case models.SubnetEvm:
		return constants.EVMRPCCompatibilityURL
	case models.CustomVM:
		// TODO: unclear yet what we should return here
		return ""
	default:
		return ""
	}
}

// GetLuxURL returns the compatibility URL for Lux
func (*versionMapper) GetLuxURL() string {
	return constants.LuxCompatibilityURL
}

func (*versionMapper) GetEligibleVersions(sortedVersions []string, repoName string, app *application.Lux) ([]string, error) {
	// get latest lux release to make sure we're not picking a release currently in progress but not available for download
	latest, err := app.Downloader.GetLatestReleaseVersion(binutils.GetGithubLatestReleaseURL(
		constants.LuxOrg,
		repoName,
	))
	if err != nil {
		return nil, err
	}

	var eligible []string
	for i, ver := range sortedVersions {
		versionComparison := semver.Compare(ver, latest)
		if versionComparison != 0 {
			continue
		}
		eligible = sortedVersions[i:]
		break
	}

	return eligible, nil
}

// GetVersionMapping returns a map of specific VMs resp. Lux e2e context keys
// to the actual version which corresponds to that key.
// This allows the e2e test to know what version to download and run.
// Returns an error if there was a problem reading the URL compatibility json
// or some other issue.
func GetVersionMapping(mapper VersionMapper) (map[string]string, error) {
	// ginkgo can run tests in parallel. However, we really just need this mapping to be
	// performed once for the whole duration of a test.
	// Therefore we store the result in a global variable, and then lock
	// the access to it.
	lock.Lock()
	defer lock.Unlock()
	// if mapping has already been done, return it right away
	if binaryToVersion != nil {
		return binaryToVersion, nil
	}
	// get compatible versions for subnetEVM
	// subnetEVMversions is a list of sorted EVM versions,
	// subnetEVMmapping maps EVM versions to their RPC versions
	subnetEVMversions, subnetEVMmapping, err := getVersions(mapper, models.SubnetEvm)
	if err != nil {
		return nil, err
	}

	// evm publishes its upcoming new version in the compatibility json
	// before the new version is actually a downloadable release
	subnetEVMversions, err = mapper.GetEligibleVersions(subnetEVMversions, constants.EVMRepoName, mapper.GetApp())
	if err != nil {
		return nil, err
	}

	// now get the node compatibility object
	luxCompat, err := getLuxCompatibility(mapper)
	if err != nil {
		return nil, err
	}

	// create the global mapping variable
	binaryToVersion = make(map[string]string)

	// sort lux compatibility by highest available RPC versions
	// to lowest (the map can not be iterated in a sorted way)
	rpcs := make([]int, 0, len(luxCompat))
	for k := range luxCompat {
		// cannot use string sort
		kint, err := strconv.Atoi(k)
		if err != nil {
			return nil, err
		}
		rpcs = append(rpcs, kint)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(rpcs)))

	// iterate the rpc versions
	// evaluate two node versions which are consecutive
	// and run with the same RPC version.
	// This is required for the for the "can deploy with multiple node versions" test
	for _, rpcVersion := range rpcs {
		versionAsString := strconv.Itoa(rpcVersion)
		versionsForRPC := luxCompat[versionAsString]
		// we need at least 2 versions for the same RPC version
		if len(versionsForRPC) > 1 {
			versionsForRPC = reverseSemverSort(versionsForRPC)
			binaryToVersion[MultiLux1Key] = versionsForRPC[0]
			binaryToVersion[MultiLux2Key] = versionsForRPC[1]

			// now iterate the subnetEVMversions and find a
			// evm version which is compatible with that RPC version.
			// The above-mentioned test runs with this as well.
			for _, evmVer := range subnetEVMversions {
				if subnetEVMmapping[evmVer] == rpcVersion {
					// we know there already exists at least one such combination.
					// unless the compatibility JSON will start to be shortened in some way,
					// we should always be able to find a matching evm
					binaryToVersion[MultiLuxEVMKey] = evmVer
					// found the version, break
					break
				}
			}
			// all good, don't need to look more
			break
		}
	}

	// when running Lux only, always use latest
	binaryToVersion[OnlyLuxKey] = OnlyLuxValue

	// now let's look for evm versions which are fit for the
	// "can deploy multiple evm versions" test.
	// We need two evm versions which run the same RPC version,
	// and then a compatible Lux
	//
	// To avoid having to iterate again, we'll also fill the values
	// for the **latest** compatible Lux and Lux EVM
	for i, ver := range subnetEVMversions {
		// safety check, should not happen, as we already know
		// compatible versions exist
		if i+1 == len(subnetEVMversions) {
			return nil, errors.New("no compatible versions for subsequent EVM found")
		}
		first := ver
		second := subnetEVMversions[i+1]
		// we should be able to safely assume that for a given evm RPC version,
		// there exists at least one compatible Lux.
		// This means we can in any case use this to set the **latest** compatibility
		soloLux, err := mapper.GetLatestLuxByProtoVersion(mapper.GetApp(), subnetEVMmapping[first], mapper.GetLuxURL())
		if err != nil {
			return nil, err
		}
		// Once latest compatibility has been set, we can skip this
		if binaryToVersion[LatestEVM2LuxKey] == "" {
			binaryToVersion[LatestEVM2LuxKey] = first
			binaryToVersion[LatestLux2EVMKey] = soloLux
		}
		// first and second are compatible
		if subnetEVMmapping[first] == subnetEVMmapping[second] {
			binaryToVersion[SoloEVMKey1] = first
			binaryToVersion[SoloEVMKey2] = second
			binaryToVersion[SoloLuxKey] = soloLux
			break
		}
	}

	return binaryToVersion, nil
}

// getVersions gets compatible versions for the given VM type.
// Returns a correctly ordered list of semantic version strings,
// from latest to oldest, and a map of version to rpc
func getVersions(mapper VersionMapper, vmType models.VMType) ([]string, map[string]int, error) {
	compat, err := getCompatibility(mapper, vmType)
	if err != nil {
		return nil, nil, err
	}
	mapping := compat.RPCChainVMProtocolVersion
	if len(mapping) == 0 {
		return nil, nil, errors.New("zero length rpcs")
	}
	versions := make([]string, len(mapping))
	if len(versions) == 0 {
		return nil, nil, errors.New("zero length versions")
	}
	i := 0
	for v := range mapping {
		versions[i] = v
		i++
	}

	versions = reverseSemverSort(versions)
	return versions, mapping, nil
}

// getCompatibility returns the compatibility object for the given VM type
func getCompatibility(mapper VersionMapper, vmType models.VMType) (models.VMCompatibility, error) {
	compatibilityBytes, err := mapper.GetApp().GetDownloader().Download(mapper.GetCompatURL(vmType))
	if err != nil {
		return models.VMCompatibility{}, err
	}

	var parsedCompat models.VMCompatibility
	if err = json.Unmarshal(compatibilityBytes, &parsedCompat); err != nil {
		return models.VMCompatibility{}, err
	}
	return parsedCompat, nil
}

// getLuxCompatibility returns the compatibility for Lux
func getLuxCompatibility(mapper VersionMapper) (models.LuxCompatiblity, error) {
	luxBytes, err := mapper.GetApp().GetDownloader().Download(mapper.GetLuxURL())
	if err != nil {
		return nil, err
	}

	var luxCompat models.LuxCompatiblity
	if err = json.Unmarshal(luxBytes, &luxCompat); err != nil {
		return nil, err
	}

	return luxCompat, nil
}

// For semantic version slices, we can't just reverse twice:
// the semver packages only has increasing `Sort`, while
// `sort.Sort(sort.Reverse(sort.StringSlice(sliceSortedWithSemverSort)))`
// again fails to sort correctly (as it will sort again with string sorting
// instead of semantic versioning)
func reverseSemverSort(slice []string) []string {
	semver.Sort(slice)
	reverse := make([]string, len(slice))
	for i, s := range slice {
		idx := len(slice) - (1 + i)
		reverse[idx] = s
	}
	return reverse
}
