// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package migrations

import (
	"bytes"
	"errors"
	"testing"

	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/config"
	"github.com/luxfi/cli/v2/v2/pkg/prompts"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
)

func TestRunMigrations(t *testing.T) {
	buffer := make([]byte, 0, 100)
	bufWriter := bytes.NewBuffer(buffer)
	ux.NewUserLog(luxlog.NewNoOpLogger(), bufWriter)
	require := require.New(t)
	testDir := t.TempDir()

	app := &application.Lux{}
	app.Setup(testDir, luxlog.NewNoOpLogger(), config.New(), prompts.NewPrompter(), application.NewDownloader())

	type migTest struct {
		migs           map[int]migrationFunc
		name           string
		shouldErr      bool
		expectedOutput string
	}

	expectedIfRan := runMessage + "\n" + endMessage + "\n"
	expectedIfFailed := runMessage + "\n" + failedEndMessage + "\n"

	tests := []migTest{
		{
			name:           "no migrations",
			shouldErr:      false,
			migs:           map[int]migrationFunc{},
			expectedOutput: "",
		},
		{
			name:      "migration fail",
			shouldErr: true,
			migs: map[int]migrationFunc{
				0: func(app *application.Lux, r *migrationRunner) error {
					return errors.New("bogus fail")
				},
			},
			expectedOutput: "",
		},
		{
			name:      "1 mig, apply",
			shouldErr: false,
			migs: map[int]migrationFunc{
				0: func(app *application.Lux, r *migrationRunner) error {
					r.printMigrationMessage()
					return nil
				},
			},
			expectedOutput: expectedIfRan,
		},
		{
			name:      "2 mig, apply both",
			shouldErr: false,
			migs: map[int]migrationFunc{
				0: func(app *application.Lux, r *migrationRunner) error {
					r.printMigrationMessage()
					return nil
				},
				1: func(app *application.Lux, r *migrationRunner) error {
					r.printMigrationMessage()
					return nil
				},
			},
			expectedOutput: expectedIfRan,
		},
		{
			name:      "2 mig, apply 1",
			shouldErr: false,
			migs: map[int]migrationFunc{
				0: func(app *application.Lux, r *migrationRunner) error {
					return nil
				},
				1: func(app *application.Lux, r *migrationRunner) error {
					r.printMigrationMessage()
					return nil
				},
			},
			expectedOutput: expectedIfRan,
		},
		{
			name:      "2 mig, first one fails",
			shouldErr: true,
			migs: map[int]migrationFunc{
				0: func(app *application.Lux, r *migrationRunner) error {
					return errors.New("bogus fail")
				},
				1: func(app *application.Lux, r *migrationRunner) error {
					r.printMigrationMessage()
					return nil
				},
			},
			expectedOutput: "",
		},
		{
			name:      "2 mig, apply 1, second one fails",
			shouldErr: true,
			migs: map[int]migrationFunc{
				0: func(app *application.Lux, r *migrationRunner) error {
					r.printMigrationMessage()
					return nil
				},
				1: func(app *application.Lux, r *migrationRunner) error {
					return errors.New("bogus fail")
				},
			},
			expectedOutput: expectedIfFailed,
		},
	}

	for _, tt := range tests {
		// reset the buffer on each run to match expected output
		bufWriter.Reset()
		runner := &migrationRunner{
			showMsg:    true,
			running:    false,
			migrations: tt.migs,
		}
		err := runner.run(app)
		if tt.shouldErr {
			require.Error(err)
		} else {
			require.NoError(err)
		}
		require.Equal(tt.expectedOutput, bufWriter.String())
	}
}
