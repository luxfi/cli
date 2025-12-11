// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"encoding/hex"
	"fmt"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show key set details",
		Long: `Show public keys and addresses for a key set.

Displays:
- EC (secp256k1) address (Ethereum format)
- BLS public key (consensus)
- Ringtail public key (ring signatures)
- ML-DSA public key (post-quantum)

Example:
  lux key show validator1`,
		Args: cobra.ExactArgs(1),
		RunE: runShow,
	}

	return cmd
}

func runShow(_ *cobra.Command, args []string) error {
	name := args[0]

	keySet, err := key.LoadKeySet(name)
	if err != nil {
		return fmt.Errorf("failed to load key set '%s': %w", name, err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Key Set: %s", name)
	ux.Logger.PrintToUser("")

	// EC key info
	ux.Logger.PrintToUser("EC (secp256k1) - Transaction Signing:")
	ux.Logger.PrintToUser("  Address:    %s", keySet.ECAddress)
	ux.Logger.PrintToUser("  Public Key: %s", hex.EncodeToString(keySet.ECPublicKey))
	ux.Logger.PrintToUser("")

	// BLS key info
	ux.Logger.PrintToUser("BLS - Consensus Signatures:")
	ux.Logger.PrintToUser("  Public Key: %s", hex.EncodeToString(keySet.BLSPublicKey))
	ux.Logger.PrintToUser("  PoP:        %s", hex.EncodeToString(keySet.BLSPoP))
	ux.Logger.PrintToUser("")

	// Ringtail key info
	ux.Logger.PrintToUser("Ringtail - Ring Signatures:")
	ux.Logger.PrintToUser("  Public Key: %s", hex.EncodeToString(keySet.RingtailPublicKey))
	ux.Logger.PrintToUser("")

	// ML-DSA key info
	ux.Logger.PrintToUser("ML-DSA - Post-Quantum Signatures:")
	ux.Logger.PrintToUser("  Public Key: %s...", hex.EncodeToString(keySet.MLDSAPublicKey[:64]))
	ux.Logger.PrintToUser("  (truncated, full key is %d bytes)", len(keySet.MLDSAPublicKey))
	ux.Logger.PrintToUser("")

	return nil
}
