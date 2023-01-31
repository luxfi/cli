// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxdefi/avalanche-cli/pkg/application"
	"github.com/luxdefi/avalanche-cli/pkg/constants"
)

func SetupNode(app *application.Lux, avagoVersion string) (string, error) {
	binDir := app.GetNodeBinDir()

	installer := NewInstaller()
	downloader := NewAvagoDownloader()
	return InstallBinary(
		app,
		avagoVersion,
		binDir,
		binDir,
		avalanchegoBinPrefix,
		constants.AvaLabsOrg,
		constants.NodeRepoName,
		downloader,
		installer,
	)
}
