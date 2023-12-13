// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/cli/pkg/constants"
)

func SetupLuxd(app *application.Lux, luxdVersion string) (string, error) {
	binDir := app.GetLuxdBinDir()

	installer := NewInstaller()
	downloader := NewLuxdDownloader()
	return InstallBinary(
		app,
		luxdVersion,
		binDir,
		binDir,
		nodeBinPrefix,
		constants.LuxDeFiOrg,
		constants.LuxdRepoName,
		downloader,
		installer,
	)
}
