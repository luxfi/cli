// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l3cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

// VM type constants for L3 chains
const (
	vmTypeEVM    = "evm"
	vmTypeCustom = "custom"
	vmTypeWASM   = "wasm"
	vmTypeMove   = "move"
)

var (
	l2Base      string
	vmType      string
	preconfirm  bool
	daLayer     string
	tokenName   string
	tokenSymbol string
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [l3Name]",
		Short: "Create a new L3 configuration",
		Long: `Create a new L3 (application-specific chain) on top of an L2.

L3s provide maximum customization for specific applications while
inheriting the security and performance characteristics of their L2 base.

Common use cases:
- Gaming chains with custom state transitions
- DeFi pools with app-specific optimizations
- Privacy-focused applications
- High-frequency trading environments

NON-INTERACTIVE MODE:
  Use flags to provide all parameters:
  --l2               Base L2 to deploy on (required)
  --vm               VM type (evm, custom, wasm, move)
  --token-name       Native token name
  --token-symbol     Native token symbol

EXAMPLES:
  lux l3 create mygame --l2 mychain --vm evm --token-name GameToken --token-symbol GAME`,
		Args: cobra.ExactArgs(1),
		RunE: createL3,
	}

	cmd.Flags().StringVar(&l2Base, "l2", "", "Base L2 to deploy on")
	cmd.Flags().StringVar(&vmType, "vm", vmTypeEVM, "VM type (evm, custom, wasm, move)")
	cmd.Flags().BoolVar(&preconfirm, "preconfirm", true, "Enable pre-confirmations")
	cmd.Flags().StringVar(&daLayer, "da", "inherit", "Data availability (inherit, blob, custom)")
	cmd.Flags().StringVar(&tokenName, "token-name", "", "Native token name")
	cmd.Flags().StringVar(&tokenSymbol, "token-symbol", "", "Native token symbol")

	return cmd
}

func createL3(cmd *cobra.Command, args []string) error {
	l3Name := args[0]

	ux.Logger.PrintToUser("🎮 Creating L3 Configuration")
	ux.Logger.PrintToUser("===========================")
	ux.Logger.PrintToUser("")

	// Select base L2
	if l2Base == "" {
		if !prompts.IsInteractive() {
			return fmt.Errorf("--l2 is required in non-interactive mode")
		}
		// List available L2s from sidecar files
		l2s, err := getAvailableL2s()
		if err != nil {
			return fmt.Errorf("failed to get available L2s: %w", err)
		}

		if len(l2s) > 0 {
			l2Base, err = app.Prompt.CaptureList("Select base L2", l2s)
			if err != nil {
				return err
			}
		} else {
			l2Base, err = app.Prompt.CaptureString("Enter base L2 name")
			if err != nil {
				return err
			}
		}
	}

	// VM type selection - already has default "evm" from flag, only prompt if explicitly empty
	// Validate the vm type
	switch vmType {
	case vmTypeEVM, vmTypeCustom, vmTypeWASM, vmTypeMove:
		// valid
	case "":
		if !prompts.IsInteractive() {
			vmType = vmTypeEVM // default
		} else {
			vmOptions := []string{
				"EVM (Ethereum compatible)",
				"Custom VM (Maximum flexibility)",
				"WASM (WebAssembly runtime)",
				"Move VM (Move language)",
			}

			choice, err := app.Prompt.CaptureList(
				"Select VM type for your L3",
				vmOptions,
			)
			if err != nil {
				return err
			}

			switch choice {
			case "EVM (Ethereum compatible)":
				vmType = vmTypeEVM
			case "Custom VM (Maximum flexibility)":
				vmType = vmTypeCustom
			case "WASM (WebAssembly runtime)":
				vmType = vmTypeWASM
			case "Move VM (Move language)":
				vmType = vmTypeMove
			}
		}
	default:
		return fmt.Errorf("invalid VM type: %s (valid: evm, custom, wasm, move)", vmType)
	}

	// Create L3 configuration
	sc := &models.Sidecar{
		Name:   l3Name,
		Subnet: l3Name,

		// L3 specific
		Sovereign:         false, // L3s are never sovereign
		BaseChain:         l2Base,
		SequencerType:     "inherit", // Inherits from L2
		PreconfirmEnabled: preconfirm,

		// Layer identifier
		ChainLayer: 3,
	}

	// Token configuration
	ux.Logger.PrintToUser("\nToken Configuration")
	tkName := tokenName
	tkSymbol := tokenSymbol
	if tkName == "" {
		if !prompts.IsInteractive() {
			tkName = "Token" // default
		} else {
			tkName, _ = app.Prompt.CaptureString("Token name")
		}
	}
	if tkSymbol == "" {
		if !prompts.IsInteractive() {
			tkSymbol = "TKN" // default
		} else {
			tkSymbol, _ = app.Prompt.CaptureString("Token symbol")
		}
	}

	sc.TokenInfo = models.TokenInfo{
		Name:     tkName,
		Symbol:   tkSymbol,
		Decimals: 18,
		Supply:   "0",
	}

	// Save configuration
	if err := app.CreateSidecar(sc); err != nil {
		return fmt.Errorf("failed to save L3 configuration: %w", err)
	}

	// Display summary
	ux.Logger.PrintToUser("\n✅ L3 Configuration Created!")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("📊 Configuration Summary:")
	ux.Logger.PrintToUser("   Name: %s", l3Name)
	ux.Logger.PrintToUser("   Base L2: %s", l2Base)
	ux.Logger.PrintToUser("   VM Type: %s", vmType)
	ux.Logger.PrintToUser("   Token: %s (%s)", tkName, tkSymbol)
	ux.Logger.PrintToUser("   Pre-confirmations: %v", preconfirm)

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("💡 Next steps:")
	ux.Logger.PrintToUser("   1. Deploy: lux l3 deploy %s", l3Name)
	ux.Logger.PrintToUser("   2. Configure bridges: lux l3 bridge enable %s", l3Name)
	ux.Logger.PrintToUser("   3. Test locally: lux network quickstart --l3 %s", l3Name)

	return nil
}

// getAvailableL2s returns a list of available L2 configurations
func getAvailableL2s() ([]string, error) {
	subnetDir := app.GetSubnetDir()
	entries, err := os.ReadDir(subnetDir)
	if err != nil {
		return nil, err
	}

	var l2s []string
	for _, entry := range entries {
		if entry.IsDir() {
			sidecarPath := filepath.Join(subnetDir, entry.Name(), "sidecar.json")
			if _, err := os.Stat(sidecarPath); err == nil {
				// Check if it's an L2 (has subnet configuration)
				data, err := os.ReadFile(sidecarPath) //nolint:gosec // G304: Reading from app's data directory
				if err == nil {
					var sc models.Sidecar
					if json.Unmarshal(data, &sc) == nil {
						// Consider it an L2 if it has subnet or blockchain configuration
						if sc.Subnet != "" || len(sc.Networks) > 0 {
							l2s = append(l2s, entry.Name())
						}
					}
				}
			}
		}
	}

	return l2s, nil
}
