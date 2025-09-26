// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package dependencies

import (
	"testing"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	testLuxdCompat  = []byte("{\"19\": [\"v1.9.2\"],\"18\": [\"v1.9.1\"],\"17\": [\"v1.9.0\",\"v1.8.0\"]}")
	testLuxdCompat2 = []byte("{\"19\": [\"v1.9.2\", \"v1.9.1\"],\"18\": [\"v1.9.0\"]}")
	testLuxdCompat3 = []byte("{\"19\": [\"v1.9.1\", \"v1.9.2\"],\"18\": [\"v1.9.0\"]}")
	testLuxdCompat4 = []byte("{\"19\": [\"v1.9.1\", \"v1.9.2\", \"v1.9.11\"],\"18\": [\"v1.9.0\"]}")
	testLuxdCompat5 = []byte("{\"39\": [\"v1.12.2\", \"v1.13.0\"],\"38\": [\"v1.11.13\", \"v1.12.0\", \"v1.12.1\"]}")
	testLuxdCompat6 = []byte("{\"39\": [\"v1.12.2\", \"v1.13.0\", \"v1.13.1\"],\"38\": [\"v1.11.13\", \"v1.12.0\", \"v1.12.1\"]}")
	testLuxdCompat7 = []byte("{\"40\": [\"v1.13.2\"],\"39\": [\"v1.12.2\", \"v1.13.0\", \"v1.13.1\"]}")
	testCLICompat   = []byte(`{"subnet-evm":"v0.7.3","rpc":39,"luxd":{"Local Network":{"latest-version":"v1.13.0"},"DevNet":{"latest-version":"v1.13.0"},"Testnet":{"latest-version":"v1.13.0"},"Mainnet":{"latest-version":"v1.13.0"}}}`)
	testCLICompat2  = []byte(`{"subnet-evm":"v0.7.3","rpc":39,"luxd":{"Local Network":{"latest-version":"v1.13.0"},"DevNet":{"latest-version":"v1.13.0"},"Testnet":{"latest-version":"v1.13.0-testnet"},"Mainnet":{"latest-version":"v1.13.0"}}}`)
)

func TestGetLatestLuxdByProtocolVersion(t *testing.T) {
	type versionTest struct {
		name            string
		rpc             int
		testData        []byte
		latestVersion   string
		expectedVersion string
		expectedErr     error
	}

	tests := []versionTest{
		{
			name:            "latest, one entry",
			rpc:             19,
			testData:        testLuxdCompat,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
		{
			name:            "older, one entry",
			rpc:             18,
			testData:        testLuxdCompat,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.1",
			expectedErr:     nil,
		},
		{
			name:            "latest, multiple entry",
			rpc:             19,
			testData:        testLuxdCompat2,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
		{
			name:            "latest, multiple entry, reverse sorted",
			rpc:             19,
			testData:        testLuxdCompat3,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
		{
			name:            "latest, multiple entry, unreleased version",
			rpc:             19,
			testData:        testLuxdCompat2,
			latestVersion:   "v1.9.1",
			expectedVersion: "v1.9.1",
			expectedErr:     nil,
		},
		{
			name:            "no rpc version",
			rpc:             20,
			testData:        testLuxdCompat2,
			latestVersion:   "v1.9.2",
			expectedVersion: "",
			expectedErr:     ErrNoLuxdVersion,
		},
		{
			name:            "existing rpc, but no eligible version",
			rpc:             19,
			testData:        testLuxdCompat,
			latestVersion:   "v1.9.1",
			expectedVersion: "",
			expectedErr:     ErrNoLuxdVersion,
		},
		{
			name:            "string sorting test",
			rpc:             19,
			testData:        testLuxdCompat4,
			latestVersion:   "v1.9.11",
			expectedVersion: "v1.9.11",
			expectedErr:     nil,
		},
		{
			name:            "string sorting test 2",
			rpc:             19,
			testData:        testLuxdCompat4,
			latestVersion:   "v1.9.2",
			expectedVersion: "v1.9.2",
			expectedErr:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			mockDownloader := &mocks.Downloader{}
			mockDownloader.On("Download", mock.Anything).Return(tt.testData, nil)
			mockDownloader.On("GetLatestReleaseVersion", mock.Anything, mock.Anything, mock.Anything).Return(tt.latestVersion, nil)

			app := application.New()
			app.Downloader = mockDownloader

			luxdVersion, err := GetLatestLuxdByProtocolVersion(app, tt.rpc)
			if tt.expectedErr == nil {
				require.NoError(err)
			} else {
				require.ErrorIs(err, tt.expectedErr)
			}
			require.Equal(tt.expectedVersion, luxdVersion)
		})
	}
}

func TestGetLatestCLISupportedDependencyVersion(t *testing.T) {
	tests := []struct {
		name              string
		dependency        string
		expectedError     bool
		expectedResult    string
		cliDependencyData []byte
		luxdData          []byte
		latestVersion     string
	}{
		{
			name:              "luxd dependency with cli supporting latest luxd release",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLICompat,
			luxdData:          testLuxdCompat5,
			latestVersion:     "v1.13.0",
			expectedError:     false,
			expectedResult:    "v1.13.0",
		},
		{
			name:              "luxd dependency with cli not supporting latest luxd release, but same rpc",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLICompat,
			luxdData:          testLuxdCompat6,
			latestVersion:     "v1.13.1",
			expectedError:     false,
			expectedResult:    "v1.13.0",
		},
		{
			name:              "luxd dependency with cli supporting lower rpc",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLICompat,
			luxdData:          testLuxdCompat7,
			latestVersion:     "v1.13.2",
			expectedError:     false,
			expectedResult:    "v1.13.0",
		},
		{
			name:              "luxd dependency with cli requiring a prerelease",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLICompat2,
			luxdData:          testLuxdCompat7,
			latestVersion:     "v1.13.2",
			expectedError:     false,
			expectedResult:    "v1.13.0-testnet",
		},
		{
			name:              "subnet-evm dependency, where cli latest.json doesn't support newest subnet evm version yet",
			dependency:        constants.SubnetEVMRepoName,
			cliDependencyData: testCLICompat,
			expectedError:     false,
			expectedResult:    "v0.7.3",
			latestVersion:     "v0.7.4",
		},
		{
			name:              "subnet-evm dependency, where cli supports newest subnet evm version",
			dependency:        constants.SubnetEVMRepoName,
			cliDependencyData: testCLICompat,
			expectedError:     false,
			expectedResult:    "v0.7.3",
			latestVersion:     "v0.7.3",
		},
		{
			name:           "empty dependency",
			dependency:     "",
			expectedError:  true,
			expectedResult: "",
		},
		{
			name:           "invalid dependency",
			dependency:     "invalid",
			expectedError:  true,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		mockDownloader := &mocks.Downloader{}
		mockDownloader.On("Download", mock.MatchedBy(func(url string) bool {
			return url == constants.CLILatestDependencyURL
		})).Return(tt.cliDependencyData, nil)

		mockDownloader.On("Download", mock.MatchedBy(func(url string) bool {
			return url == constants.LuxdCompatibilityURL
		})).Return(tt.luxdData, nil)
		mockDownloader.On("GetLatestReleaseVersion", mock.Anything, mock.Anything, mock.Anything).Return(tt.latestVersion, nil)

		app := application.New()
		app.Downloader = mockDownloader

		t.Run(tt.name, func(t *testing.T) {
			rpcVersion := 39
			result, err := GetLatestCLISupportedDependencyVersion(app, tt.dependency, models.NewTestnetNetwork(), &rpcVersion)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestGetLatestCLISupportedDependencyVersionWithLowerRPC(t *testing.T) {
	tests := []struct {
		name              string
		dependency        string
		expectedError     bool
		expectedResult    string
		cliDependencyData []byte
		luxdData          []byte
		latestVersion     string
	}{
		{
			name:              "luxd dependency with cli supporting latest luxd release, user using lower rpc",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLICompat,
			luxdData:          testLuxdCompat5,
			expectedError:     false,
			expectedResult:    "v1.12.1",
			latestVersion:     "v1.13.0",
		},
		{
			name:              "luxd dependency with cli supporting latest luxd release, user using lower rpc, prerelease required",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLICompat2,
			luxdData:          testLuxdCompat6,
			expectedError:     false,
			expectedResult:    "v1.12.1",
			latestVersion:     "v1.13.2",
		},
		{
			name:              "subnet-evm dependency, where cli supports newest subnet evm version",
			dependency:        constants.SubnetEVMRepoName,
			cliDependencyData: testCLICompat,
			expectedError:     false,
			expectedResult:    "v0.7.3",
			latestVersion:     "v0.7.3",
		},
		{
			name:           "empty dependency",
			dependency:     "",
			expectedError:  true,
			expectedResult: "",
		},
		{
			name:           "invalid dependency",
			dependency:     "invalid",
			expectedError:  true,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		mockDownloader := &mocks.Downloader{}
		mockDownloader.On("Download", mock.MatchedBy(func(url string) bool {
			return url == constants.CLILatestDependencyURL
		})).Return(tt.cliDependencyData, nil)

		mockDownloader.On("Download", mock.MatchedBy(func(url string) bool {
			return url == constants.LuxdCompatibilityURL
		})).Return(tt.luxdData, nil)
		mockDownloader.On("GetLatestReleaseVersion", mock.Anything, mock.Anything, mock.Anything).Return(tt.latestVersion, nil)

		app := application.New()
		app.Downloader = mockDownloader

		t.Run(tt.name, func(t *testing.T) {
			rpcVersion := 38
			result, err := GetLatestCLISupportedDependencyVersion(app, tt.dependency, models.NewTestnetNetwork(), &rpcVersion)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
