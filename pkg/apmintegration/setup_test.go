// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/v2/pkg/constants"
	"github.com/stretchr/testify/require"
)

func TestSetupLPM(t *testing.T) {
	require := require.New(t)
	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	err := os.MkdirAll(filepath.Dir(app.GetLPMLog()), constants.DefaultPerms755)
	require.NoError(err)

	err = SetupApm(app, testDir)
	require.NoError(err)
	require.NotEqual(nil, app.Apm)
	require.Equal(testDir, app.ApmDir)
}
