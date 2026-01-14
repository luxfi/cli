// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	deriveCount  int
	derivePrefix string
	deriveStart  int
	deriveShow   bool
	deriveExport bool
)

func newDeriveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "derive",
		Short: "Derive keys from LUX_MNEMONIC environment variable",
		Long: `Derive multiple key sets from a single mnemonic phrase.

Uses the LUX_MNEMONIC environment variable to derive keys deterministically.
Each key uses a different BIP-44 account index: m/44'/9000'/0'/0/{index}

This ensures the same mnemonic always produces the same keys, which is
essential for validator setup where keys must match P-Chain genesis allocations.

Examples:
  # Derive 5 validator keys from mnemonic
  export LUX_MNEMONIC="your 24 words here"
  lux key derive -n 5 --prefix validator

  # Derive keys starting at index 5
  lux key derive -n 3 --start 5 --prefix backup

  # Show addresses without saving (for verification)
  lux key derive -n 5 --show

  # Show addresses with private keys (DANGER)
  lux key derive -n 1 --show --export`,
		RunE: runDerive,
	}

	cmd.Flags().IntVarP(&deriveCount, "count", "n", 5, "Number of keys to derive")
	cmd.Flags().StringVarP(&derivePrefix, "prefix", "p", "mainnet-key", "Prefix for key names")
	cmd.Flags().IntVarP(&deriveStart, "start", "s", 0, "Starting account index")
	cmd.Flags().BoolVar(&deriveShow, "show", false, "Only show addresses, don't save keys")
	cmd.Flags().BoolVar(&deriveExport, "export", false, "Show private keys in output (DANGER - keep secret!)")

	return cmd
}

// ValidatorKeyInfo represents exported validator key information
type ValidatorKeyInfo struct {
	Index      uint32 `json:"index"`
	PrivateKey string `json:"private_key"`
	EthAddress string `json:"eth_address"`
	PChain     string `json:"p_chain"`
	XChain     string `json:"x_chain"`
	ShortID    string `json:"short_id"`
}

func runDerive(_ *cobra.Command, _ []string) error {
	// Get mnemonic from environment
	mnemonic := key.GetMnemonicFromEnv()
	if mnemonic == "" {
		return fmt.Errorf("LUX_MNEMONIC environment variable not set or invalid")
	}

	// Mask the mnemonic in output (show first and last word)
	words := strings.Fields(mnemonic)
	maskedMnemonic := fmt.Sprintf("%s ... %s (%d words)", words[0], words[len(words)-1], len(words))

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Deriving %d keys from mnemonic: %s", deriveCount, maskedMnemonic)
	ux.Logger.PrintToUser("BIP-44 path: m/44'/9000'/0'/0/{index}")
	ux.Logger.PrintToUser("")

	// Use mainnet network ID for address formatting
	networkID := uint32(96369) // Lux mainnet

	var results []ValidatorKeyInfo

	for i := 0; i < deriveCount; i++ {
		accountIndex := uint32(deriveStart + i) //nolint:gosec // G115: Index values are bounded by BIP-44 limits
		name := fmt.Sprintf("%s-%02d", derivePrefix, accountIndex+1)

		// Derive key using BIP-44 path with account index
		sf, err := key.NewSoftFromMnemonicWithAccount(networkID, mnemonic, accountIndex)
		if err != nil {
			return fmt.Errorf("failed to derive key for index %d: %w", accountIndex, err)
		}

		// Get addresses
		pAddrs := sf.P()
		xAddrs := sf.X()
		cAddr := sf.C()
		shortAddrs := sf.Addresses()

		pAddr := ""
		if len(pAddrs) > 0 {
			pAddr = pAddrs[0]
		}
		xAddr := ""
		if len(xAddrs) > 0 {
			xAddr = xAddrs[0]
		}
		shortID := ""
		if len(shortAddrs) > 0 {
			shortID = fmt.Sprintf("%x", shortAddrs[0][:])
		}

		info := ValidatorKeyInfo{
			Index:      accountIndex,
			EthAddress: cAddr,
			PChain:     pAddr,
			XChain:     xAddr,
			ShortID:    shortID,
		}
		if deriveExport {
			info.PrivateKey = sf.PrivKeyHex()
		}
		results = append(results, info)

		if !deriveShow {
			// Create HDKeySet for saving
			keySet := &key.HDKeySet{
				Name:         name,
				Mnemonic:     mnemonic,
				ECPrivateKey: sf.Raw(),
				ECPublicKey:  sf.Key().PublicKey().Bytes(),
				ECAddress:    cAddr,
			}

			// Save through encrypted backend
			if err := key.SaveKeySet(keySet); err != nil {
				ux.Logger.PrintToUser("Warning: failed to save key %s: %v", name, err)
			}
		}

		ux.Logger.PrintToUser("Account %d:", accountIndex)
		ux.Logger.PrintToUser("  Name:      %s", name)
		ux.Logger.PrintToUser("  C-Chain:   %s", cAddr)
		ux.Logger.PrintToUser("  P-Chain:   %s", pAddr)
		if deriveExport {
			ux.Logger.PrintToUser("  Private:   0x%s", sf.PrivKeyHex())
		}
		if !deriveShow {
			ux.Logger.PrintToUser("  Saved:     ✓")
		}
		ux.Logger.PrintToUser("")
	}

	// Export validator info for genesis use
	if !deriveShow {
		validatorsPath := os.ExpandEnv("$HOME/.lux/keys/mainnet_validators.json")
		data, err := json.MarshalIndent(results, "", "  ")
		if err == nil {
			if err := os.WriteFile(validatorsPath, data, 0o600); err != nil {
				ux.Logger.PrintToUser("Warning: failed to write validators file: %v", err)
			} else {
				ux.Logger.PrintToUser("Validator info exported to: %s", validatorsPath)
			}
		}
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("First 5 accounts can be used as genesis validators:")
	ux.Logger.PrintToUser("  Account 0: Primary deployer key (pays for transactions)")
	ux.Logger.PrintToUser("  Accounts 0-4: Bootstrap validators (need P-Chain allocations)")
	ux.Logger.PrintToUser("")

	if deriveShow {
		ux.Logger.PrintToUser("Run without --show to save keys to encrypted storage")
	} else {
		ux.Logger.PrintToUser("Keys saved. Use 'lux key list' to see all keys")
	}

	return nil
}
