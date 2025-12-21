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

var (
	useMnemonic    bool
	mnemonicPhrase string
	accountIndex   uint32
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new key set",
		Long: `Create a new key set with all cryptographic key types.

Generates a BIP39 mnemonic phrase and derives:
- EC (secp256k1) key for transactions
- BLS key for consensus
- Ringtail key for ring signatures
- ML-DSA key for post-quantum signatures

Keys are stored in ~/.lux/keys/<name>/

Examples:
  lux key create validator1                           # Generate new mnemonic
  lux key create validator1 --mnemonic                # Prompt for existing mnemonic
  lux key create validator1 --phrase "word1 word2..." # Use provided mnemonic
  lux key create mainnet-key-01 --phrase "$LUX_MNEMONIC" --account 1  # Derive account 1`,
		Args: cobra.ExactArgs(1),
		RunE: runCreate,
	}

	cmd.Flags().BoolVarP(&useMnemonic, "mnemonic", "m", false, "Import from existing mnemonic (prompts for input)")
	cmd.Flags().StringVar(&mnemonicPhrase, "phrase", "", "Mnemonic phrase to import (12 or 24 words)")
	cmd.Flags().Uint32Var(&accountIndex, "account", 0, "Account index for HD derivation (0-based)")

	return cmd
}

func runCreate(_ *cobra.Command, args []string) error {
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

	var mnemonic string

	if mnemonicPhrase != "" {
		// Use provided mnemonic
		mnemonic = strings.TrimSpace(mnemonicPhrase)
		if !key.ValidateMnemonic(mnemonic) {
			return fmt.Errorf("invalid mnemonic phrase")
		}
		ux.Logger.PrintToUser("Using provided mnemonic phrase")
	} else if useMnemonic {
		// Prompt for mnemonic
		ux.Logger.PrintToUser("Enter your 24-word mnemonic phrase:")
		var err error
		mnemonic, err = app.Prompt.CaptureString("Mnemonic")
		if err != nil {
			return err
		}
		mnemonic = strings.TrimSpace(mnemonic)
		if !key.ValidateMnemonic(mnemonic) {
			return fmt.Errorf("invalid mnemonic phrase")
		}
	} else {
		// Generate new mnemonic
		var err error
		mnemonic, err = key.GenerateMnemonic()
		if err != nil {
			return fmt.Errorf("failed to generate mnemonic: %w", err)
		}

		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Generated new mnemonic phrase (SAVE THIS SECURELY!):")
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("  %s", mnemonic)
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("WARNING: This is the ONLY time you will see this mnemonic!")
		ux.Logger.PrintToUser("Store it safely - it can recover all your keys.")
		ux.Logger.PrintToUser("")
	}

	// Derive all keys from mnemonic with account index
	if accountIndex > 0 {
		ux.Logger.PrintToUser("Deriving keys from mnemonic (account %d)...", accountIndex)
	} else {
		ux.Logger.PrintToUser("Deriving keys from mnemonic...")
	}

	keySet, err := key.DeriveAllKeysWithAccount(name, mnemonic, accountIndex)
	if err != nil {
		return fmt.Errorf("failed to derive keys: %w", err)
	}

	// Save key set
	if err := key.SaveKeySet(keySet); err != nil {
		return fmt.Errorf("failed to save keys: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Key set '%s' created successfully!", name)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Key types generated:")
	ux.Logger.PrintToUser("  - EC (secp256k1): Transaction signing")
	ux.Logger.PrintToUser("  - BLS: Consensus signatures")
	ux.Logger.PrintToUser("  - Ringtail: Ring signatures")
	ux.Logger.PrintToUser("  - ML-DSA: Post-quantum signatures")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Use 'lux key show %s' to view public keys and addresses.", name)

	return nil
}
