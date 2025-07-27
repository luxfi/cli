// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

func SetupLuxgo(app *application.Lux, luxdVersion string) (string, string, error) {
	binDir := app.GetLuxgoBinDir()

	installer := NewInstaller()
	downloader := NewAvagoDownloader()
	return InstallBinary(
		app,
		luxdVersion,
		binDir,
		binDir,
		luxdBinPrefix,
		constants.LuxOrg,
		constants.LuxGoRepoName,
		"",
		downloader,
		installer,
	)
}
