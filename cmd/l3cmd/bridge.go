// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l3cmd

import (
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newBridgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "Manage cross-layer bridges for L3s",
		Long: `Configure bridges between L3s and other layers.

Supports:
- L3 ↔ L2 native bridging
- L3 ↔ L1 via L2 bridge
- L3 ↔ L3 cross-app communication
- IBC for Cosmos interoperability`,
	}

	// Subcommands
	cmd.AddCommand(newBridgeEnableCmd())
	cmd.AddCommand(newBridgeTransferCmd())

	return cmd
}

func newBridgeEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable [l3Name]",
		Short: "Enable bridge connections for an L3",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l3Name := args[0]
			ux.Logger.PrintToUser("🌉 Enabling bridges for L3: %s", l3Name)
			// TODO: Implement bridge enabling logic
			return nil
		},
	}
}

func newBridgeTransferCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transfer",
		Short: "Transfer assets between layers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("💸 Cross-Layer Transfer")
			// TODO: Implement transfer logic
			return nil
		},
	}
}