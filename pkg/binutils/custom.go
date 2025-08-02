// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import "github.com/luxfi/cli/v2/v2/pkg/application"

func SetupCustomBin(app *application.Lux, subnetName string) string {
	// Just need to get the path of the vm
	return app.GetCustomVMPath(subnetName)
}

func SetupLPMBin(app *application.Lux, vmid string) string {
	// Just need to get the path of the vm
	return app.GetLPMVMPath(vmid)
}
