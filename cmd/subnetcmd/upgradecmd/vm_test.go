// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package upgradecmd

import (
	"os"
	"testing"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/utils/logging"
	"github.com/stretchr/testify/require"
)

func TestAtMostOneNetworkSelected(t *testing.T) {
	assert := require.New(t)

	type test struct {
		name       string
		useConfig  bool
		useLocal   bool
		useTestnet    bool
		useMainnet bool
		valid      bool
	}

	tests := []test{
		{
			name:       "all false",
			useConfig:  false,
			useLocal:   false,
			useTestnet:    false,
			useMainnet: false,
			valid:      true,
		},
		{
			name:       "future true",
			useConfig:  true,
			useLocal:   false,
			useTestnet:    false,
			useMainnet: false,
			valid:      true,
		},
		{
			name:       "local true",
			useConfig:  false,
			useLocal:   true,
			useTestnet:    false,
			useMainnet: false,
			valid:      true,
		},
		{
			name:       "testnet true",
			useConfig:  false,
			useLocal:   false,
			useTestnet:    true,
			useMainnet: false,
			valid:      true,
		},
		{
			name:       "mainnet true",
			useConfig:  false,
			useLocal:   false,
			useTestnet:    false,
			useMainnet: true,
			valid:      true,
		},
		{
			name:       "double true 1",
			useConfig:  true,
			useLocal:   true,
			useTestnet:    false,
			useMainnet: false,
			valid:      false,
		},
		{
			name:       "double true 2",
			useConfig:  true,
			useLocal:   false,
			useTestnet:    true,
			useMainnet: false,
			valid:      false,
		},
		{
			name:       "double true 3",
			useConfig:  true,
			useLocal:   false,
			useTestnet:    false,
			useMainnet: true,
			valid:      false,
		},
		{
			name:       "double true 4",
			useConfig:  false,
			useLocal:   true,
			useTestnet:    true,
			useMainnet: false,
			valid:      false,
		},
		{
			name:       "double true 5",
			useConfig:  false,
			useLocal:   true,
			useTestnet:    false,
			useMainnet: true,
			valid:      false,
		},
		{
			name:       "double true 6",
			useConfig:  false,
			useLocal:   false,
			useTestnet:    true,
			useMainnet: true,
			valid:      false,
		},
		{
			name:       "all true",
			useConfig:  true,
			useLocal:   true,
			useTestnet:    true,
			useMainnet: true,
			valid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useConfig = tt.useConfig
			useLocal = tt.useLocal
			useTestnet = tt.useTestnet
			useMainnet = tt.useMainnet

			accepted := atMostOneNetworkSelected()
			if tt.valid {
				assert.True(accepted)
			} else {
				assert.False(accepted)
			}
		})
	}
}

func TestAtMostOneVersionSelected(t *testing.T) {
	assert := require.New(t)

	type test struct {
		name      string
		useLatest bool
		version   string
		binary    string
		valid     bool
	}

	tests := []test{
		{
			name:      "all empty",
			useLatest: false,
			version:   "",
			binary:    "",
			valid:     true,
		},
		{
			name:      "one selected 1",
			useLatest: true,
			version:   "",
			binary:    "",
			valid:     true,
		},
		{
			name:      "one selected 2",
			useLatest: false,
			version:   "v1.2.0",
			binary:    "",
			valid:     true,
		},
		{
			name:      "one selected 3",
			useLatest: false,
			version:   "",
			binary:    "home",
			valid:     true,
		},
		{
			name:      "two selected 1",
			useLatest: true,
			version:   "v1.2.0",
			binary:    "",
			valid:     false,
		},
		{
			name:      "two selected 2",
			useLatest: true,
			version:   "",
			binary:    "home",
			valid:     false,
		},
		{
			name:      "two selected 3",
			useLatest: false,
			version:   "v1.2.0",
			binary:    "home",
			valid:     false,
		},
		{
			name:      "all selected",
			useLatest: true,
			version:   "v1.2.0",
			binary:    "home",
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useLatest = tt.useLatest
			targetVersion = tt.version
			binaryPathArg = tt.binary

			accepted := atMostOneVersionSelected()
			if tt.valid {
				assert.True(accepted)
			} else {
				assert.False(accepted)
			}
		})
	}
}

func TestAtMostOneAutomationSelected(t *testing.T) {
	assert := require.New(t)

	type test struct {
		name      string
		useManual bool
		pluginDir string
		valid     bool
	}

	tests := []test{
		{
			name:      "all empty",
			useManual: false,
			pluginDir: "",
			valid:     true,
		},
		{
			name:      "manual selected",
			useManual: true,
			pluginDir: "",
			valid:     true,
		},
		{
			name:      "auto selected",
			useManual: false,
			pluginDir: "home",
			valid:     true,
		},
		{
			name:      "both selected",
			useManual: true,
			pluginDir: "home",
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useManual = tt.useManual
			pluginDir = tt.pluginDir

			accepted := atMostOneAutomationSelected()
			if tt.valid {
				assert.True(accepted)
			} else {
				assert.False(accepted)
			}
		})
	}
}

func TestUpdateToCustomBin(t *testing.T) {
	assert := require.New(t)
	testDir := t.TempDir()

	subnetName := "testSubnet"
	sc := models.Sidecar{
		Name:       subnetName,
		VM:         models.EVM,
		VMVersion:  "v3.0.0",
		RPCVersion: 20,
		Subnet:     subnetName,
	}
	networkToUpgrade := futureDeployment

	factory := logging.NewFactory(logging.Config{})
	log, err := factory.Make("lux")
	assert.NoError(err)

	// create the user facing logger as a global var
	ux.NewUserLog(log, os.Stdout)

	app = &application.Lux{}
	app.Setup(testDir, log, config.New(), prompts.NewPrompter(), application.NewDownloader())

	err = os.MkdirAll(app.GetSubnetDir(), constants.DefaultPerms755)
	assert.NoError(err)

	err = app.CreateSidecar(&sc)
	assert.NoError(err)

	err = os.MkdirAll(app.GetCustomVMDir(), constants.DefaultPerms755)
	assert.NoError(err)

	binaryPath := "../../../tests/assets/dummyVmBinary.bin"

	assert.FileExists(binaryPath)

	err = updateToCustomBin(sc, networkToUpgrade, binaryPath)
	assert.NoError(err)

	// check new binary exists and matches
	placedBinaryPath := app.GetCustomVMPath(subnetName)
	assert.FileExists(placedBinaryPath)
	expectedHash, err := utils.GetSHA256FromDisk(binaryPath)
	assert.NoError(err)

	actualHash, err := utils.GetSHA256FromDisk(placedBinaryPath)
	assert.NoError(err)

	assert.Equal(expectedHash, actualHash)

	// check sidecar
	diskSC, err := app.LoadSidecar(subnetName)
	assert.NoError(err)
	assert.Equal(models.VMTypeFromString(models.CustomVM), diskSC.VM)
	assert.Empty(diskSC.VMVersion)
}
