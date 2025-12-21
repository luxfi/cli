// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLockCmd(t *testing.T) {
	cmd := newLockCmd()

	assert.Equal(t, "lock [name]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check --all flag exists
	allFlag := cmd.Flags().Lookup("all")
	assert.NotNil(t, allFlag)
	assert.Equal(t, "a", allFlag.Shorthand)
}

func TestNewUnlockCmd(t *testing.T) {
	cmd := newUnlockCmd()

	assert.Equal(t, "unlock <name>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check --password flag exists
	passwordFlag := cmd.Flags().Lookup("password")
	assert.NotNil(t, passwordFlag)
	assert.Equal(t, "p", passwordFlag.Shorthand)

	// Check --timeout flag exists
	timeoutFlag := cmd.Flags().Lookup("timeout")
	assert.NotNil(t, timeoutFlag)
	assert.Equal(t, "t", timeoutFlag.Shorthand)
}

func TestNewBackendCmd(t *testing.T) {
	cmd := newBackendCmd()

	assert.Equal(t, "backend", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check subcommands exist
	listCmd, _, err := cmd.Find([]string{"list"})
	assert.NoError(t, err)
	assert.NotNil(t, listCmd)

	setCmd, _, err := cmd.Find([]string{"set"})
	assert.NoError(t, err)
	assert.NotNil(t, setCmd)

	infoCmd, _, err := cmd.Find([]string{"info"})
	assert.NoError(t, err)
	assert.NotNil(t, infoCmd)
}
