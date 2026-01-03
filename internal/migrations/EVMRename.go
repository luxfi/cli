// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package migrations

import (
	"os"
	"path/filepath"

	"github.com/luxfi/sdk/models"

	"github.com/luxfi/cli/pkg/application"
)

const oldEVM = "EVM"

func migrateEVMNames(app *application.Lux, runner *migrationRunner) error {
	subnetDir := app.GetChainsDir()
	subnets, err := os.ReadDir(subnetDir)
	if err != nil {
		return err
	}

	for _, subnet := range subnets {
		if !subnet.IsDir() {
			continue
		}
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

		if string(sc.VM) == oldEVM {
			runner.printMigrationMessage()
			sc.VM = models.EVM
			if err = app.UpdateSidecar(&sc); err != nil {
				return err
			}
		}
	}
	return nil
}
