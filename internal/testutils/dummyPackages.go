// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package testutils

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/stretchr/testify/require"
)

const (
	nodeBin = "node"
	pluginDirName  = "plugins"
	evmBin         = "evm"
	buildDirName   = "build"
	subnetEVMBin   = "subnet-evm"
	readme         = "README.md"
	license        = "LICENSE"

	nodeBinPrefix = "node-"

	luxdTar     = "/tmp/luxd.tar.gz"
	luxdZip     = "/tmp/luxd.zip"
	subnetEVMTar = "/tmp/subevm.tar.gz"
)

var (
	evmBinary       = []byte{0x00, 0xe1, 0x40, 0x00}
	readmeContents  = []byte("README")
	licenseContents = []byte("LICENSE")
)

func verifyLuxdTarContents(require *require.Assertions, tarBytes []byte, version string) {
	topDir := nodeBinPrefix + version
	bin := filepath.Join(topDir, nodeBin)
	plugins := filepath.Join(topDir, pluginDirName)
	evm := filepath.Join(plugins, evmBin)

	binExists := false
	pluginsExists := false
	evmExists := false

	file := bytes.NewReader(tarBytes)
	gzRead, err := gzip.NewReader(file)
	require.NoError(err)
	tarReader := tar.NewReader(gzRead)
	require.NoError(err)
	for {
		file, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(err)
		switch file.Name {
		case topDir:
			// we don't need to check the top dir, it is implied through other checks
			continue
		case bin:
			binExists = true
		case plugins:
			pluginsExists = true
		case evm:
			evmExists = true
		default:
			require.FailNow("Tar has extra files")
		}
	}

	require.True(binExists)
	require.True(pluginsExists)
	require.True(evmExists)
}

func verifySubnetEVMTarContents(require *require.Assertions, tarBytes []byte) {
	binExists := false
	readmeExists := false
	licenseExists := false

	file := bytes.NewReader(tarBytes)
	gzRead, err := gzip.NewReader(file)
	require.NoError(err)
	tarReader := tar.NewReader(gzRead)
	require.NoError(err)
	for {
		file, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(err)
		switch file.Name {
		case subnetEVMBin:
			binExists = true
		case readme:
			readmeExists = true
		case license:
			licenseExists = true
		default:
			require.FailNow("Tar has extra files: " + file.Name)
		}
	}
	require.True(binExists)
	require.True(readmeExists)
	require.True(licenseExists)
}

func verifyLuxdZipContents(require *require.Assertions, zipFile string) {
	topDir := buildDirName
	bin := filepath.Join(topDir, nodeBin)
	plugins := filepath.Join(topDir, pluginDirName)
	evm := filepath.Join(plugins, evmBin)

	topDirExists := false
	binExists := false
	pluginsExists := false
	evmExists := false

	reader, err := zip.OpenReader(zipFile)
	require.NoError(err)
	defer reader.Close()
	for _, file := range reader.File {
		// Zip directories end in "/" which is annoying for string matching
		switch strings.TrimSuffix(file.Name, "/") {
		case topDir:
			topDirExists = true
		case bin:
			binExists = true
		case plugins:
			pluginsExists = true
		case evm:
			evmExists = true
		default:
			require.FailNow("Zip has extra files: " + file.Name)
		}
	}
	require.True(topDirExists)
	require.True(binExists)
	require.True(pluginsExists)
	require.True(evmExists)
}

func CreateDummyLuxdZip(require *require.Assertions, binary []byte) []byte {
	sourceDir, err := os.MkdirTemp(os.TempDir(), "binutils-source")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)

	topDir := filepath.Join(sourceDir, buildDirName)
	err = os.Mkdir(topDir, 0o700)
	require.NoError(err)

	binPath := filepath.Join(topDir, nodeBin)
	err = os.WriteFile(binPath, binary, 0o600)
	require.NoError(err)

	pluginDir := filepath.Join(topDir, pluginDirName)
	err = os.Mkdir(pluginDir, 0o700)
	require.NoError(err)

	evmBinPath := filepath.Join(pluginDir, evmBin)
	err = os.WriteFile(evmBinPath, evmBinary, 0o600)
	require.NoError(err)

	// Put into zip
	CreateZip(require, topDir, luxdZip)
	defer os.Remove(luxdZip)

	verifyLuxdZipContents(require, luxdZip)

	zipBytes, err := os.ReadFile(luxdZip)
	require.NoError(err)
	return zipBytes
}

func CreateDummyLuxdTar(require *require.Assertions, binary []byte, version string) []byte {
	sourceDir, err := os.MkdirTemp(os.TempDir(), "binutils-source")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)

	topDir := filepath.Join(sourceDir, nodeBinPrefix+version)
	err = os.Mkdir(topDir, 0o700)
	require.NoError(err)

	binPath := filepath.Join(topDir, nodeBin)
	err = os.WriteFile(binPath, binary, 0o600)
	require.NoError(err)

	pluginDir := filepath.Join(topDir, pluginDirName)
	err = os.Mkdir(pluginDir, 0o700)
	require.NoError(err)

	evmBinPath := filepath.Join(pluginDir, evmBin)
	err = os.WriteFile(evmBinPath, evmBinary, 0o600)
	require.NoError(err)

	// Put into tar
	CreateTarGz(require, topDir, luxdTar, true)
	defer os.Remove(luxdTar)
	tarBytes, err := os.ReadFile(luxdTar)
	require.NoError(err)
	verifyLuxdTarContents(require, tarBytes, version)
	return tarBytes
}

func CreateDummySubnetEVMTar(require *require.Assertions, binary []byte) []byte {
	sourceDir, err := os.MkdirTemp(os.TempDir(), "binutils-source")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)

	binPath := filepath.Join(sourceDir, subnetEVMBin)
	err = os.WriteFile(binPath, binary, 0o600)
	require.NoError(err)

	readmePath := filepath.Join(sourceDir, readme)
	err = os.WriteFile(readmePath, readmeContents, 0o600)
	require.NoError(err)

	licensePath := filepath.Join(sourceDir, license)
	err = os.WriteFile(licensePath, licenseContents, 0o600)
	require.NoError(err)

	// Put into tar
	CreateTarGz(require, sourceDir, subnetEVMTar, false)
	defer os.Remove(subnetEVMTar)
	tarBytes, err := os.ReadFile(subnetEVMTar)
	require.NoError(err)
	verifySubnetEVMTarContents(require, tarBytes)
	return tarBytes
}
