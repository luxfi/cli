// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package migrations

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"github.com/stretchr/testify/require"
)

func TestEVMRenameMigration(t *testing.T) {
	type test struct {
		name       string
		sc         *models.Sidecar
		expectedVM string
	}

	subnetName := "test"

	tests := []test{
		{
			name: "Convert EVM",
			sc: &models.Sidecar{
				Name: subnetName,
				VM:   "EVM",
			},
			expectedVM: "Lux EVM",
		},
		{
			name: "Preserve Lux EVM",
			sc: &models.Sidecar{
				Name: subnetName,
				VM:   "Lux EVM",
			},
			expectedVM: "Lux EVM",
		},
		{
			name: "Ignore unknown",
			sc: &models.Sidecar{
				Name: subnetName,
				VM:   "unknown",
			},
			expectedVM: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)
			require := require.New(t)
			testDir := t.TempDir()

			app := &application.Lux{}
			app.Setup(testDir, luxlog.NewNoOpLogger(), config.New(), prompts.NewPrompter(), application.NewDownloader())

			err := app.CreateSidecar(tt.sc)
			require.NoError(err)

			runner := migrationRunner{
				showMsg: true,
				running: false,
				migrations: map[int]migrationFunc{
					0: migrateEVMNames,
				},
			}
			// run the migration
			err = runner.run(app)
			require.NoError(err)

			loadedSC, err := app.LoadSidecar(tt.sc.Name)
			require.NoError(err)
			require.Equal(tt.expectedVM, string(loadedSC.VM))
		})
	}
}

func TestEVMRenameMigration_EmptyDir(t *testing.T) {
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)
	require := require.New(t)
	testDir := t.TempDir()

	app := &application.Lux{}
	app.Setup(testDir, luxlog.NewNoOpLogger(), config.New(), prompts.NewPrompter(), application.NewDownloader())

	emptySubnetName := "emptySubnet"

	subnetDir := filepath.Join(app.GetSubnetDir(), emptySubnetName)
	err := os.MkdirAll(subnetDir, constants.DefaultPerms755)
	require.NoError(err)

	runner := migrationRunner{
		showMsg: true,
		running: false,
		migrations: map[int]migrationFunc{
			0: migrateEVMNames,
		},
	}
	// run the migration
	err = runner.run(app)
	require.NoError(err)
}
