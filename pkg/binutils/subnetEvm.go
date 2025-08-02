// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"path/filepath"

	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/constants"
)

func SetupSubnetEVM(app *application.Lux, subnetEVMVersion string) (string, string, error) {
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
		constants.SubnetEVMRepoName,
		downloader,
		installer,
	)
	return subnetEVMVersion, filepath.Join(vmDir, constants.SubnetEVMBin), err
}
