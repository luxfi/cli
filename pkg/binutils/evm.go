// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/constants"
)

func SetupEVM(app *application.Lux, evmVersion string) (string, error) {
	binDir := app.GetEVMBinDir()

	installer := NewInstaller()
	downloader := NewEVMDownloader()
	binPath, err := InstallBinary(
		app,
		evmVersion,
		binDir,
		binDir,
		evmBinPrefix,
		constants.LuxOrg,
		constants.EVMRepoName,
		downloader,
		installer,
	)
	return binPath, err
}