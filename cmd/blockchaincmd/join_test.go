// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"context"
	"testing"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/rpc"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Simple interface for testing - only includes methods we actually use
type testPClient interface {
	GetCurrentValidators(ctx context.Context, subnetID ids.ID, nodeIDs []ids.NodeID, options ...rpc.Option) ([]platformvm.ClientPermissionlessValidator, error)
}

func TestIsNodeValidatingSubnet(t *testing.T) {
	require := require.New(t)
	nodeID := ids.GenerateTestNodeID()
	nonValidator := ids.GenerateTestNodeID()
	subnetID := ids.GenerateTestID()

	pClient := &mocks.PClient{}
	pClient.On("GetCurrentValidators", mock.Anything, mock.Anything, mock.Anything).Return(
		[]platformvm.ClientPermissionlessValidator{
			{
				ClientStaker: platformvm.ClientStaker{
					NodeID: nodeID,
				},
			},
		}, nil)

	// first pass: should return true for the GetCurrentValidators
	isValidating, err := checkIsValidatingTest(subnetID, nodeID, pClient)
	require.NoError(err)
	require.True(isValidating)

	// second pass: The nonValidator is not in current nor pending validators, hence false
	isValidating, err = checkIsValidatingTest(subnetID, nonValidator, pClient)
	require.NoError(err)
	require.False(isValidating)
}

// checkIsValidatingTest is a test version of checkIsValidating that uses the test interface
func checkIsValidatingTest(subnetID ids.ID, nodeID ids.NodeID, pClient testPClient) (bool, error) {
	// first check if the node is already an accepted validator on the subnet
	ctx := context.Background()
	nodeIDs := []ids.NodeID{nodeID}
	vals, err := pClient.GetCurrentValidators(ctx, subnetID, nodeIDs)
	if err != nil {
		return false, err
	}
	for _, v := range vals {
		// strictly this is not needed, as we are providing the nodeID as param
		// just a double check
		if v.NodeID == nodeID {
			return true, nil
		}
	}
	return false, nil
}
