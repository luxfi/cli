// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package plugins

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/internal/testutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/sdk/models"

	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
)

const (
	chainName1 = "TEST_chain"
	chainName2 = "TEST_copied_chain"

	chainID   = "testSubNet"
	networkID = uint32(67443)
)

// testing backward compatibility
func TestEditConfigFileWithOldPattern(t *testing.T) {
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)

	require := require.New(t)

	ap := testutils.SetupTestInTempDir(t)

	genesisBytes := []byte("genesis")
	err := ap.WriteGenesisFile(chainName1, genesisBytes)
	require.NoError(err)

	configFile := constants.NodeFileName

	// Create ConfigFile
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, configFile)
	defer func() { _ = os.Remove(configPath) }()

	// testing backward compatibility
	configBytes := []byte("{\"whitelisted-chains\": \"subNetId000\"}")
	err = os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755)
	require.NoError(err)
	err = os.WriteFile(configPath, configBytes, 0o600)
	require.NoError(err)

	err = EditConfigFile(ap, chainID, models.NetworkFromNetworkID(networkID), configPath, true, "")
	require.NoError(err)

	fileBytes, err := os.ReadFile(configPath) //nolint:gosec // G304: Test utility
	require.NoError(err)

	var luxConfig map[string]interface{}
	err = json.Unmarshal(fileBytes, &luxConfig)
	require.NoError(err)

	require.Equal("subNetId000,testSubNet", luxConfig["track-chains"])

	// ensure that the old setting has been deleted
	require.Equal(nil, luxConfig["whitelisted-chains"])
}

// testing backward compatibility
func TestEditConfigFileWithNewPattern(t *testing.T) {
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)

	require := require.New(t)

	ap := testutils.SetupTestInTempDir(t)

	genesisBytes := []byte("genesis")
	err := ap.WriteGenesisFile(chainName1, genesisBytes)
	require.NoError(err)

	configFile := constants.NodeFileName

	// Create ConfigFile
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, configFile)
	defer func() { _ = os.Remove(configPath) }()

	// testing backward compatibility
	configBytes := []byte("{\"track-chains\": \"subNetId000\"}")
	err = os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755)
	require.NoError(err)
	err = os.WriteFile(configPath, configBytes, 0o600)
	require.NoError(err)

	err = EditConfigFile(ap, chainID, models.NetworkFromNetworkID(networkID), configPath, true, "")
	require.NoError(err)

	fileBytes, err := os.ReadFile(configPath) //nolint:gosec // G304: Test utility
	require.NoError(err)

	var luxConfig map[string]interface{}
	err = json.Unmarshal(fileBytes, &luxConfig)
	require.NoError(err)

	require.Equal("subNetId000,testSubNet", luxConfig["track-chains"])

	// ensure that the old setting wont be applied at all
	require.Equal(nil, luxConfig["whitelisted-chains"])
}

func TestEditConfigFileWithNoSettings(t *testing.T) {
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)

	require := require.New(t)

	ap := testutils.SetupTestInTempDir(t)

	genesisBytes := []byte("genesis")
	err := ap.WriteGenesisFile(chainName1, genesisBytes)
	require.NoError(err)

	configFile := constants.NodeFileName

	// Create ConfigFile
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, configFile)
	defer func() { _ = os.Remove(configPath) }()

	// testing when no setting for tracked chains exists
	configBytes := []byte("{\"networkId\": \"5\"}")
	err = os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755)
	require.NoError(err)
	err = os.WriteFile(configPath, configBytes, 0o600)
	require.NoError(err)

	err = EditConfigFile(ap, chainID, models.NetworkFromNetworkID(networkID), configPath, true, "")
	require.NoError(err)

	fileBytes, err := os.ReadFile(configPath) //nolint:gosec // G304: Test utility
	require.NoError(err)

	var luxConfig map[string]interface{}
	err = json.Unmarshal(fileBytes, &luxConfig)
	require.NoError(err)

	require.Equal("testSubNet", luxConfig["track-chains"])

	// ensure that the old setting wont be applied at all
	require.Equal(nil, luxConfig["whitelisted-chains"])
}
