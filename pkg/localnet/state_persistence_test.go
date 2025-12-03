// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetworkStateDataSerialization(t *testing.T) {
	data := NetworkStateData{
		TrackedSubnets: []string{"subnet1", "subnet2"},
		DevMode:        true,
		Subnets: []SubnetStateInfo{
			{
				SubnetID:     "test-subnet-id",
				BlockchainID: "test-blockchain-id",
				VMID:         "test-vm-id",
				Name:         "testnet",
			},
		},
		Validators: []ValidatorStateInfo{
			{
				NodeID:    "NodeID-test123",
				SubnetID:  "test-subnet-id",
				Weight:    1000,
				StartTime: 1000000,
				EndTime:   2000000,
			},
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
	require.Equal(t, data.Subnets[0].SubnetID, decoded.Subnets[0].SubnetID)
	require.Len(t, decoded.Validators, 1)
	require.Equal(t, data.Validators[0].NodeID, decoded.Validators[0].NodeID)
	require.Equal(t, data.NetworkID, decoded.NetworkID)
}

func TestGetPersistedTrackingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	subnets, devMode, err := getPersistedTrackingConfig(tmpDir)
	require.NoError(t, err)
	require.Nil(t, subnets)
	require.False(t, devMode)

	stateData := NetworkStateData{
		TrackedSubnets: []string{"test-subnet-1", "test-subnet-2"},
		DevMode:        true,
	}
	statePath := filepath.Join(tmpDir, networkStateFilename)
	bs, err := json.Marshal(&stateData)
	require.NoError(t, err)
	err = os.WriteFile(statePath, bs, 0644)
	require.NoError(t, err)

	subnets, devMode, err = getPersistedTrackingConfig(tmpDir)
	require.NoError(t, err)
	require.Len(t, subnets, 2)
	require.True(t, devMode)
}

func TestSubnetStateInfo(t *testing.T) {
	info := SubnetStateInfo{
		SubnetID:     "subnet-abc",
		BlockchainID: "blockchain-xyz",
		VMID:         "vm-123",
		Name:         "my-subnet",
	}

	bs, err := json.Marshal(&info)
	require.NoError(t, err)

	var decoded SubnetStateInfo
	err = json.Unmarshal(bs, &decoded)
	require.NoError(t, err)

	require.Equal(t, info.SubnetID, decoded.SubnetID)
	require.Equal(t, info.BlockchainID, decoded.BlockchainID)
	require.Equal(t, info.VMID, decoded.VMID)
	require.Equal(t, info.Name, decoded.Name)
}

func TestValidatorStateInfo(t *testing.T) {
	info := ValidatorStateInfo{
		NodeID:    "NodeID-abc123",
		SubnetID:  "subnet-xyz",
		Weight:    2000,
		StartTime: 1000000000,
		EndTime:   2000000000,
	}

	bs, err := json.Marshal(&info)
	require.NoError(t, err)

	var decoded ValidatorStateInfo
	err = json.Unmarshal(bs, &decoded)
	require.NoError(t, err)

	require.Equal(t, info.NodeID, decoded.NodeID)
	require.Equal(t, info.SubnetID, decoded.SubnetID)
	require.Equal(t, info.Weight, decoded.Weight)
	require.Equal(t, info.StartTime, decoded.StartTime)
	require.Equal(t, info.EndTime, decoded.EndTime)
}
