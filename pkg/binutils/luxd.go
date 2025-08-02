// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/constants"
)

func SetupLuxgo(app *application.Lux, luxdVersion string) (string, string, error) {
	binDir := app.GetLuxBinDir()

	installer := NewInstaller()
	downloader := NewLuxDownloader()
	binPath, err := InstallBinary(
		app,
		luxdVersion,
		binDir,
		binDir,
		luxdBinPrefix,
		constants.LuxOrg,
		constants.LuxdRepoName,
		downloader,
		installer,
	)
	return luxdVersion, binPath, err
}
