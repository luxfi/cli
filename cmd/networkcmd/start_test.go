// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"testing"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/cli/internal/testutils"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/node/ids"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testLuxCompat = []byte("{\"19\": [\"v1.9.2\"],\"18\": [\"v1.9.1\"],\"17\": [\"v1.9.0\",\"v1.8.0\"]}")

func Test_determineLuxVersion(t *testing.T) {
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
		VM: models.EVM,
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
		VM: models.EVM,
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
		VM: models.EVM,
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
		name        string
		userLux     string
		sidecars    []models.Sidecar
		expectedLux string
		expectedErr bool
	}

	tests := []test{
		{
			name:        "user not latest",
			userLux:     "v1.9.5",
			sidecars:    []models.Sidecar{sc1},
			expectedLux: "v1.9.5",
			expectedErr: false,
		},
		{
			name:        "single sc",
			userLux:     "latest",
			sidecars:    []models.Sidecar{sc1},
			expectedLux: "v1.9.1",
			expectedErr: false,
		},
		{
			name:        "multi sc matching",
			userLux:     "latest",
			sidecars:    []models.Sidecar{sc1, sc2},
			expectedLux: "v1.9.1",
			expectedErr: false,
		},
		{
			name:        "multi sc mismatch",
			userLux:     "latest",
			sidecars:    []models.Sidecar{sc1, sc3},
			expectedLux: "",
			expectedErr: true,
		},
		{
			name:        "single custom",
			userLux:     "latest",
			sidecars:    []models.Sidecar{scCustom},
			expectedLux: "latest",
			expectedErr: false,
		},
		{
			name:        "custom plus user selected",
			userLux:     "v1.9.1",
			sidecars:    []models.Sidecar{scCustom},
			expectedLux: "v1.9.1",
			expectedErr: false,
		},
		{
			name:        "multi sc matching plus custom",
			userLux:     "latest",
			sidecars:    []models.Sidecar{sc1, sc2, scCustom},
			expectedLux: "v1.9.1",
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app = testutils.SetupTestInTempDir(t)
			mockDownloader := &mocks.Downloader{}
			mockDownloader.On("Download", mock.Anything).Return(testLuxCompat, nil)
			mockDownloader.On("GetLatestReleaseVersion", mock.Anything).Return("v1.9.2", nil)

			app.Downloader = mockDownloader

			for i := range tt.sidecars {
				err := app.CreateSidecar(&tt.sidecars[i])
				require.NoError(t, err)
			}

			luxVersion, err := determineLuxVersion(tt.userLux)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedLux, luxVersion)
		})
	}
}
