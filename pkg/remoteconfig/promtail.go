// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package remoteconfig

import "github.com/luxfi/cli/pkg/utils"

func PromtailFoldersToCreate() []string {
	return []string{
		utils.GetRemoteComposeServicePath("promtail"),
		"/home/ubuntu/.luxd/logs",
	}
}
