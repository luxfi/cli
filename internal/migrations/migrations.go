// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package migrations

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
)

type migrationFunc func(*application.Lux, *migrationRunner) error

type migrationRunner struct {
	showMsg    bool
	running    bool
	migrations map[int]migrationFunc
}

var (
	runMessage       = "The tool needs to apply some internal updates first..."
	endMessage       = "Update process successfully completed"
	failedEndMessage = "Some updates succeeded - others failed. Check output for hints"
)

// poor-man's migrations: there are no rollbacks (for now)
func RunMigrations(app *application.Lux) error {
	runner := &migrationRunner{
		showMsg: true,
		migrations: map[int]migrationFunc{
			// add new migrations here in rising index order
			// next one is 2
			0: migrateTopLevelFiles,
			1: migrateEVMNames,
		},
	}
	return runner.run(app)
}

func (m *migrationRunner) run(app *application.Lux) error {
	// by using an int index we can sort of "enforce" an order
	// with just an array it could easily happen that someone
	// prepends a new migration at the front instead of the bottom
	for i := 0; i < len(m.migrations); i++ {
		err := m.migrations[i](app, m)
		if err != nil {
			if m.running {
				ux.Logger.PrintToUser("%s", failedEndMessage)
			}
			return fmt.Errorf("migration #%d failed: %w", i, err)
		}
	}
	if m.running {
		ux.Logger.PrintToUser("%s", endMessage)
		m.running = false
	}
	return nil
}

// Every migration should first check if there are migrations to be run.
// If yes, should run this function to print a message only once
func (m *migrationRunner) printMigrationMessage() {
	if m.showMsg {
		ux.Logger.PrintToUser("%s", runMessage)
	}
	m.showMsg = false
	m.running = true
}
