// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package dependencies

import (
	"testing"

	"github.com/luxfi/cli/v2/v2/internal/mocks"
	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/constants"
	"github.com/luxfi/cli/v2/v2/pkg/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testCLIMinVersion = []byte(`{"subnet-evm":"v0.7.3","rpc":39,"luxd":{"Local Network":{"latest-version":"v1.13.0", "minimum-version":""},"DevNet":{"latest-version":"v1.13.0", "minimum-version":""},"Testnet":{"latest-version":"v1.13.0", "minimum-version":"v1.13.0-testnet"},"Mainnet":{"latest-version":"v1.13.0", "minimum-version":"v1.13.0"}}}`)

func TestCheckMinDependencyVersion(t *testing.T) {
	tests := []struct {
		name              string
		dependency        string
		expectedError     bool
		cliDependencyData []byte
		customVersion     string
		network           models.Network
	}{
		{
			name:              "custom luxd dependency equal to cli minimum supported version of luxd",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLIMinVersion,
			expectedError:     false,
			customVersion:     "v1.13.0-testnet",
			network:           models.NewTestnetNetwork(),
		},
		{
			name:              "custom luxd dependency higher than cli minimum supported version of luxd",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLIMinVersion,
			expectedError:     false,
			customVersion:     "v1.13.0",
			network:           models.NewTestnetNetwork(),
		},
		{
			name:              "custom luxd dependency equal to cli minimum supported version of luxd",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLIMinVersion,
			expectedError:     false,
			customVersion:     "v1.13.0-testnet",
			network:           models.NewTestnetNetwork(),
		},
		{
			name:              "custom luxd dependency higher than cli minimum supported version of luxd",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLIMinVersion,
			expectedError:     false,
			customVersion:     "v1.13.1",
			network:           models.NewTestnetNetwork(),
		},
		{
			name:              "custom luxd dependency lower than cli minimum supported version of luxd",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLIMinVersion,
			expectedError:     true,
			customVersion:     "v1.12.2",
			network:           models.NewTestnetNetwork(),
		},
		{
			name:              "custom luxd dependency for network that doesn't have minimum supported version of luxd",
			dependency:        constants.LuxdRepoName,
			cliDependencyData: testCLIMinVersion,
			expectedError:     false,
			customVersion:     "v1.12.2",
			network:           models.NewLocalNetwork(),
		},
	}

	for _, tt := range tests {
		mockDownloader := &mocks.Downloader{}
		mockDownloader.On("Download", mock.MatchedBy(func(url string) bool {
			return url == constants.CLILatestDependencyURL
		})).Return(tt.cliDependencyData, nil)

		app := application.New()
		app.Downloader = mockDownloader

		t.Run(tt.name, func(t *testing.T) {
			err := CheckVersionIsOverMin(app, tt.dependency, tt.network, tt.customVersion)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
