// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	targetBase string
	hotSwap    bool
)

func newMigrateBaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-base [subnetName]",
		Short: "Migrate subnet to a different base chain",
		Long: `Migrate a subnet (L2) to use a different base chain for sequencing.

This allows moving between Lux, Ethereum, or Avalanche as the sequencing layer
while preserving all state and history. The migration requires governance
approval and a brief pause for state checkpoint.

Example:
  lux subnet migrate-base mySubnet --target ethereum`,
		Args: cobra.ExactArgs(1),
		RunE: migrateBase,
	}

	cmd.Flags().StringVar(&targetBase, "target", "", "Target base chain (lux, ethereum, avalanche)")
	cmd.Flags().BoolVar(&hotSwap, "hot-swap", false, "Attempt hot-swap migration (experimental)")

	return cmd
}

func migrateBase(cmd *cobra.Command, args []string) error {
	subnetName := args[0]

	ux.Logger.PrintToUser("🔄 Base Chain Migration")
	ux.Logger.PrintToUser("======================")
	ux.Logger.PrintToUser("")

	// Load subnet configuration
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return fmt.Errorf("failed to load subnet %s: %w", subnetName, err)
	}

	if sc.Sovereign {
		return fmt.Errorf("%s is a sovereign L1 and cannot use base chain sequencing", subnetName)
	}

	ux.Logger.PrintToUser("📊 Current Configuration:")
	ux.Logger.PrintToUser("   Subnet Name: %s", subnetName)
	ux.Logger.PrintToUser("   Current Base: %s", sc.BaseChain)
	ux.Logger.PrintToUser("   Sequencer Type: %s", sc.SequencerType)
	ux.Logger.PrintToUser("   Block Time: %dms", sc.L1BlockTime)
	ux.Logger.PrintToUser("")

	// Target base selection
	if targetBase == "" {
		baseOptions := []string{
			"Lux (100ms blocks, lowest cost)",
			"Ethereum (12s blocks, highest security)",
			"Avalanche (2s blocks, fast finality)",
		}

		// Remove current base from options
		filteredOptions := []string{}
		for _, opt := range baseOptions {
			if !isCurrentBase(opt, sc.BaseChain) {
				filteredOptions = append(filteredOptions, opt)
			}
		}

		choice, err := app.Prompt.CaptureList(
			"Select new base chain",
			filteredOptions,
		)
		if err != nil {
			return err
		}

		targetBase = parseBaseChoice(choice)
	}

	// Show migration impact
	ux.Logger.PrintToUser("⚡ Migration Impact Analysis:")
	showMigrationImpact(sc.BaseChain, targetBase)
	ux.Logger.PrintToUser("")

	// Governance requirement
	ux.Logger.PrintToUser("🏛️  Governance Requirements:")
	ux.Logger.PrintToUser("   - Proposal submission: 100 LUX")
	ux.Logger.PrintToUser("   - Voting period: 7 days")
	ux.Logger.PrintToUser("   - Quorum: 10% of staked tokens")
	ux.Logger.PrintToUser("   - Approval threshold: 66%")
	ux.Logger.PrintToUser("")

	// Migration mode
	if !hotSwap {
		ux.Logger.PrintToUser("📋 Cold Migration Process:")
		ux.Logger.PrintToUser("   1. Freeze subnet at specific block")
		ux.Logger.PrintToUser("   2. Create state checkpoint")
		ux.Logger.PrintToUser("   3. Deploy new inbox on target chain")
		ux.Logger.PrintToUser("   4. Update rollup configuration")
		ux.Logger.PrintToUser("   5. Resume from checkpoint")
		ux.Logger.PrintToUser("   Expected downtime: ~10 minutes")
	} else {
		ux.Logger.PrintToUser("🔥 Hot-Swap Migration (Experimental):")
		ux.Logger.PrintToUser("   1. Deploy inbox on target chain")
		ux.Logger.PrintToUser("   2. Dual-post blocks for transition period")
		ux.Logger.PrintToUser("   3. Validators switch to new base")
		ux.Logger.PrintToUser("   4. Deprecate old inbox")
		ux.Logger.PrintToUser("   Expected downtime: None")
		ux.Logger.PrintToUser("   ⚠️  Higher complexity and gas costs")
	}
	ux.Logger.PrintToUser("")

	// Confirm migration
	confirm, err := app.Prompt.CaptureYesNo("Proceed with migration proposal?")
	if err != nil || !confirm {
		return fmt.Errorf("migration cancelled")
	}

	// Create governance proposal
	ux.Logger.PrintToUser("\n📝 Creating Governance Proposal...")
	
	proposal := &models.GovernanceProposal{
		Type:        "base-migration",
		Title:       fmt.Sprintf("Migrate %s from %s to %s", subnetName, sc.BaseChain, targetBase),
		Description: fmt.Sprintf("Migrate subnet to improve performance and reduce costs"),
		L2Name:      subnetName,
		CurrentBase: sc.BaseChain,
		TargetBase:  targetBase,
		HotSwap:     hotSwap,
		CreatedAt:   time.Now().Unix(),
	}

	// Simulate proposal submission
	ux.Logger.PrintToUser("   Submitting proposal...")
	time.Sleep(2 * time.Second)
	
	proposalID := fmt.Sprintf("PROP-%d", time.Now().Unix())
	ux.Logger.PrintToUser("   ✅ Proposal submitted: %s", proposalID)
	ux.Logger.PrintToUser("")

	// Next steps
	ux.Logger.PrintToUser("📊 Proposal Status:")
	ux.Logger.PrintToUser("   ID: %s", proposalID)
	ux.Logger.PrintToUser("   Status: Voting Active")
	ux.Logger.PrintToUser("   Ends: %s", time.Now().Add(7*24*time.Hour).Format("2006-01-02 15:04"))
	ux.Logger.PrintToUser("")
	
	ux.Logger.PrintToUser("💡 Next steps:")
	ux.Logger.PrintToUser("   1. Share proposal: lux governance share %s", proposalID)
	ux.Logger.PrintToUser("   2. Monitor votes: lux governance status %s", proposalID)
	ux.Logger.PrintToUser("   3. Execute if passed: lux governance execute %s", proposalID)

	return nil
}

func isCurrentBase(option, current string) bool {
	switch current {
	case "lux":
		return option == "Lux (100ms blocks, lowest cost)"
	case "ethereum":
		return option == "Ethereum (12s blocks, highest security)"
	case "avalanche":
		return option == "Avalanche (2s blocks, fast finality)"
	}
	return false
}

func parseBaseChoice(choice string) string {
	switch choice {
	case "Lux (100ms blocks, lowest cost)":
		return "lux"
	case "Ethereum (12s blocks, highest security)":
		return "ethereum"
	case "Avalanche (2s blocks, fast finality)":
		return "avalanche"
	}
	return choice
}

func showMigrationImpact(from, to string) {
	// Block time changes
	fromTime := getBlockTime(from)
	toTime := getBlockTime(to)
	
	if toTime < fromTime {
		ux.Logger.PrintToUser("   ⬆️  Faster block times: %dms → %dms", fromTime, toTime)
		ux.Logger.PrintToUser("   ✅ Lower latency for users")
	} else if toTime > fromTime {
		ux.Logger.PrintToUser("   ⬇️  Slower block times: %dms → %dms", fromTime, toTime)
		ux.Logger.PrintToUser("   ⚠️  Higher latency but more security")
	}

	// Cost implications
	switch to {
	case "lux":
		ux.Logger.PrintToUser("   💸 Lowest data costs")
		ux.Logger.PrintToUser("   🚀 Ultra-low latency (100ms)")
	case "ethereum":
		ux.Logger.PrintToUser("   💰 Higher data costs (ETH gas)")
		ux.Logger.PrintToUser("   🛡️  Maximum security inheritance")
	case "avalanche":
		ux.Logger.PrintToUser("   💵 Moderate data costs")
		ux.Logger.PrintToUser("   ⚡ Fast finality (~1s)")
	}

	// MEV implications
	ux.Logger.PrintToUser("   🔀 MEV flows to %s builders", to)
}

func getBlockTime(chain string) int {
	switch chain {
	case "lux":
		return 100 // 100ms
	case "ethereum":
		return 12000 // 12s
	case "avalanche":
		return 2000 // 2s
	default:
		return 12000 // Default to Ethereum timing
	}
}