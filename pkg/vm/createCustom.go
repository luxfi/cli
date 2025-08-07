// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package vm

import (
	"os"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/ux"
)

func CreateCustomSubnetConfig(app *application.Lux, subnetName string, genesisPath, vmPath string) ([]byte, *models.Sidecar, error) {
	ux.Logger.PrintToUser("creating custom VM subnet %s", subnetName)

	genesisBytes, err := loadCustomGenesis(app, genesisPath)
	if err != nil {
		return nil, &models.Sidecar{}, err
	}

	sc := &models.Sidecar{
		Name:      subnetName,
		VM:        models.CustomVM,
		Subnet:    subnetName,
		TokenName: "",
	}

	err = CopyCustomVM(app, subnetName, vmPath)

	return genesisBytes, sc, err
}

func loadCustomGenesis(app *application.Lux, genesisPath string) ([]byte, error) {
	var err error
	if genesisPath == "" {
		genesisPath, err = app.Prompt.CaptureExistingFilepath("Enter path to custom genesis")
		if err != nil {
			return nil, err
		}
	}

	genesisBytes, err := os.ReadFile(genesisPath)
	return genesisBytes, err
}

func CopyCustomVM(app *application.Lux, subnetName string, vmPath string) error {
	var err error
	if vmPath == "" {
		vmPath, err = app.Prompt.CaptureExistingFilepath("Enter path to vm binary")
		if err != nil {
			return err
		}
	}

	return app.CopyVMBinary(vmPath, subnetName)
}
