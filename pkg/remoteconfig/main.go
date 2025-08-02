// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package remoteconfig

import (
	"embed"

	"github.com/luxfi/cli/v2/v2/pkg/utils"
)

//go:embed templates/*
var templates embed.FS

// RemoteFoldersToCreateMonitoring returns a list of folders that need to be created on the remote Monitoring server
func RemoteFoldersToCreateMonitoring() []string {
	return utils.AppendSlices[string](
		GrafanaFoldersToCreate(),
		LokiFoldersToCreate(),
		PrometheusFoldersToCreate(),
		PromtailFoldersToCreate(),
	)
}

// RemoteFoldersToCreateLuxgo returns a list of folders that need to be created on the remote Luxgo server
func RemoteFoldersToCreateLuxgo() []string {
	return utils.AppendSlices[string](
		LuxFolderToCreate(),
		PromtailFoldersToCreate(),
	)
}
