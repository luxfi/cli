// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package keycmd provides commands for managing cryptographic keys.
// Keys are stored in ~/.lux/keys/<name>/{ec,bls,rt,mldsa}/ directories.
package keycmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the key command suite.
// Commands:
//   - lux key create <name>     - Generate new key set from mnemonic
//   - lux key list              - List all key sets
//   - lux key show <name>       - Show key set details and addresses
//   - lux key delete <name>     - Delete a key set
//   - lux key export <name>     - Export key set (mnemonic or individual keys)
//   - lux key import <name>     - Import key set from mnemonic
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:     "key",
		Aliases: []string{"keys"},
		Short:   "Manage cryptographic keys for validators and accounts",
		Long: `The key command suite provides tools for managing all cryptographic keys
used in the Lux network.

Key types managed:
- EC (secp256k1): Transaction signing, Ethereum compatibility
- BLS: Consensus participation, aggregated signatures
- Ringtail: Ring signatures for privacy
- ML-DSA: Post-quantum digital signatures (NIST Level 3)

All keys are derived from a single BIP39 mnemonic phrase using HKDF,
stored in ~/.lux/keys/<name>/ with separate subdirectories for each type.

Examples:
  lux key create validator1              # Create new key set
  lux key create validator1 --mnemonic   # Create from existing mnemonic
  lux key generate -n 5                  # Batch generate 5 key sets (key-0 to key-4)
  lux key generate -n 10 -p validator    # Generate validator-0 to validator-9
  lux key list                           # List all key sets
  lux key show validator1                # Show public keys and addresses
  lux key delete validator1              # Delete key set
  lux key export validator1              # Export mnemonic (DANGER!)`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	// Key management commands
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newGenerateCmd())

	return cmd
}
