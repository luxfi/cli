// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package remoteconfig

import "github.com/luxfi/cli/pkg/utils"

func PrometheusFoldersToCreate() []string {
	return []string{
		utils.GetRemoteComposeServicePath("prometheus"),
		utils.GetRemoteComposeServicePath("prometheus", "data"),
	}
}
