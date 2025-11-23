// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/cli/tests/e2e/utils"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestExportImportSubnet(t *testing.T) {
	testDir := t.TempDir()
	require := require.New(t)
	testSubnet := "testSubnet"
	vmVersion := "v0.9.99"
	testEVMCompat := []byte("{\"rpcChainVMProtocolVersion\": {\"v0.9.99\": 18}}")

	app = application.New()

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", mock.Anything).Return(testEVMCompat, nil)

	app.Setup(testDir, luxlog.NewNoOpLogger(), nil, prompts.NewPrompter(), &mockAppDownloader)
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)
	genBytes, sc, err := vm.CreateEvmConfig(app, testSubnet, "../../"+utils.SubnetEvmGenesisPath, vmVersion)
	require.NoError(err)
	err = app.WriteGenesisFile(testSubnet, genBytes)
	require.NoError(err)
	err = app.CreateSidecar(sc)
	require.NoError(err)

	exportOutputDir := filepath.Join(testDir, "output")
	err = os.MkdirAll(exportOutputDir, constants.DefaultPerms755)
	require.NoError(err)
	exportOutput = filepath.Join(exportOutputDir, testSubnet)
	defer func() {
		exportOutput = ""
		app = nil
	}()

	err = exportSubnet(nil, []string{"this-does-not-exist-should-fail"})
	require.Error(err)

	err = exportSubnet(nil, []string{testSubnet})
	require.NoError(err)
	require.FileExists(exportOutput)
	sidecarFile := filepath.Join(app.GetBaseDir(), constants.SubnetDir, testSubnet, constants.SidecarFileName)
	orig, err := os.ReadFile(sidecarFile)
	require.NoError(err)

	var control map[string]interface{}
	err = json.Unmarshal(orig, &control)
	require.NoError(err)
	require.Equal(testSubnet, control["Name"])
	require.Equal("Lux EVM", control["VM"])
	require.Equal(vmVersion, control["VMVersion"])
	require.Equal(testSubnet, control["Subnet"])
	require.Equal("TEST", control["TokenName"])
	require.Equal(constants.SidecarVersion, control["Version"])
	require.Equal(nil, control["Networks"])

	err = os.Remove(sidecarFile)
	require.NoError(err)

	err = importSubnet(nil, []string{"this-does-also-not-exist-import-should-fail"})
	require.ErrorIs(err, os.ErrNotExist)
	err = importSubnet(nil, []string{exportOutput})
	require.ErrorContains(err, "subnet already exists")
	genFile := filepath.Join(app.GetBaseDir(), constants.SubnetDir, testSubnet, constants.GenesisFileName)
	err = os.Remove(genFile)
	require.NoError(err)
	err = importSubnet(nil, []string{exportOutput})
	require.NoError(err)
}
