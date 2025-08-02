// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package remoteconfig

import (
	"github.com/luxfi/cli/v2/pkg/utils"
)

func LokiFoldersToCreate() []string {
	return []string{utils.GetRemoteComposeServicePath("loki", "data")}
}
