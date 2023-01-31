// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"path/filepath"

	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/cli/pkg/constants"
)

func SetupSpacesVM(app *application.Lux, spacesVMVersion string) (string, error) {
	// Check if already installed
	binDir := app.GetSpacesVMBinDir()
	subDir := filepath.Join(binDir, spacesVMBinPrefix+spacesVMVersion)

	installer := NewInstaller()
	downloader := NewSpacesVMDownloader()
	vmDir, err := InstallBinary(
		app,
		spacesVMVersion,
		binDir,
		subDir,
		spacesVMBinPrefix,
		constants.AvaLabsOrg,
		constants.SpacesVMRepoName,
		downloader,
		installer,
	)
	return filepath.Join(vmDir, constants.SpacesVMBin), err
}
