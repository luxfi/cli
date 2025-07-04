// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"io"
	"testing"
	"time"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/json"
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/node/vms/platformvm/api"
	"github.com/olekukonko/tablewriter"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStats(t *testing.T) {
	require := require.New(t)

	ux.NewUserLog(logging.NoLog{}, io.Discard)

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

	rows, err := buildCurrentValidatorStats(pClient, iClient, table, subnetID)
	table.Append(rows[0])

	require.NoError(err)
	require.Equal(1, table.NumLines())
	require.Equal(localNodeID.String(), rows[0][0])
	require.Equal("true", rows[0][1])
	require.Equal("42", rows[0][2])
	require.Equal(remaining, rows[0][3])
	require.Equal(expectedVerStr, rows[0][4])

	pendingV := make([]interface{}, 1)

	jweight := json.Uint64(weight)

	pendingV[0] = api.PermissionlessValidator{
		Staker: api.Staker{
			StartTime: json.Uint64(uint64(startTime.Unix())),
			EndTime:   json.Uint64(uint64(endTime.Unix())),
			NodeID:    localNodeID,
			Weight:    jweight,
		},
	}

	pClient.On("GetPendingValidators", mock.Anything, mock.Anything, mock.Anything).Return(pendingV, nil, nil)

	table = tablewriter.NewWriter(io.Discard)
	rows, err = buildPendingValidatorStats(pClient, iClient, table, subnetID)
	table.Append(rows[0])

	// we can't use `startTime` resp. `endTime` for controlling the end string:
	// both are time.Now(), which contains nanosecond information
	// we need to cut off nanoseconds, and just use seconds,
	// as that is how the API returns the information too.
	// Unix() calls return seconds only
	controlStartTime := time.Unix(startTime.Unix(), 0)
	controlEndTime := time.Unix(endTime.Unix(), 0)

	require.NoError(err)
	require.Equal(1, table.NumLines())
	require.Equal(localNodeID.String(), rows[0][0])
	require.Equal("42", rows[0][1])
	require.Equal(controlStartTime.Local().String(), rows[0][2])
	require.Equal(controlEndTime.Local().String(), rows[0][3])
	require.Equal(expectedVerStr, rows[0][4])
}
