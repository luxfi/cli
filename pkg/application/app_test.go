// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package application

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/models"
	"github.com/stretchr/testify/require"
)

const (
	chainName1 = "TEST_chain"
	chainName2 = "TEST_copied_chain"
)

func TestUpdateSideCar(t *testing.T) {
	require := require.New(t)
	chainID := ids.GenerateTestID()
	sc := &models.Sidecar{
		Name:      "TEST",
		VM:        models.EVM,
		TokenName: "TEST",
		ChainID:   chainID,
	}

	ap := newTestApp(t)

	err := ap.CreateSidecar(sc)
	require.NoError(err)
	control, err := ap.LoadSidecar(sc.Name)
	require.NoError(err)
	require.Equal(*sc, control)
	sc.Networks = make(map[string]models.NetworkData)
	sc.Networks["local"] = models.NetworkData{
		BlockchainID: ids.GenerateTestID(),
		ChainID:      ids.GenerateTestID(),
	}

	err = ap.UpdateSidecar(sc)
	require.NoError(err)
	control, err = ap.LoadSidecar(sc.Name)
	require.NoError(err)
	require.Equal(*sc, control)
}

func Test_writeGenesisFile_success(t *testing.T) {
	require := require.New(t)
	genesisBytes := []byte("genesis")
	genesisFile := constants.GenesisFileName

	ap := newTestApp(t)
	// Write genesis
	err := ap.WriteGenesisFile(chainName1, genesisBytes)
	require.NoError(err)

	// Check file exists
	createdPath := filepath.Join(ap.GetChainsDir(), chainName1, genesisFile)
	_, err = os.Stat(createdPath)
	require.NoError(err)

	// Cleanup file
	err = os.Remove(createdPath)
	require.NoError(err)
}

func Test_copyGenesisFile_success(t *testing.T) {
	require := require.New(t)
	genesisBytes := []byte("genesis")

	ap := newTestApp(t)
	// Create original genesis
	err := ap.WriteGenesisFile(chainName1, genesisBytes)
	require.NoError(err)

	// Copy genesis
	createdGenesis := ap.GetGenesisPath(chainName1)
	err = ap.CopyGenesisFile(createdGenesis, chainName2)
	require.NoError(err)

	// Check copied file exists
	copiedGenesis := ap.GetGenesisPath(chainName2)
	_, err = os.Stat(copiedGenesis)
	require.NoError(err)

	// Cleanup files
	err = os.Remove(createdGenesis)
	require.NoError(err)
	err = os.Remove(copiedGenesis)
	require.NoError(err)
}

func Test_copyGenesisFile_failure(t *testing.T) {
	require := require.New(t)
	// copy genesis that doesn't exist

	ap := newTestApp(t)
	// Copy genesis
	createdGenesis := ap.GetGenesisPath(chainName1)
	err := ap.CopyGenesisFile(createdGenesis, chainName2)
	require.Error(err)

	// Check no copied file exists
	copiedGenesis := ap.GetGenesisPath(chainName2)
	_, err = os.Stat(copiedGenesis)
	require.Error(err)
}

func Test_createSidecar_success(t *testing.T) {
	type test struct {
		name              string
		chainName         string
		tokenName         string
		expectedTokenName string
		chainID           ids.ID
	}

	tests := []test{
		{
			name:              "Success",
			chainName:         chainName1,
			tokenName:         "TOKEN",
			expectedTokenName: "TOKEN",
			chainID:           ids.GenerateTestID(),
		},
		{
			name:              "no token name",
			chainName:         chainName1,
			tokenName:         "",
			expectedTokenName: "TEST",
			chainID:           ids.GenerateTestID(),
		},
	}

	ap := newTestApp(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			const vm = models.EVM

			sc := &models.Sidecar{
				Name:      tt.chainName,
				VM:        vm,
				TokenName: tt.tokenName,
				ChainID:   tt.chainID,
			}

			// Write sidecar
			err := ap.CreateSidecar(sc)
			require.NoError(err)

			// Check file exists
			createdPath := ap.GetSidecarPath(tt.chainName)
			_, err = os.Stat(createdPath)
			require.NoError(err)

			control, err := ap.LoadSidecar(tt.chainName)
			require.NoError(err)
			require.Equal(*sc, control)

			require.Equal(sc.TokenName, tt.expectedTokenName)

			// Cleanup file
			err = os.Remove(createdPath)
			require.NoError(err)
		})
	}
}

func Test_loadSidecar_success(t *testing.T) {
	require := require.New(t)
	const vm = models.EVM

	ap := newTestApp(t)

	// Write sidecar
	sidecarBytes := []byte("{  \"Name\": \"TEST_chain\",\n  \"VM\": \"Lux EVM\",\n  \"Chain\": \"TEST_chain\"\n  }")
	sidecarPath := ap.GetSidecarPath(chainName1)
	err := os.MkdirAll(filepath.Dir(sidecarPath), constants.DefaultPerms755)
	require.NoError(err)

	err = os.WriteFile(sidecarPath, sidecarBytes, 0o600)
	require.NoError(err)

	// Check file exists
	_, err = os.Stat(sidecarPath)
	require.NoError(err)

	// Check contents
	expectedSc := models.Sidecar{
		Name:      chainName1,
		VM:        vm,
		Chain:     chainName1,
		TokenName: constants.DefaultTokenName,
	}

	sc, err := ap.LoadSidecar(chainName1)
	require.NoError(err)
	require.Equal(sc, expectedSc)

	// Cleanup file
	err = os.Remove(sidecarPath)
	require.NoError(err)
}

func Test_loadSidecar_failure_notFound(t *testing.T) {
	require := require.New(t)

	ap := newTestApp(t)

	// Assert file doesn't exist at start
	sidecarPath := ap.GetSidecarPath(chainName1)
	_, err := os.Stat(sidecarPath)
	require.Error(err)

	_, err = ap.LoadSidecar(chainName1)
	require.Error(err)
}

func Test_loadSidecar_failure_malformed(t *testing.T) {
	require := require.New(t)

	ap := newTestApp(t)

	// Write sidecar
	sidecarBytes := []byte("bad_sidecar")
	sidecarPath := ap.GetSidecarPath(chainName1)
	err := os.MkdirAll(filepath.Dir(sidecarPath), constants.DefaultPerms755)
	require.NoError(err)

	err = os.WriteFile(sidecarPath, sidecarBytes, 0o600)
	require.NoError(err)

	// Check file exists
	_, err = os.Stat(sidecarPath)
	require.NoError(err)

	// Check contents
	_, err = ap.LoadSidecar(chainName1)
	require.Error(err)

	// Cleanup file
	err = os.Remove(sidecarPath)
	require.NoError(err)
}

func Test_genesisExists(t *testing.T) {
	require := require.New(t)

	ap := newTestApp(t)

	// Assert file doesn't exist at start
	result := ap.GenesisExists(chainName1)
	require.False(result)

	// Create genesis
	genesisPath := ap.GetGenesisPath(chainName1)
	genesisBytes := []byte("genesis")
	err := os.MkdirAll(filepath.Dir(genesisPath), constants.DefaultPerms755)
	require.NoError(err)
	err = os.WriteFile(genesisPath, genesisBytes, 0o600)
	require.NoError(err)

	// Verify genesis exists
	result = ap.GenesisExists(chainName1)
	require.True(result)

	// Clean up created genesis
	err = os.Remove(genesisPath)
	require.NoError(err)
}

func newTestApp(t *testing.T) *Lux {
	tempDir := t.TempDir()
	app := New()
	app.Setup(tempDir, luxlog.NewNoOpLogger(), nil, nil, nil)
	return app
}
