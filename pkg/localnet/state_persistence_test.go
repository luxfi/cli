// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/prompts"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
)

func setupTestApp(t *testing.T) (*application.Lux, string) {
	app := application.New()
	appDir, err := os.MkdirTemp(os.TempDir(), "cli-state-test")
	require.NoError(t, err)
	app.Setup(appDir, luxlog.NewNoOpLogger(), config.New(), prompts.NewPrompter(), application.NewDownloader())
	return app, appDir
}

func TestNetworkStateDataSerialization(t *testing.T) {
	data := NetworkStateData{
		TrackedSubnets: []string{"subnet1", "subnet2"},
		DevMode:        true,
		Subnets: []SubnetStateInfo{
			{SubnetID: "sub1", BlockchainID: "bc1", VMID: "vm1", Name: "test"},
		},
		Validators: []ValidatorStateInfo{
			{NodeID: "node1", SubnetID: "sub1", Weight: 1000, StartTime: 1000000, EndTime: 2000000},
		},
		NetworkID:   1337,
		LastSavedAt: "1234567890",
	}

	bs, err := json.Marshal(&data)
	require.NoError(t, err)

	var decoded NetworkStateData
	err = json.Unmarshal(bs, &decoded)
	require.NoError(t, err)

	require.Equal(t, data.TrackedSubnets, decoded.TrackedSubnets)
	require.Equal(t, data.DevMode, decoded.DevMode)
	require.Len(t, decoded.Subnets, 1)
	require.Equal(t, "sub1", decoded.Subnets[0].SubnetID)
	require.Len(t, decoded.Validators, 1)
	require.Equal(t, "node1", decoded.Validators[0].NodeID)
	require.Equal(t, uint64(1000), decoded.Validators[0].Weight)
	require.Equal(t, uint32(1337), decoded.NetworkID)
}

func TestAddTrackedSubnet(t *testing.T) {
	app, appDir := setupTestApp(t)
	defer os.RemoveAll(appDir)

	networkDir := filepath.Join(appDir, "network")
	err := os.MkdirAll(networkDir, constants.DefaultPerms755)
	require.NoError(t, err)

	err = SaveLocalNetworkMeta(app, networkDir)
	require.NoError(t, err)

	err = AddTrackedSubnet(app, "subnet-123")
	require.NoError(t, err)

	subnets, err := GetTrackedSubnets(app)
	require.NoError(t, err)
	require.Contains(t, subnets, "subnet-123")

	err = AddTrackedSubnet(app, "subnet-123")
	require.NoError(t, err)

	subnets, err = GetTrackedSubnets(app)
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	err = AddTrackedSubnet(app, "subnet-456")
	require.NoError(t, err)

	subnets, err = GetTrackedSubnets(app)
	require.NoError(t, err)
	require.Len(t, subnets, 2)
}

func TestRemoveTrackedSubnet(t *testing.T) {
	app, appDir := setupTestApp(t)
	defer os.RemoveAll(appDir)

	networkDir := filepath.Join(appDir, "network")
	err := os.MkdirAll(networkDir, constants.DefaultPerms755)
	require.NoError(t, err)

	err = SaveLocalNetworkMeta(app, networkDir)
	require.NoError(t, err)

	err = AddTrackedSubnet(app, "subnet-123")
	require.NoError(t, err)
	err = AddTrackedSubnet(app, "subnet-456")
	require.NoError(t, err)

	err = RemoveTrackedSubnet(app, "subnet-123")
	require.NoError(t, err)

	subnets, err := GetTrackedSubnets(app)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Contains(t, subnets, "subnet-456")
}

func TestDevModeToggle(t *testing.T) {
	app, appDir := setupTestApp(t)
	defer os.RemoveAll(appDir)

	networkDir := filepath.Join(appDir, "network")
	err := os.MkdirAll(networkDir, constants.DefaultPerms755)
	require.NoError(t, err)

	err = SaveLocalNetworkMeta(app, networkDir)
	require.NoError(t, err)

	enabled, err := IsDevModeEnabled(app)
	require.NoError(t, err)
	require.False(t, enabled)

	err = SetDevMode(app, true)
	require.NoError(t, err)

	enabled, err = IsDevModeEnabled(app)
	require.NoError(t, err)
	require.True(t, enabled)

	err = SetDevMode(app, false)
	require.NoError(t, err)

	enabled, err = IsDevModeEnabled(app)
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestClearNetworkState(t *testing.T) {
	app, appDir := setupTestApp(t)
	defer os.RemoveAll(appDir)

	networkDir := filepath.Join(appDir, "network")
	err := os.MkdirAll(networkDir, constants.DefaultPerms755)
	require.NoError(t, err)

	err = SaveLocalNetworkMeta(app, networkDir)
	require.NoError(t, err)

	stateData := NetworkStateData{TrackedSubnets: []string{"subnet1"}}
	bs, err := json.Marshal(&stateData)
	require.NoError(t, err)
	statePath := filepath.Join(networkDir, networkStateFilename)
	err = os.WriteFile(statePath, bs, constants.WriteReadReadPerms)
	require.NoError(t, err)

	_, err = os.Stat(statePath)
	require.NoError(t, err)

	err = ClearNetworkState(app)
	require.NoError(t, err)

	_, err = os.Stat(statePath)
	require.True(t, os.IsNotExist(err))
}
