// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l3cmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	l2Base     string
	vmType     string
	preconfirm bool
	daLayer    string
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
- High-frequency trading environments`,
		Args: cobra.ExactArgs(1),
		RunE: createL3,
	}

	cmd.Flags().StringVar(&l2Base, "l2", "", "Base L2 to deploy on")
	cmd.Flags().StringVar(&vmType, "vm", "evm", "VM type (evm, custom, wasm)")
	cmd.Flags().BoolVar(&preconfirm, "preconfirm", true, "Enable pre-confirmations")
	cmd.Flags().StringVar(&daLayer, "da", "inherit", "Data availability (inherit, blob, custom)")

	return cmd
}

func createL3(cmd *cobra.Command, args []string) error {
	l3Name := args[0]

	ux.Logger.PrintToUser("🎮 Creating L3 Configuration")
	ux.Logger.PrintToUser("===========================")
	ux.Logger.PrintToUser("")

	// Select base L2
	if l2Base == "" {
		// TODO: List available L2s
		l2Base, _ = app.Prompt.CaptureString("Enter base L2 name")
	}

	// VM type selection
	if vmType == "" {
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
			vmType = "evm"
		case "Custom VM (Maximum flexibility)":
			vmType = "custom"
		case "WASM (WebAssembly runtime)":
			vmType = "wasm"
		case "Move VM (Move language)":
			vmType = "move"
		}
	}

	// Create L3 configuration
	sc := &models.Sidecar{
		Name:    l3Name,
		Subnet:  l3Name,
		Version: "2.0.0",

		// L3 specific
		Sovereign:         false, // L3s are never sovereign
		BaseChain:         l2Base,
		SequencerType:     "inherit", // Inherits from L2
		PreconfirmEnabled: preconfirm,

		// Layer identifier
		ChainLayer: 3,
	}

	// Token configuration
	ux.Logger.PrintToUser("\n💰 Token Configuration")
	tokenName, _ := app.Prompt.CaptureString("Token name")
	tokenSymbol, _ := app.Prompt.CaptureString("Token symbol")

	sc.TokenInfo = models.TokenInfo{
		Name:     tokenName,
		Symbol:   tokenSymbol,
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
	ux.Logger.PrintToUser("   Token: %s (%s)", tokenName, tokenSymbol)
	ux.Logger.PrintToUser("   Pre-confirmations: %v", preconfirm)

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("💡 Next steps:")
	ux.Logger.PrintToUser("   1. Deploy: lux l3 deploy %s", l3Name)
	ux.Logger.PrintToUser("   2. Configure bridges: lux l3 bridge enable %s", l3Name)
	ux.Logger.PrintToUser("   3. Test locally: lux network quickstart --l3 %s", l3Name)

	return nil
}
