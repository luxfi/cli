// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package testutils

import (
	"io"
	"testing"

	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/cli/pkg/config"
	"github.com/luxdefi/cli/pkg/ux"
	"github.com/luxdefi/luxgo/utils/logging"
	"github.com/stretchr/testify/require"
)

func SetupTest(t *testing.T) *require.Assertions {
	// use io.Discard to not print anything
	ux.NewUserLog(logging.NoLog{}, io.Discard)
	return require.New(t)
}

func SetupTestInTempDir(t *testing.T) *application.Lux {
	testDir := t.TempDir()

	app := application.New()
	app.Setup(testDir, logging.NoLog{}, &config.Config{}, nil, nil)
	ux.NewUserLog(logging.NoLog{}, io.Discard)
	return app
}
