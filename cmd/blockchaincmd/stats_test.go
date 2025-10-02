// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/utils/rpc"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/node/vms/platformvm/signer"
	"github.com/olekukonko/tablewriter"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test interfaces with only the methods we need
type testStatsPC interface {
	GetCurrentValidators(ctx context.Context, subnetID ids.ID, nodeIDs []ids.NodeID, options ...rpc.Option) ([]platformvm.ClientPermissionlessValidator, error)
}

type testStatsIC interface {
	GetNodeID(ctx context.Context, options ...rpc.Option) (ids.NodeID, *signer.ProofOfPossession, error)
	GetNodeVersion(ctx context.Context, options ...rpc.Option) (*info.GetNodeVersionReply, error)
}

func TestStats(t *testing.T) {
	require := require.New(t)

	ux.NewUserLog(luxlog.NoLog{}, io.Discard)

	pClient := &mocks.PClient{}
	iClient := &mocks.InfoClient{}

	localNodeID := ids.GenerateTestNodeID()
	subnetID := ids.GenerateTestID()

	startTime := time.Now()
	endTime := time.Now()
	weight := uint64(42)
	conn := true

	remaining := ux.FormatDuration(endTime.Sub(startTime))

	reply := []platformvm.ClientPermissionlessValidator{
		{
			ClientStaker: platformvm.ClientStaker{
				StartTime: uint64(startTime.Unix()),
				EndTime:   uint64(endTime.Unix()),
				NodeID:    localNodeID,
				Weight:    weight,
			},
			Connected: &conn,
		},
	}

	pClient.On("GetCurrentValidators", mock.Anything, mock.Anything, mock.Anything).Return(reply, nil)
	iClient.On("GetNodeID", mock.Anything).Return(localNodeID, nil, nil)
	iClient.On("GetNodeVersion", mock.Anything).Return(&info.GetNodeVersionReply{
		VMVersions: map[string]string{
			subnetID.String(): "0.1.23",
		},
	}, nil)

	table := tablewriter.NewWriter(io.Discard)

	expectedVerStr := subnetID.String() + ": 0.1.23\n"

	rows, err := buildCurrentValidatorStatsTest(pClient, iClient, table, subnetID)
	table.Append(rows[0])

	require.NoError(err)
	require.Len(rows, 1) // Check we have 1 row instead of using NumLines
	require.Equal(localNodeID.String(), rows[0][0])
	require.Equal("true", rows[0][1])
	require.Equal("42", rows[0][2])
	require.Equal(remaining, rows[0][3])
	require.Equal(expectedVerStr, rows[0][4])
}

// Test version of buildCurrentValidatorStats that uses test interfaces
func buildCurrentValidatorStatsTest(pClient testStatsPC, infoClient testStatsIC, table *tablewriter.Table, subnetID ids.ID) ([][]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	currValidators, err := pClient.GetCurrentValidators(ctx, subnetID, []ids.NodeID{})
	if err != nil {
		return nil, err
	}

	rows := [][]string{}

	var (
		startTime, endTime           time.Time
		localNodeID                  ids.NodeID
		remaining, connected, weight string
		localVersionStr, versionStr  string
	)

	// try querying the local node for its node version
	reply, err := infoClient.GetNodeVersion(ctx)
	if err == nil {
		// we can ignore err here; if it worked, we have a non-zero node ID
		localNodeID, _, _ = infoClient.GetNodeID(ctx)
		for k, v := range reply.VMVersions {
			localVersionStr = k + ": " + v + "\n"
		}
	}

	for _, v := range currValidators {
		startTime = time.Unix(int64(v.StartTime), 0)
		endTime = time.Unix(int64(v.EndTime), 0)
		remaining = ux.FormatDuration(endTime.Sub(startTime))

		// some members of the returned object are pointers
		// so we need to check the pointer is actually valid
		if v.Connected != nil {
			connected = "true"
			if !*v.Connected {
				connected = "false"
			}
		} else {
			connected = "N/A"
		}

		weight = "42"

		// if retrieval of localNodeID failed, it will be empty,
		// and this comparison fails
		if v.NodeID == localNodeID {
			versionStr = localVersionStr
		}
		// query peers for IP address of this NodeID...
		rows = append(rows, []string{
			v.NodeID.String(),
			connected,
			weight,
			remaining,
			versionStr,
		})
	}

	return rows, nil
}
