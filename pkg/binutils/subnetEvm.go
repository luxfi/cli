// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

func SetupEVM(app *application.Lux, subnetEVMVersion string) (string, error) {
	// Check if already installed
	binDir := app.GetEVMBinDir()
	subDir := filepath.Join(binDir, subnetEVMBinPrefix+subnetEVMVersion)

	installer := NewInstaller()
	downloader := NewEVMDownloader()
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
	return filepath.Join(vmDir, constants.EVMBin), err
}
