// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/models"
)

// CreateCustomSidecar creates a sidecar for custom VMs
func CreateCustomSidecar(
	sc *models.Sidecar,
	app *application.Lux,
	blockchainName string,
	vmVersion string,
) (*models.Sidecar, error) {
	// If sc is nil, create a new sidecar
	if sc == nil {
		sc = &models.Sidecar{
			Version: "1.0.0",
		}
	}

	// Always set Name and Subnet from blockchainName
	sc.Name = blockchainName
	sc.Chain = blockchainName

	// Update sidecar with custom VM information
	sc.VM = models.CustomVM
	sc.VMVersion = vmVersion

	return sc, nil
}
