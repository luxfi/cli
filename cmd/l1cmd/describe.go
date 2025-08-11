// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newDescribeCmd() *cobra.Command {
	var showGenesis bool

	cmd := &cobra.Command{
		Use:   "describe [l1Name]",
		Short: "Show detailed information about an L1",
		Long:  `Show detailed configuration and status information for a sovereign L1 blockchain.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]

			sc, err := app.LoadSidecar(l1Name)
			if err != nil {
				return fmt.Errorf("failed to load L1 %s: %w", l1Name, err)
			}

			ux.Logger.PrintToUser("L1 Blockchain: %s", l1Name)
			ux.Logger.PrintToUser("%s", "="+strings.Repeat("=", len(l1Name)+14))
			ux.Logger.PrintToUser("")

			// Basic info
			ux.Logger.PrintToUser("📋 Configuration:")
			ux.Logger.PrintToUser("   Type: Sovereign L1")
			ux.Logger.PrintToUser("   Chain ID: %s", sc.ChainID)
			ux.Logger.PrintToUser("   VM: %s (version %s)", sc.VM, sc.VMVersion)
			ux.Logger.PrintToUser("")

			// Token info
			ux.Logger.PrintToUser("💰 Native Token:")
			ux.Logger.PrintToUser("   Name: %s", sc.TokenInfo.Name)
			ux.Logger.PrintToUser("   Symbol: %s", sc.TokenInfo.Symbol)
			ux.Logger.PrintToUser("")

			// Validator info
			ux.Logger.PrintToUser("🔐 Validator Management:")
			ux.Logger.PrintToUser("   Type: %s", sc.ValidatorManagement)
			if sc.ValidatorManagement == "proof-of-authority" {
				ux.Logger.PrintToUser("   - Owner controlled validator set")
				ux.Logger.PrintToUser("   - Instant finality")
				ux.Logger.PrintToUser("   - No token staking required")
			} else {
				ux.Logger.PrintToUser("   - Permissionless staking")
				ux.Logger.PrintToUser("   - Token-based security")
				ux.Logger.PrintToUser("   - Delegator support")
			}
			ux.Logger.PrintToUser("")

			// Deployment info
			if sc.BlockchainID.String() != "" {
				ux.Logger.PrintToUser("🚀 Deployment Status:")
				ux.Logger.PrintToUser("   Blockchain ID: %s", sc.BlockchainID)
				ux.Logger.PrintToUser("   Subnet ID: %s", sc.SubnetID)
				vmid, _ := sc.GetVMID()
				ux.Logger.PrintToUser("   VM ID: %s", vmid)
				ux.Logger.PrintToUser("")
			}

			// Protocol support
			ux.Logger.PrintToUser("🔗 Protocol Support:")
			ux.Logger.PrintToUser("   - Lux Native ✓")
			// Additional protocol support (OP Stack, Arbitrum, etc.) will be displayed when available
			ux.Logger.PrintToUser("")

			// Layer support
			ux.Logger.PrintToUser("📊 Layer Support:")
			ux.Logger.PrintToUser("   - Can host L2 chains")
			ux.Logger.PrintToUser("   - Can host L3 chains")
			ux.Logger.PrintToUser("   - Cross-chain messaging ready")
			ux.Logger.PrintToUser("")

			if showGenesis {
				genesis, err := app.LoadRawGenesis(l1Name)
				if err != nil {
					return err
				}

				ux.Logger.PrintToUser("📄 Genesis Configuration:")
				ux.Logger.PrintToUser("```json")
				formatted, _ := json.MarshalIndent(genesis, "", "  ")
				ux.Logger.PrintToUser("%s", string(formatted))
				ux.Logger.PrintToUser("```")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showGenesis, "genesis", "g", false, "Show genesis configuration")

	return cmd
}
