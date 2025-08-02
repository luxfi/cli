// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package testutils

import (
	"io"
	"testing"

	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/luxfi/cli/v2/pkg/config"
	"github.com/luxfi/cli/v2/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
)

func SetupTest(t *testing.T) *require.Assertions {
	// use io.Discard to not print anything
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)
	return require.New(t)
}

func SetupTestInTempDir(t *testing.T) *application.Lux {
	testDir := t.TempDir()

	app := application.New()
	app.Setup(testDir, luxlog.NewNoOpLogger(), &config.Config{}, nil, nil)
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)
	return app
}
