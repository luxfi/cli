// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"errors"
	"testing"

	"github.com/luxdefi/cli/cmd/flags"
	"github.com/luxdefi/cli/internal/mocks"
	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/node/utils/logging"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testLuxdVersion1      = "v1.9.2"
	testLuxdVersion2      = "v1.9.1"
	testLatestLuxdVersion = "latest"
)

var testLuxdCompat = []byte("{\"19\": [\"v1.9.2\"],\"18\": [\"v1.9.1\"],\"17\": [\"v1.9.0\",\"v1.8.0\"]}")

func TestMutuallyExclusive(t *testing.T) {
	require := require.New(t)
	type test struct {
		flagA       bool
		flagB       bool
		flagC       bool
		expectError bool
	}

	tests := []test{
		{
			flagA:       false,
			flagB:       false,
			flagC:       false,
			expectError: false,
		},
		{
			flagA:       true,
			flagB:       false,
			flagC:       false,
			expectError: false,
		},
		{
			flagA:       false,
			flagB:       true,
			flagC:       false,
			expectError: false,
		},
		{
			flagA:       false,
			flagB:       false,
			flagC:       true,
			expectError: false,
		},
		{
			flagA:       true,
			flagB:       false,
			flagC:       true,
			expectError: true,
		},
		{
			flagA:       false,
			flagB:       true,
			flagC:       true,
			expectError: true,
		},
		{
			flagA:       true,
			flagB:       true,
			flagC:       false,
			expectError: true,
		},
		{
			flagA:       true,
			flagB:       true,
			flagC:       true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		isEx := flags.EnsureMutuallyExclusive([]bool{tt.flagA, tt.flagB, tt.flagC})
		if tt.expectError {
			require.False(isEx)
		} else {
			require.True(isEx)
		}
	}
}

func TestCheckForInvalidDeployAndSetLuxdVersion(t *testing.T) {
	type test struct {
		name            string
		networkRPC      int
		networkVersion  string
		networkErr      error
		networkUp       bool
		desiredRPC      int
		desiredVersion  string
		compatData      []byte
		expectError     bool
		expectedVersion string
		compatError     error
	}

	tests := []test{
		{
			name:            "network already running, rpc matches",
			networkRPC:      18,
			networkVersion:  testLuxdVersion1,
			networkErr:      nil,
			desiredRPC:      18,
			desiredVersion:  testLatestLuxdVersion,
			expectError:     false,
			expectedVersion: testLuxdVersion1,
			networkUp:       true,
		},
		{
			name:            "network already running, rpc mismatch",
			networkRPC:      18,
			networkVersion:  testLuxdVersion1,
			networkErr:      nil,
			desiredRPC:      19,
			desiredVersion:  testLatestLuxdVersion,
			expectError:     true,
			expectedVersion: "",
			networkUp:       true,
		},
		{
			name:            "network already running, version mismatch",
			networkRPC:      18,
			networkVersion:  testLuxdVersion1,
			networkErr:      nil,
			desiredRPC:      19,
			desiredVersion:  testLuxdVersion2,
			expectError:     true,
			expectedVersion: "",
			networkUp:       true,
		},
		{
			name:            "network stopped, no err",
			networkRPC:      0,
			networkVersion:  "",
			networkErr:      nil,
			desiredRPC:      19,
			desiredVersion:  testLatestLuxdVersion,
			expectError:     false,
			expectedVersion: testLuxdVersion1,
			compatData:      testLuxdCompat,
			compatError:     nil,
			networkUp:       false,
		},
		{
			name:            "network stopped, no compat",
			networkRPC:      0,
			networkVersion:  "",
			networkErr:      nil,
			desiredRPC:      19,
			desiredVersion:  testLatestLuxdVersion,
			expectError:     true,
			expectedVersion: testLuxdVersion1,
			compatData:      nil,
			compatError:     errors.New("no compat"),
			networkUp:       false,
		},
		{
			name:            "network up, network err",
			networkRPC:      0,
			networkVersion:  "",
			networkErr:      errors.New("unable to determine rpc version"),
			desiredRPC:      19,
			desiredVersion:  testLatestLuxdVersion,
			expectError:     true,
			expectedVersion: testLuxdVersion1,
			compatData:      testLuxdCompat,
			compatError:     nil,
			networkUp:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			mockSC := mocks.StatusChecker{}
			mockSC.On("GetCurrentNetworkVersion").Return(tt.networkVersion, tt.networkRPC, tt.networkUp, tt.networkErr)

			userProvidedLuxdVersion = tt.desiredVersion

			mockDownloader := &mocks.Downloader{}
			mockDownloader.On("Download", mock.Anything).Return(tt.compatData, nil)
			mockDownloader.On("GetLatestReleaseVersion", mock.Anything).Return(tt.expectedVersion, nil)

			app = application.New()
			app.Log = logging.NoLog{}
			app.Downloader = mockDownloader

			desiredLuxdVersion, err := CheckForInvalidDeployAndGetLuxdVersion(&mockSC, tt.desiredRPC)

			if tt.expectError {
				require.Error(err)
			} else {
				require.NoError(err)
				require.Equal(tt.expectedVersion, desiredLuxdVersion)
			}
		})
	}
}
