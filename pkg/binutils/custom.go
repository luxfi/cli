// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import "github.com/luxdefi/cli/pkg/application"

func SetupCustomBin(app *application.Lux, subnetName string) string {
	// Just need to get the path of the vm
	return app.GetCustomVMPath(subnetName)
}

func SetupLPMBin(app *application.Lux, vmid string) string {
	// Just need to get the path of the vm
	return app.GetLPMVMPath(vmid)
}
