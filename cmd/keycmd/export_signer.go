// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	exportSignerCount  int
	exportSignerStart  int
	exportSignerOutput string
)

func newExportSignerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-signer",
		Short: "Export BLS signer keys for luxd nodes",
		Long: `Export BLS signer keys derived from LUX_MNEMONIC for use as luxd
staking signer keys. Each key is written as a raw 32-byte file.

This is needed when starting luxd nodes manually (outside of netrunner)
that need to use mnemonic-derived BLS keys for consensus.

Examples:
  # Export signer keys for accounts 5-9
  export LUX_MNEMONIC="your mnemonic here"
  lux key export-signer -n 5 --start 5 --output ~/.lux/local-validators

  # This creates:
  #   ~/.lux/local-validators/node5/signer.key
  #   ~/.lux/local-validators/node6/signer.key
  #   ...`,
		RunE:         exportSignerKeys,
		SilenceUsage: true,
	}

	cmd.Flags().IntVarP(&exportSignerCount, "count", "n", 5, "Number of signer keys to export")
	cmd.Flags().IntVarP(&exportSignerStart, "start", "s", 0, "Starting account index")
	cmd.Flags().StringVarP(&exportSignerOutput, "output", "o", "", "Output directory (required)")
	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func exportSignerKeys(_ *cobra.Command, _ []string) error {
	mnemonic := key.GetMnemonicFromEnv()
	if mnemonic == "" {
		return fmt.Errorf("LUX_MNEMONIC environment variable not set")
	}

	ux.Logger.PrintToUser("Exporting %d BLS signer keys (indices %d-%d)...",
		exportSignerCount, exportSignerStart, exportSignerStart+exportSignerCount-1)

	for i := 0; i < exportSignerCount; i++ {
		idx := exportSignerStart + i
		name := fmt.Sprintf("node%d", idx)

		keySet, err := key.DeriveAllKeysWithAccount(name, mnemonic, uint32(idx))
		if err != nil {
			return fmt.Errorf("failed to derive keys for account %d: %w", idx, err)
		}

		// Create output directory
		nodeDir := filepath.Join(exportSignerOutput, name)
		if err := os.MkdirAll(nodeDir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", nodeDir, err)
		}

		// Derive the actual BLS signer key from the HKDF seed.
		// BLSPrivateKey is the HKDF seed; we need to run it through BLS KeyGen
		// to get a valid BLS secret key that luxd can deserialize.
		signerBytes, err := key.DeriveBLSSignerBytes(keySet.BLSPrivateKey)
		if err != nil {
			return fmt.Errorf("failed to derive BLS signer for account %d: %w", idx, err)
		}

		signerPath := filepath.Join(nodeDir, "signer.key")
		if err := os.WriteFile(signerPath, signerBytes, 0o600); err != nil {
			return fmt.Errorf("failed to write signer key: %w", err)
		}

		ux.Logger.PrintToUser("  [%d] %s → %s", idx, keySet.NodeID, signerPath)
	}

	ux.Logger.PrintToUser("\nSigner keys exported. Use --staking-signer-key-file to point luxd at these files.")
	return nil
}
