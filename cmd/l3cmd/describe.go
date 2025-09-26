// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l3cmd

import (
	"fmt"
	"os"
	
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [l3Name]",
		Short: "Show detailed information about an L3",
		Args:  cobra.ExactArgs(1),
		RunE:  describeL3,
	}

	return cmd
}

func describeL3(cmd *cobra.Command, args []string) error {
	l3Name := args[0]
	ux.Logger.PrintToUser("📊 L3 Details: %s", l3Name)
	ux.Logger.PrintToUser("================================")
	ux.Logger.PrintToUser("")
	
	// Load L3 configuration
	sc, err := app.LoadSidecar(l3Name)
	if err != nil {
		return fmt.Errorf("failed to load L3 configuration: %w", err)
	}
	
	// Display basic information
	ux.Logger.PrintToUser("⚙️  Configuration:")
	ux.Logger.PrintToUser("  • Name: %s", l3Name)
	ux.Logger.PrintToUser("  • VM: %s", sc.VM)
	ux.Logger.PrintToUser("  • VM Version: %s", sc.VMVersion)
	ux.Logger.PrintToUser("  • RPC Version: %d", sc.RPCVersion)
	ux.Logger.PrintToUser("")
	
	// Display token information
	if sc.TokenInfo.Name != "" {
		ux.Logger.PrintToUser("💰 Token Information:")
		ux.Logger.PrintToUser("  • Name: %s", sc.TokenInfo.Name)
		ux.Logger.PrintToUser("  • Symbol: %s", sc.TokenInfo.Symbol)
		ux.Logger.PrintToUser("  • Decimals: %d", sc.TokenInfo.Decimals)
		ux.Logger.PrintToUser("  • Supply: %s", sc.TokenInfo.Supply)
		ux.Logger.PrintToUser("")
	}
	
	// Display network deployments
	if len(sc.Networks) > 0 {
		ux.Logger.PrintToUser("🌐 Network Deployments:")
		for network, data := range sc.Networks {
			ux.Logger.PrintToUser("  • %s:", network)
			ux.Logger.PrintToUser("    - Subnet ID: %s", data.SubnetID)
			ux.Logger.PrintToUser("    - Blockchain ID: %s", data.BlockchainID)
			if len(data.RPCEndpoints) > 0 {
				ux.Logger.PrintToUser("    - RPC: %s", data.RPCEndpoints[0])
			}
		}
		ux.Logger.PrintToUser("")
	}
	
	// Display L3-specific configuration
	if sc.ExtraNetworkData != nil {
		if bridgeEnabled, ok := sc.ExtraNetworkData["bridgeEnabled"].(bool); ok && bridgeEnabled {
			ux.Logger.PrintToUser("🌉 Bridge Configuration:")
			ux.Logger.PrintToUser("  • Status: Enabled")
			if bridgeType, ok := sc.ExtraNetworkData["bridgeType"].(string); ok {
				ux.Logger.PrintToUser("  • Type: %s", bridgeType)
			}
			ux.Logger.PrintToUser("")
		}
		
		if baseL2, ok := sc.ExtraNetworkData["baseL2"].(string); ok {
			ux.Logger.PrintToUser("🔗 Base Layer:")
			ux.Logger.PrintToUser("  • L2: %s", baseL2)
			ux.Logger.PrintToUser("")
		}
		
		if preconfirm, ok := sc.ExtraNetworkData["preconfirmation"].(bool); ok && preconfirm {
			ux.Logger.PrintToUser("⚡ Performance:")
			ux.Logger.PrintToUser("  • Pre-confirmations: Enabled")
			ux.Logger.PrintToUser("")
		}
	}
	
	// Display import/export options
	ux.Logger.PrintToUser("📦 Genesis Configuration:")
	genesisPath := app.GetGenesisPath(l3Name)
	if _, err := os.Stat(genesisPath); err == nil {
		ux.Logger.PrintToUser("  • Genesis file: %s", genesisPath)
	} else {
		ux.Logger.PrintToUser("  • Genesis file: Not generated")
	}
	ux.Logger.PrintToUser("")
	
	// Display available commands
	ux.Logger.PrintToUser("💡 Available Commands:")
	ux.Logger.PrintToUser("  • Deploy: lux l3 deploy %s", l3Name)
	ux.Logger.PrintToUser("  • Bridge: lux l3 bridge enable %s", l3Name)
	ux.Logger.PrintToUser("  • Transfer: lux l3 bridge transfer")
	ux.Logger.PrintToUser("  • Test: lux network quickstart --l3 %s", l3Name)
	
	return nil
}
