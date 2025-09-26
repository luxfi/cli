// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var showDeployed bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all L1 blockchain configurations",
		Long:  `List all sovereign L1 blockchain configurations and their deployment status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get all subnet names (which will become L1s)
			l1s, err := app.GetSidecarNames()
			if err != nil {
				return err
			}

			if len(l1s) == 0 {
				ux.Logger.PrintToUser("No L1 blockchains found.")
				ux.Logger.PrintToUser("\nCreate your first L1:")
				ux.Logger.PrintToUser("  lux l1 create myL1")
				ux.Logger.PrintToUser("  lux l1 import-historic  # Import LUX, ZOO, SPC as L1s")
				return nil
			}

			ux.Logger.PrintToUser("Sovereign L1 Blockchains:")
			ux.Logger.PrintToUser("")

			for _, l1Name := range l1s {
				sc, err := app.LoadSidecar(l1Name)
				if err != nil {
					continue
				}

				ux.Logger.PrintToUser("🔷 %s", l1Name)
				ux.Logger.PrintToUser("   Chain ID: %s", sc.ChainID)
				ux.Logger.PrintToUser("   Token: %s (%s)", sc.TokenInfo.Name, sc.TokenInfo.Symbol)
				ux.Logger.PrintToUser("   Validator Type: %s", sc.ValidatorManagement)

				if showDeployed && sc.BlockchainID.String() != "" {
					ux.Logger.PrintToUser("   Blockchain ID: %s", sc.BlockchainID)
					ux.Logger.PrintToUser("   Status: Deployed")

					// Check which protocols this L1 supports
					ux.Logger.PrintToUser("   Protocols: Lux")
					// Additional protocol compatibility (OP Stack, Arbitrum, etc.) will be shown when detection is implemented
				}

				ux.Logger.PrintToUser("")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showDeployed, "deployed", "d", false, "Show deployment information")

	return cmd
}
