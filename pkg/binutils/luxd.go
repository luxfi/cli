// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
)

const (
	luxdBinPrefix = "luxd-"
)

func SetupLuxgo(app *application.Lux, luxdVersion string) (string, error) {
	binDir := app.GetLuxgoBinDir()

	installer := NewInstaller()
	downloader := NewLuxdDownloader()
	return InstallBinary(
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
}

// SetupLux is an alias for SetupLuxgo for backward compatibility
func SetupLux(app *application.Lux, luxdVersion string) (string, error) {
	return SetupLuxgo(app, luxdVersion)
}
