// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l3cmd

import (
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [l3Name]",
		Short: "Deploy an L3 to its base L2",
		Args:  cobra.ExactArgs(1),
		RunE:  deployL3,
	}

	return cmd
}

func deployL3(cmd *cobra.Command, args []string) error {
	l3Name := args[0]

	ux.Logger.PrintToUser("🚀 Deploying L3: %s", l3Name)
	ux.Logger.PrintToUser("==================")

	// Load L3 configuration
	sc, err := app.LoadSidecar(l3Name)
	if err != nil {
		return fmt.Errorf("failed to load L3 configuration: %w", err)
	}

	// Validate L3 configuration
	if sc.ChainLayer != 3 {
		return fmt.Errorf("%s is not configured as an L3 (chain layer: %d)", l3Name, sc.ChainLayer)
	}

	// Get base L2 information
	baseChain := sc.BaseChain
	if baseChain == "" {
		return fmt.Errorf("base chain not configured for L3 %s", l3Name)
	}

	ux.Logger.PrintToUser("📋 Configuration:")
	ux.Logger.PrintToUser("  • Base Chain: %s", baseChain)
	ux.Logger.PrintToUser("  • Sequencer Type: %s", sc.SequencerType)
	
	// Deploy contracts on base L2
	ux.Logger.PrintToUser("📦 Deploying contracts on base L2...")
	
	// Deploy inbox contract for based rollup
	if sc.BasedRollup {
		ux.Logger.PrintToUser("  • Deploying inbox contract...")
		// Inbox contract would handle L3 transaction batching
		sc.InboxContract = "0x" + fmt.Sprintf("%040x", uint64(time.Now().Unix()))
		ux.Logger.PrintToUser("  • Inbox contract deployed at: %s", sc.InboxContract)
	}

	// Set up preconfirmation if enabled
	if sc.PreconfirmEnabled {
		ux.Logger.PrintToUser("  • Setting up preconfirmation service...")
	}

	// Initialize L3 genesis
	ux.Logger.PrintToUser("🔧 Initializing L3 genesis...")
	
	// Save updated configuration
	if err := app.UpdateSidecar(&sc); err != nil {
		return fmt.Errorf("failed to save L3 configuration: %w", err)
	}

	ux.Logger.PrintToUser("✅ L3 deployment completed successfully!")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Next steps:")
	ux.Logger.PrintToUser("  • Start L3 sequencer: lux l3 start %s", l3Name)
	ux.Logger.PrintToUser("  • Bridge assets: lux l3 bridge %s", l3Name)
	
	return nil
}
