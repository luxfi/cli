// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	unlockPassword string
	unlockTimeout  time.Duration
)

func newUnlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock <name>",
		Short: "Unlock a key for use",
		Long: `Unlock a key by providing the password.

The key remains unlocked for the session duration (default 15 minutes).
After the timeout, the key is automatically locked and requires
re-authentication.

Password can be provided via:
  --password flag
  LUX_KEY_PASSWORD environment variable
  Interactive prompt (most secure)

Examples:
  lux key unlock validator1                    # Prompts for password
  lux key unlock validator1 --timeout 30m      # 30 minute session
  lux key unlock validator1 --password secret  # Password via flag (less secure)`,
		Args: cobra.ExactArgs(1),
		RunE: runUnlock,
	}

	cmd.Flags().StringVarP(&unlockPassword, "password", "p", "", "Password for the key")
	cmd.Flags().DurationVarP(&unlockTimeout, "timeout", "t", key.SessionTimeout, "Session timeout duration")

	return cmd
}

func runUnlock(_ *cobra.Command, args []string) error {
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

	// Check if already unlocked
	if !key.IsKeyLocked(name) {
		ux.Logger.PrintToUser("Key '%s' is already unlocked.", name)
		return nil
	}

	// Get password
	password := unlockPassword
	if password == "" {
		password = key.GetPasswordFromEnv()
	}
	if password == "" {
		// Prompt for password - requires interactive mode
		if !prompts.IsInteractive() {
			return fmt.Errorf("password required: use --password or set LUX_KEY_PASSWORD environment variable")
		}
		var err error
		password, err = app.Prompt.CaptureString("Password")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
	}

	if password == "" {
		return fmt.Errorf("password required")
	}

	// Unlock the key
	if err := key.UnlockKey(name, password); err != nil {
		if err == key.ErrInvalidPassword {
			return fmt.Errorf("invalid password")
		}
		return fmt.Errorf("failed to unlock key: %w", err)
	}

	ux.Logger.PrintToUser("Key '%s' unlocked (session expires in %s).", name, unlockTimeout)
	return nil
}
