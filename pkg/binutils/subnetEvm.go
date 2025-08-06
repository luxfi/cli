// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

func SetupSubnetEVM(app *application.Lux, subnetEVMVersion string) (string, string, error) {
	// Check if already installed
	binDir := filepath.Join(app.GetBaseDir(), constants.EVMInstallDir)
	subDir := filepath.Join(binDir, subnetEVMBinPrefix+subnetEVMVersion)

	installer := NewInstaller()
	downloader := NewEVMDownloader() // Use the existing EVM downloader
	vmDir, err := InstallBinary(
		app,
		subnetEVMVersion,
		binDir,
		subDir,
		subnetEVMBinPrefix,
		constants.LuxOrg,
		constants.EVMRepoName,
		downloader,
		installer,
	)
	if err != nil {
		return "", "", err
	}
	return subnetEVMVersion, filepath.Join(vmDir, constants.EVMBin), nil
}
