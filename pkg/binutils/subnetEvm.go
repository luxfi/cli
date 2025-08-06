// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

func SetupEVM(app *application.Lux, evmVersion string) (string, error) {
	// Setup EVM for L1 or L2 chains
	binDir := filepath.Join(app.GetBaseDir(), constants.EVMInstallDir)
	subDir := filepath.Join(binDir, subnetEVMBinPrefix+evmVersion)

	installer := NewInstaller()
	downloader := NewEVMDownloader()
	vmDir, err := InstallBinary(
		app,
		evmVersion,
		binDir,
		subDir,
		subnetEVMBinPrefix,
		constants.LuxOrg,
		constants.EVMRepoName,
		downloader,
		installer,
	)
	if err != nil {
		return "", err
	}
	return filepath.Join(vmDir, constants.EVMBin), nil
}
