// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import "github.com/luxfi/cli/pkg/application"

// SetupCustomBin returns the path for a custom VM binary.
func SetupCustomBin(app *application.Lux, chainName string) string {
	// Just need to get the path of the vm
	return app.GetCustomVMPath(chainName)
}

// SetupLPMBin returns the path for an LPM VM binary.
func SetupLPMBin(app *application.Lux, vmid string) string {
	// Just need to get the path of the vm
	return app.GetLPMVMPath(vmid)
}
