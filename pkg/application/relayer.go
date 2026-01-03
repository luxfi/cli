// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package application

import (
	"path/filepath"
)

// GetWarpRelayerServiceConfigPath returns the path to the warp relayer service config
func (app *Lux) GetWarpRelayerServiceConfigPath(blockchainName string) string {
	return filepath.Join(app.GetBaseDir(), "services", "warp-relayer", blockchainName+".yml")
}
