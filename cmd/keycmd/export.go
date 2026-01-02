// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	exportMnemonic bool
	exportOutput   string
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export key set",
		Long: `Export key set data.

By default, exports public keys. Use --mnemonic to export the seed phrase.

WARNING: Exporting the mnemonic exposes your private keys!

Examples:
  lux key export validator1                    # Export public keys
  lux key export validator1 --mnemonic         # Export mnemonic (DANGER!)
  lux key export validator1 -o keys.json       # Export to file`,
		Args: cobra.ExactArgs(1),
		RunE: runExport,
	}

	cmd.Flags().BoolVar(&exportMnemonic, "mnemonic", false, "Export mnemonic phrase (DANGEROUS!)")
	cmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")

	return cmd
}

func runExport(_ *cobra.Command, args []string) error {
	name := args[0]

	keySet, err := key.LoadKeySet(name)
	if err != nil {
		return fmt.Errorf("failed to load key set '%s': %w", name, err)
	}

	var output string

	if exportMnemonic {
		// Read mnemonic from file
		keysDir, err := key.GetKeysDir()
		if err != nil {
			return fmt.Errorf("failed to get keys directory: %w", err)
		}
		mnemonicPath := filepath.Join(keysDir, name, key.MnemonicFile)
		data, err := os.ReadFile(mnemonicPath) //nolint:gosec // G304: Reading from user's key directory
		if err != nil {
			return fmt.Errorf("failed to read mnemonic: %w", err)
		}

		output = fmt.Sprintf(`{
  "name": "%s",
  "mnemonic": "%s",
  "ec_address": "%s"
}`, name, string(data), keySet.ECAddress)

		if exportOutput == "" {
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("WARNING: This exposes your private keys!")
			ux.Logger.PrintToUser("")
		}
	} else {
		output = fmt.Sprintf(`{
  "name": "%s",
  "ec": {
    "address": "%s",
    "public_key": "%s"
  },
  "bls": {
    "public_key": "%s",
    "proof_of_possession": "%s"
  },
  "ringtail": {
    "public_key": "%s"
  },
  "mldsa": {
    "public_key": "%s"
  }
}`,
			name,
			keySet.ECAddress,
			hex.EncodeToString(keySet.ECPublicKey),
			hex.EncodeToString(keySet.BLSPublicKey),
			hex.EncodeToString(keySet.BLSPoP),
			hex.EncodeToString(keySet.RingtailPublicKey),
			hex.EncodeToString(keySet.MLDSAPublicKey),
		)
	}

	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, []byte(output), 0o600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		ux.Logger.PrintToUser("Exported to %s", exportOutput)
	} else {
		fmt.Println(output)
	}

	return nil
}
