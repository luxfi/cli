// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"os"
	"testing"

	"github.com/luxfi/cli/internal/testutils"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
)

func TestCleanBins(t *testing.T) {
	require := require.New(t)
	ux.NewUserLog(luxlog.NewNoOpLogger(), os.Stdout)
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "bin-test")
	require.NoError(err)
	f2, err := os.CreateTemp(dir, "another-test")
	require.NoError(err)
	cleanBins(dir)
	require.NoFileExists(f.Name())
	require.NoFileExists(f2.Name())
	require.NoDirExists(dir)
}

func Test_removeLocalDeployInfoFromSidecars(t *testing.T) {
	app = testutils.SetupTestInTempDir(t)

	subnetName := "test1"

	localMap := make(map[string]models.NetworkData)

	localMap[models.Local.String()] = models.NetworkData{
		SubnetID:     ids.ID{1, 2, 3, 4},
		BlockchainID: ids.ID{1, 2, 3, 4},
	}

	sc := models.Sidecar{
		Name:     subnetName,
		Networks: localMap,
	}

	err := app.CreateSidecar(&sc)
	require.NoError(t, err)

	loadedSC, err := app.LoadSidecar(subnetName)
	require.NoError(t, err)
	require.Contains(t, loadedSC.Networks, models.Local.String())

	err = removeLocalDeployInfoFromSidecars()
	require.NoError(t, err)

	loadedSC, err = app.LoadSidecar(subnetName)
	require.NoError(t, err)
	require.NotContains(t, loadedSC.Networks, models.Local.String())
}
