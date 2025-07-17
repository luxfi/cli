// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

func SetupLux(app *application.Lux, luxVersion string) (string, error) {
	binDir := app.GetLuxBinDir()

	installer := NewInstaller()
	downloader := NewLuxDownloader()
	return InstallBinary(
		app,
		luxVersion,
		binDir,
		binDir,
		nodeBinPrefix,
		constants.AvaLabsOrg,
		constants.LuxRepoName,
		downloader,
		installer,
	)
}
