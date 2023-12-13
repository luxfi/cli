// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"testing"

	"github.com/luxdefi/cli/internal/mocks"
	"github.com/luxdefi/cli/internal/testutils"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/node/ids"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testLuxdCompat = []byte("{\"19\": [\"v1.9.2\"],\"18\": [\"v1.9.1\"],\"17\": [\"v1.9.0\",\"v1.8.0\"]}")

func Test_determineLuxdVersion(t *testing.T) {
	subnetName1 := "test1"
	subnetName2 := "test2"
	subnetName3 := "test3"
	subnetName4 := "test4"

	dummySlice := ids.ID{1, 2, 3, 4}

	sc1 := models.Sidecar{
		Name: subnetName1,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   18,
			},
		},
		VM: models.SubnetEvm,
	}

	sc2 := models.Sidecar{
		Name: subnetName2,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   18,
			},
		},
		VM: models.SubnetEvm,
	}

	sc3 := models.Sidecar{
		Name: subnetName3,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   19,
			},
		},
		VM: models.SubnetEvm,
	}

	scCustom := models.Sidecar{
		Name: subnetName4,
		Networks: map[string]models.NetworkData{
			models.Local.String(): {
				SubnetID:     dummySlice,
				BlockchainID: dummySlice,
				RPCVersion:   0,
			},
		},
		VM: models.CustomVM,
	}

	type test struct {
		name          string
		userLuxd     string
		sidecars      []models.Sidecar
		expectedLuxd string
		expectedErr   bool
	}

	tests := []test{
		{
			name:          "user not latest",
			userLuxd:     "v1.9.5",
			sidecars:      []models.Sidecar{sc1},
			expectedLuxd: "v1.9.5",
			expectedErr:   false,
		},
		{
			name:          "single sc",
			userLuxd:     "latest",
			sidecars:      []models.Sidecar{sc1},
			expectedLuxd: "v1.9.1",
			expectedErr:   false,
		},
		{
			name:          "multi sc matching",
			userLuxd:     "latest",
			sidecars:      []models.Sidecar{sc1, sc2},
			expectedLuxd: "v1.9.1",
			expectedErr:   false,
		},
		{
			name:          "multi sc mismatch",
			userLuxd:     "latest",
			sidecars:      []models.Sidecar{sc1, sc3},
			expectedLuxd: "",
			expectedErr:   true,
		},
		{
			name:          "single custom",
			userLuxd:     "latest",
			sidecars:      []models.Sidecar{scCustom},
			expectedLuxd: "latest",
			expectedErr:   false,
		},
		{
			name:          "custom plus user selected",
			userLuxd:     "v1.9.1",
			sidecars:      []models.Sidecar{scCustom},
			expectedLuxd: "v1.9.1",
			expectedErr:   false,
		},
		{
			name:          "multi sc matching plus custom",
			userLuxd:     "latest",
			sidecars:      []models.Sidecar{sc1, sc2, scCustom},
			expectedLuxd: "v1.9.1",
			expectedErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app = testutils.SetupTestInTempDir(t)
			mockDownloader := &mocks.Downloader{}
			mockDownloader.On("Download", mock.Anything).Return(testLuxdCompat, nil)
			mockDownloader.On("GetLatestReleaseVersion", mock.Anything).Return("v1.9.2", nil)

			app.Downloader = mockDownloader

			for i := range tt.sidecars {
				err := app.CreateSidecar(&tt.sidecars[i])
				require.NoError(t, err)
			}

			luxdVersion, err := determineLuxdVersion(tt.userLuxd)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedLuxd, luxdVersion)
		})
	}
}
