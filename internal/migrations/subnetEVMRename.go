// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package migrations

import (
	"os"
	"path/filepath"

	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/cli/pkg/models"
)

const oldSubnetEVM = "SubnetEVM"

func migrateSubnetEVMNames(app *application.Lux, runner *migrationRunner) error {
	subnetDir := app.GetSubnetDir()
	subnets, err := os.ReadDir(subnetDir)
	if err != nil {
		return err
	}

	for _, subnet := range subnets {
		// disregard any empty subnet directories
		dirContents, err := os.ReadDir(filepath.Join(subnetDir, subnet.Name()))
		if err != nil {
			return err
		}
		if len(dirContents) == 0 {
			continue
		}

		sc, err := app.LoadSidecar(subnet.Name())
		if err != nil {
			return err
		}

		if string(sc.VM) == oldSubnetEVM {
			runner.printMigrationMessage()
			sc.VM = models.SubnetEvm
			if err = app.UpdateSidecar(&sc); err != nil {
				return err
			}
		}
	}
	return nil
}
