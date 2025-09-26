// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/models"
)

// CreateEvmSidecar creates a sidecar for EVM-based blockchains
func CreateEvmSidecar(
	sc *models.Sidecar,
	app *application.Lux,
	blockchainName string,
	vmVersion string,
	tokenSymbol string,
	deployInterop bool,
	sovereign bool,
	useACP99 bool,
) (*models.Sidecar, error) {
	// Update sidecar with EVM-specific information
	sc.VM = models.EVM
	sc.VMVersion = vmVersion
	sc.TokenSymbol = tokenSymbol
	sc.Sovereign = sovereign
	sc.UseACP99 = useACP99
	
	return sc, nil
}