// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var lockAll bool

func newLockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock [name]",
		Short: "Lock a key (clear from memory session)",
		Long: `Lock a key to clear it from the memory session.

A locked key requires password authentication to use again.
This is a security measure to protect keys when not in use.

Examples:
  lux key lock validator1    # Lock a specific key
  lux key lock --all         # Lock all keys`,
		Args: cobra.MaximumNArgs(1),
		RunE: runLock,
	}

	cmd.Flags().BoolVarP(&lockAll, "all", "a", false, "Lock all keys")

	return cmd
}

func runLock(_ *cobra.Command, args []string) error {
	if lockAll {
		return lockAllKeys()
	}

	if len(args) == 0 {
		return fmt.Errorf("key name required (or use --all to lock all keys)")
	}

	name := args[0]

	// Verify key exists
	keys, err := key.ListKeySets()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	found := false
	for _, k := range keys {
		if k == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("key '%s' not found", name)
	}

	// Check if already locked
	if key.IsKeyLocked(name) {
		ux.Logger.PrintToUser("Key '%s' is already locked.", name)
		return nil
	}

	// Lock the key
	if err := key.LockKey(name); err != nil {
		return fmt.Errorf("failed to lock key: %w", err)
	}

	ux.Logger.PrintToUser("Key '%s' locked.", name)
	return nil
}

func lockAllKeys() error {
	keys, err := key.ListKeySets()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		ux.Logger.PrintToUser("No keys found.")
		return nil
	}

	key.LockAllKeys()

	ux.Logger.PrintToUser("All keys locked (%d keys).", len(keys))
	return nil
}
