// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/cli/internal/testutils"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/prompts"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	version1 = "v1.17.1"
	version2 = "v1.18.1"

	nodeBin = "node"
)

var (
	binary1 = []byte{0xde, 0xad, 0xbe, 0xef}
	binary2 = []byte{0xfe, 0xed, 0xc0, 0xde}
)

func setupInstallDir(require *require.Assertions) *application.Lux {
	rootDir, err := os.MkdirTemp(os.TempDir(), "binutils-tests")
	require.NoError(err)
	defer os.RemoveAll(rootDir)

	app := application.New()
	app.Setup(rootDir, luxlog.NewNoOpLogger(), &config.Config{}, prompts.NewPrompter(), application.NewDownloader())
	return app
}

func Test_installLuxWithVersion_Zip(t *testing.T) {
	require := testutils.SetupTest(t)

	zipBytes := testutils.CreateDummyLuxZip(require, binary1)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "darwin")

	githubDownloader := NewLuxDownloader()

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", mock.Anything).Return(zipBytes, nil)
	app.Downloader = &mockAppDownloader

	expectedDir := filepath.Join(app.GetLuxBinDir(), nodeBinPrefix+version1)

	binDir, err := installBinaryWithVersion(app, version1, app.GetLuxBinDir(), nodeBinPrefix, githubDownloader, mockInstaller)
	require.Equal(expectedDir, binDir)
	require.NoError(err)

	// Check the installed binary
	installedBin, err := os.ReadFile(filepath.Join(binDir, nodeBin))
	require.NoError(err)
	require.Equal(binary1, installedBin)
}

func Test_installLuxWithVersion_Tar(t *testing.T) {
	require := testutils.SetupTest(t)

	tarBytes := testutils.CreateDummyLuxTar(require, binary1, version1)

	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "linux")

	downloader := NewLuxDownloader()

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", mock.Anything).Return(tarBytes, nil)
	app.Downloader = &mockAppDownloader

	expectedDir := filepath.Join(app.GetLuxBinDir(), nodeBinPrefix+version1)

	binDir, err := installBinaryWithVersion(app, version1, app.GetLuxBinDir(), nodeBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir, binDir)
	require.NoError(err)

	// Check the installed binary
	installedBin, err := os.ReadFile(filepath.Join(binDir, nodeBin))
	require.NoError(err)
	require.Equal(binary1, installedBin)
}

func Test_installLuxWithVersion_MultipleCoinstalls(t *testing.T) {
	require := testutils.SetupTest(t)

	zipBytes1 := testutils.CreateDummyLuxZip(require, binary1)
	zipBytes2 := testutils.CreateDummyLuxZip(require, binary2)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "darwin")

	downloader := NewLuxDownloader()
	url1, _, err := downloader.GetDownloadURL(version1, mockInstaller)
	require.NoError(err)
	url2, _, err := downloader.GetDownloadURL(version2, mockInstaller)
	require.NoError(err)
	mockInstaller.On("DownloadRelease", url1).Return(zipBytes1, nil)
	mockInstaller.On("DownloadRelease", url2).Return(zipBytes2, nil)

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", url1).Return(zipBytes1, nil)
	mockAppDownloader.On("Download", url2).Return(zipBytes2, nil)
	app.Downloader = &mockAppDownloader

	expectedDir1 := filepath.Join(app.GetLuxBinDir(), nodeBinPrefix+version1)
	expectedDir2 := filepath.Join(app.GetLuxBinDir(), nodeBinPrefix+version2)

	binDir1, err := installBinaryWithVersion(app, version1, app.GetLuxBinDir(), nodeBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir1, binDir1)
	require.NoError(err)

	binDir2, err := installBinaryWithVersion(app, version2, app.GetLuxBinDir(), nodeBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir2, binDir2)
	require.NoError(err)

	require.NotEqual(binDir1, binDir2)

	// Check the installed binary
	installedBin1, err := os.ReadFile(filepath.Join(binDir1, nodeBin))
	require.NoError(err)
	require.Equal(binary1, installedBin1)

	installedBin2, err := os.ReadFile(filepath.Join(binDir2, nodeBin))
	require.NoError(err)
	require.Equal(binary2, installedBin2)
}

func Test_installEVMWithVersion(t *testing.T) {
	require := testutils.SetupTest(t)

	tarBytes := testutils.CreateDummyEVMTar(require, binary1)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("amd64", "darwin")

	downloader := NewEVMDownloader()

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", mock.Anything).Return(tarBytes, nil)
	app.Downloader = &mockAppDownloader

	expectedDir := filepath.Join(app.GetEVMBinDir(), subnetEVMBinPrefix+version1)

	subDir := filepath.Join(app.GetEVMBinDir(), subnetEVMBinPrefix+version1)

	binDir, err := installBinaryWithVersion(app, version1, subDir, subnetEVMBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir, binDir)
	require.NoError(err)

	// Check the installed binary
	installedBin, err := os.ReadFile(filepath.Join(binDir, constants.EVMBin))
	require.NoError(err)
	require.Equal(binary1, installedBin)
}

func Test_installEVMWithVersion_MultipleCoinstalls(t *testing.T) {
	require := testutils.SetupTest(t)

	tarBytes1 := testutils.CreateDummyEVMTar(require, binary1)
	tarBytes2 := testutils.CreateDummyEVMTar(require, binary2)
	app := setupInstallDir(require)

	mockInstaller := &mocks.Installer{}
	mockInstaller.On("GetArch").Return("arm64", "linux")

	downloader := NewEVMDownloader()
	url1, _, err := downloader.GetDownloadURL(version1, mockInstaller)
	require.NoError(err)
	url2, _, err := downloader.GetDownloadURL(version2, mockInstaller)
	require.NoError(err)

	mockAppDownloader := mocks.Downloader{}
	mockAppDownloader.On("Download", url1).Return(tarBytes1, nil)
	mockAppDownloader.On("Download", url2).Return(tarBytes2, nil)
	app.Downloader = &mockAppDownloader

	expectedDir1 := filepath.Join(app.GetEVMBinDir(), subnetEVMBinPrefix+version1)
	expectedDir2 := filepath.Join(app.GetEVMBinDir(), subnetEVMBinPrefix+version2)

	subDir1 := filepath.Join(app.GetEVMBinDir(), subnetEVMBinPrefix+version1)
	subDir2 := filepath.Join(app.GetEVMBinDir(), subnetEVMBinPrefix+version2)

	binDir1, err := installBinaryWithVersion(app, version1, subDir1, subnetEVMBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir1, binDir1)
	require.NoError(err)

	binDir2, err := installBinaryWithVersion(app, version2, subDir2, subnetEVMBinPrefix, downloader, mockInstaller)
	require.Equal(expectedDir2, binDir2)
	require.NoError(err)

	require.NotEqual(binDir1, binDir2)

	// Check the installed binary
	installedBin1, err := os.ReadFile(filepath.Join(binDir1, constants.EVMBin))
	require.NoError(err)
	require.Equal(binary1, installedBin1)

	installedBin2, err := os.ReadFile(filepath.Join(binDir2, constants.EVMBin))
	require.NoError(err)
	require.Equal(binary2, installedBin2)
}
