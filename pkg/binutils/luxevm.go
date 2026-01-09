// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/constants"
)

// SetupEVM downloads and installs the EVM binary.
func SetupEVM(app *application.Lux, evmVersion string) (string, error) {
	// Setup EVM for L1 or L2 chains
	binDir := filepath.Join(app.GetBaseDir(), constants.EVMInstallDir)
	subDir := filepath.Join(binDir, evmBinPrefix+evmVersion)

	installer := NewInstaller()
	downloader := NewEVMDownloader()
	vmDir, err := InstallBinary(
		app,
		evmVersion,
		binDir,
		subDir,
		evmBinPrefix,
		constants.LuxOrg,
		constants.EVMRepoName,
		downloader,
		installer,
	)
	if err != nil {
		return "", err
	}

	binaryPath := filepath.Join(vmDir, constants.EVMBin)

	// If the expected binary doesn't exist, look for platform-specific binary and symlink it
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// The release tarball contains binary with platform suffix (e.g., evm-linux-amd64)
		platformBinary := filepath.Join(vmDir, constants.EVMBin+"-"+runtime.GOOS+"-"+runtime.GOARCH)
		if _, err := os.Stat(platformBinary); err == nil {
			// Create symlink from platform-specific binary to expected name
			if err := os.Symlink(platformBinary, binaryPath); err != nil {
				return "", err
			}
		}
	}

	return binaryPath, nil
}
