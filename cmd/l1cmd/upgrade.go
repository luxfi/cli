// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade L1 blockchain configuration",
		Long: `Upgrade various aspects of a sovereign L1 blockchain:
- VM version
- Validator management (PoA to PoS migration)
- Protocol support (add Lux compatibility, OP Stack, etc.)
- Network parameters`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	// upgrade vm
	cmd.AddCommand(newUpgradeVMCmd())
	// upgrade validator-management
	cmd.AddCommand(newUpgradeValidatorCmd())
	// upgrade protocol
	cmd.AddCommand(newUpgradeProtocolCmd())

	return cmd
}

func newUpgradeVMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm [l1Name]",
		Short: "Upgrade L1 VM version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]

			sc, err := app.LoadSidecar(l1Name)
			if err != nil {
				return err
			}

			ux.Logger.PrintToUser("Current VM version: %s", sc.VMVersion)

			newVersion, err := app.Prompt.CaptureString("Enter new VM version")
			if err != nil {
				return err
			}

			// Update VM version in sidecar
			sc.VMVersion = newVersion
			if err := app.UpdateSidecar(&sc); err != nil {
				return err
			}

			ux.Logger.PrintToUser("✅ VM upgraded to version %s", newVersion)
			ux.Logger.PrintToUser("Please restart your validators to apply the upgrade")
			return nil
		},
	}
	return cmd
}

func newUpgradeValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-management [l1Name]",
		Short: "Upgrade validator management (e.g., PoA to PoS)",
		Long: `Upgrade the validator management system of an L1.

Common upgrades:
- PoA to PoS: Transition from authority-based to stake-based validation
- Update PoS parameters: Change staking requirements, rewards, etc.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]

			sc, err := app.LoadSidecar(l1Name)
			if err != nil {
				return err
			}

			ux.Logger.PrintToUser("Current validator management: %s", sc.ValidatorManagement)

			if sc.ValidatorManagement == "proof-of-authority" {
				ux.Logger.PrintToUser("\n🔄 Available upgrades:")
				ux.Logger.PrintToUser("1. Migrate to Proof of Stake")
				ux.Logger.PrintToUser("   - Enable permissionless validation")
				ux.Logger.PrintToUser("   - Implement token staking")
				ux.Logger.PrintToUser("   - Add delegation support")

				migrate, err := app.Prompt.CaptureYesNo("Migrate to Proof of Stake?")
				if err != nil {
					return err
				}

				if migrate {
					ux.Logger.PrintToUser("\n📋 PoS Migration Parameters:")

					// Capture staking parameters
					minStake, err := app.Prompt.CaptureUint64("Minimum stake required (in tokens)")
					if err != nil {
						return err
					}

					rewardRate, err := app.Prompt.CaptureFloat("Annual reward rate (%)", nil)
					if err != nil {
						return err
					}

					enableDelegation, err := app.Prompt.CaptureYesNo("Enable delegation?")
					if err != nil {
						return err
					}

					// Update validator management in sidecar
					sc.ValidatorManagement = "proof-of-stake"
					sc.MinStake = minStake
					sc.RewardRate = rewardRate
					sc.DelegationEnabled = enableDelegation

					if err := app.UpdateSidecar(&sc); err != nil {
						return err
					}

					ux.Logger.PrintToUser("\n✅ Successfully migrated to Proof of Stake!")
					ux.Logger.PrintToUser("Validators can now stake tokens to participate in consensus")
				}
			}

			return nil
		},
	}
	return cmd
}

func newUpgradeProtocolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protocol [l1Name]",
		Short: "Add protocol support to L1",
		Long: `Add support for additional protocols to your L1:
- lux: Enable Lux subnet compatibility
- opstack: Enable OP Stack L2/L3 support
- cosmos: Enable IBC compatibility`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args[0] // l1Name already defined

			protocols := []string{
				"Lux Compatibility",
				"OP Stack Support",
				"Cosmos IBC",
				"Ethereum Bridge",
			}

			choice, err := app.Prompt.CaptureList(
				"Choose protocol to add",
				protocols,
			)
			if err != nil {
				return err
			}

			switch choice {
			case "Lux Compatibility":
				ux.Logger.PrintToUser("\n🔺 Enabling Lux compatibility...")
				ux.Logger.PrintToUser("This allows your L1 to:")
				ux.Logger.PrintToUser("- Accept Lux subnet validators")
				ux.Logger.PrintToUser("- Support Lux Warp messaging")
				ux.Logger.PrintToUser("- Bridge with Lux C-Chain")

				// Load and update sidecar
				sc, err := app.LoadSidecar(args[0])
				if err != nil {
					return err
				}
				sc.LuxCompatible = true
				sc.WarpEnabled = true
				if err := app.UpdateSidecar(&sc); err != nil {
					return err
				}
				ux.Logger.PrintToUser("\n✅ Lux compatibility enabled!")

			case "OP Stack Support":
				ux.Logger.PrintToUser("\n🟦 Enabling OP Stack support...")
				ux.Logger.PrintToUser("This allows your L1 to:")
				ux.Logger.PrintToUser("- Host OP Stack L2s")
				ux.Logger.PrintToUser("- Use optimistic rollup technology")
				ux.Logger.PrintToUser("- Ethereum-compatible L2 scaling")

				// Load and update sidecar
				sc, err := app.LoadSidecar(args[0])
				if err != nil {
					return err
				}
				sc.OPStackEnabled = true
				sc.RollupSupport = true
				if err := app.UpdateSidecar(&sc); err != nil {
					return err
				}
				ux.Logger.PrintToUser("\n✅ OP Stack support enabled!")
			}

			return nil
		},
	}
	return cmd
}
