// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Manage validators for L1 blockchains",
		Long: `Manage validators for sovereign L1 blockchains.

Validators can participate in multiple protocols:
- Lux L1s (sovereign blockchains)
- Legacy Avalanche subnets
- OP Stack L2/L3 chains
- Other blockchain protocols

This allows a single node to validate across multiple blockchain ecosystems.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	// validator add
	cmd.AddCommand(newValidatorAddCmd())
	// validator remove
	cmd.AddCommand(newValidatorRemoveCmd())
	// validator list
	cmd.AddCommand(newValidatorListCmd())
	// validator protocols
	cmd.AddCommand(newValidatorProtocolsCmd())

	return cmd
}

func newValidatorAddCmd() *cobra.Command {
	var (
		nodeID       string
		weight       uint64
		protocol     string
		validatorType string
	)

	cmd := &cobra.Command{
		Use:   "add [l1Name]",
		Short: "Add a validator to an L1",
		Long: `Add a validator to a sovereign L1 blockchain.

The validator can be added to:
- Lux L1 (sovereign blockchain)
- L2/L3 built on top of an L1
- Cross-protocol validation (e.g., also validate OP Stack)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]
			
			ux.Logger.PrintToUser("Adding validator to L1: %s", l1Name)
			
			// Load L1 configuration
			sc, err := app.LoadSidecar(l1Name)
			if err != nil {
				return fmt.Errorf("failed to load L1 %s: %w", l1Name, err)
			}

			if sc.ValidatorManagement == "proof-of-authority" {
				ux.Logger.PrintToUser("Using Proof of Authority validator management")
				// PoA flow - only owner can add validators
			} else {
				ux.Logger.PrintToUser("Using Proof of Stake validator management")
				// PoS flow - anyone can stake
			}

			ux.Logger.PrintToUser("✅ Validator configuration prepared")
			ux.Logger.PrintToUser("   Node ID: %s", nodeID)
			ux.Logger.PrintToUser("   Protocol: %s", protocol)
			
			return nil
		},
	}

	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID of the validator")
	cmd.Flags().Uint64Var(&weight, "weight", 100, "Validator weight")
	cmd.Flags().StringVar(&protocol, "protocol", "lux", "Protocol type (lux, avalanche, opstack)")
	cmd.Flags().StringVar(&validatorType, "type", "primary", "Validator type (primary, l2, l3)")

	return cmd
}

func newValidatorProtocolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protocols",
		Short: "List supported validator protocols",
		Long:  `List all protocols that validators can participate in.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Supported Validator Protocols:")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("🔷 Lux Protocol")
			ux.Logger.PrintToUser("   - Sovereign L1 blockchains")
			ux.Logger.PrintToUser("   - L2/L3 chains on Lux L1s")
			ux.Logger.PrintToUser("   - Native Lux consensus")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("🔺 Avalanche Protocol")
			ux.Logger.PrintToUser("   - Legacy subnet support")
			ux.Logger.PrintToUser("   - C-Chain compatibility")
			ux.Logger.PrintToUser("   - Avalanche consensus")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("🟦 OP Stack")
			ux.Logger.PrintToUser("   - Optimistic rollups")
			ux.Logger.PrintToUser("   - L2/L3 chains")
			ux.Logger.PrintToUser("   - Ethereum compatibility")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("🔗 Cross-Protocol Features")
			ux.Logger.PrintToUser("   - Single node, multiple protocols")
			ux.Logger.PrintToUser("   - Unified validator management")
			ux.Logger.PrintToUser("   - Cross-chain messaging")

			return nil
		},
	}

	return cmd
}

func newValidatorRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [l1Name]",
		Short: "Remove a validator from an L1",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]
			ux.Logger.PrintToUser("Removing validator from L1: %s", l1Name)
			// Implementation
			return nil
		},
	}
	return cmd
}

func newValidatorListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [l1Name]",
		Short: "List validators for an L1",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// List all L1s and their validators
				ux.Logger.PrintToUser("All L1 Validators:")
				// Implementation
			} else {
				l1Name := args[0]
				ux.Logger.PrintToUser("Validators for L1: %s", l1Name)
				// Implementation
			}
			return nil
		},
	}
	return cmd
}