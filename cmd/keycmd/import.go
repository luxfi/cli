// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"
	"strings"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <name>",
		Short: "Import key set from mnemonic",
		Long: `Import a key set by recovering from a mnemonic phrase.

Derives all key types (EC, BLS, Ringtail, ML-DSA) from the mnemonic.

Example:
  lux key import validator1`,
		Args: cobra.ExactArgs(1),
		RunE: runImport,
	}

	return cmd
}

func runImport(_ *cobra.Command, args []string) error {
	name := args[0]

	// Check if key set already exists
	existing, err := key.ListKeySets()
	if err != nil {
		return fmt.Errorf("failed to list existing keys: %w", err)
	}
	for _, k := range existing {
		if k == name {
			return fmt.Errorf("key set '%s' already exists, use 'lux key delete %s' first", name, name)
		}
	}

	ux.Logger.PrintToUser("Enter your 24-word mnemonic phrase:")
	mnemonic, err := app.Prompt.CaptureString("Mnemonic")
	if err != nil {
		return err
	}

	mnemonic = strings.TrimSpace(mnemonic)
	if !key.ValidateMnemonic(mnemonic) {
		return fmt.Errorf("invalid mnemonic phrase")
	}

	ux.Logger.PrintToUser("Deriving keys from mnemonic...")

	keySet, err := key.DeriveAllKeys(name, mnemonic)
	if err != nil {
		return fmt.Errorf("failed to derive keys: %w", err)
	}

	if err := key.SaveKeySet(keySet); err != nil {
		return fmt.Errorf("failed to save keys: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Key set '%s' imported successfully!", name)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("EC Address: %s", keySet.ECAddress)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Use 'lux key show %s' to view all public keys.", name)

	return nil
}
