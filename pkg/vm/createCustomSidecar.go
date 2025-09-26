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
	// Update sidecar with custom VM information
	sc.VM = models.CustomVM
	sc.VMVersion = vmVersion
	
	return sc, nil
}