// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package binutils

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/models"
)

func UpgradeVM(app *application.Lux, vmID string, vmBinPath string) error {
	installer := NewPluginBinaryDownloader(app)
	if err := installer.UpgradeVM(vmID, vmBinPath); err != nil {
		return fmt.Errorf("failed to upgrade vm: %w", err)
	}

	return nil
}

// update the RPC version of the VM in the sidecar file
func UpdateLocalSidecarRPC(app *application.Lux, sc models.Sidecar, rpcVersion int) error {
	// find local network deployment info in sidecar
	networkData, ok := sc.Networks[models.Local.String()]
	if !ok {
		return fmt.Errorf("failed to find local network in sidecar")
	}

	networkData.RPCVersion = rpcVersion

	sc.Networks[models.Local.String()] = networkData

	if err := app.UpdateSidecar(&sc); err != nil {
		return fmt.Errorf("failed to update sidecar: %w", err)
	}

	return nil
}
