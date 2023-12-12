// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxdefi/cli/pkg/constants"
	"github.com/stretchr/testify/require"
)

func TestSetupLPM(t *testing.T) {
	require := require.New(t)
	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	err := os.MkdirAll(filepath.Dir(app.GetLPMLog()), constants.DefaultPerms755)
	require.NoError(err)

	err = SetupLpm(app, testDir)
	require.NoError(err)
	require.NotEqual(nil, app.Lpm)
	require.Equal(testDir, app.LpmDir)
}
