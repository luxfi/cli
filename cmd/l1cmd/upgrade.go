// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/prompts"
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

var upgradeVMVersion string

func newUpgradeVMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm [l1Name]",
		Short: "Upgrade L1 VM version",
		Long: `Upgrade the VM version for an L1 blockchain.

NON-INTERACTIVE MODE:
  Use --version to specify the new VM version without prompting.

EXAMPLES:
  lux l1 upgrade vm mychain --version v1.2.3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]

			sc, err := app.LoadSidecar(l1Name)
			if err != nil {
				return err
			}

			ux.Logger.PrintToUser("Current VM version: %s", sc.VMVersion)

			newVersion := upgradeVMVersion
			if newVersion == "" {
				if !prompts.IsInteractive() {
					return fmt.Errorf("--version is required in non-interactive mode")
				}
				newVersion, err = app.Prompt.CaptureString("Enter new VM version")
				if err != nil {
					return err
				}
			}

			// Update VM version in sidecar
			sc.VMVersion = newVersion
			if err := app.UpdateSidecar(&sc); err != nil {
				return err
			}

			ux.Logger.PrintToUser("VM upgraded to version %s", newVersion)
			ux.Logger.PrintToUser("Please restart your validators to apply the upgrade")
			return nil
		},
	}
	cmd.Flags().StringVar(&upgradeVMVersion, "version", "", "New VM version to upgrade to")
	return cmd
}

var (
	upgradeToPoS        bool
	upgradeMinStake     uint64
	upgradeRewardRate   float64
	upgradeDelegation   bool
	upgradeNoDelegation bool
)

func newUpgradeValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-management [l1Name]",
		Short: "Upgrade validator management (e.g., PoA to PoS)",
		Long: `Upgrade the validator management system of an L1.

Common upgrades:
- PoA to PoS: Transition from authority-based to stake-based validation
- Update PoS parameters: Change staking requirements, rewards, etc.

NON-INTERACTIVE MODE:
  Use flags to provide all parameters:
  --to-pos              Migrate to Proof of Stake
  --min-stake           Minimum stake required (in tokens)
  --reward-rate         Annual reward rate (%)
  --delegation          Enable delegation
  --no-delegation       Disable delegation

EXAMPLES:
  lux l1 upgrade validator-management mychain --to-pos --min-stake 1000 --reward-rate 5.0 --delegation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]

			sc, err := app.LoadSidecar(l1Name)
			if err != nil {
				return err
			}

			ux.Logger.PrintToUser("Current validator management: %s", sc.ValidatorManagement)

			if sc.ValidatorManagement == "proof-of-authority" {
				migrate := upgradeToPoS
				if !migrate {
					if !prompts.IsInteractive() {
						ux.Logger.PrintToUser("Use --to-pos to migrate to Proof of Stake")
						return nil
					}
					ux.Logger.PrintToUser("\nAvailable upgrades:")
					ux.Logger.PrintToUser("1. Migrate to Proof of Stake")
					ux.Logger.PrintToUser("   - Enable permissionless validation")
					ux.Logger.PrintToUser("   - Implement token staking")
					ux.Logger.PrintToUser("   - Add delegation support")

					migrate, err = app.Prompt.CaptureYesNo("Migrate to Proof of Stake?")
					if err != nil {
						return err
					}
				}

				if migrate {
					ux.Logger.PrintToUser("\nPoS Migration Parameters:")

					// Capture staking parameters
					minStake := upgradeMinStake
					if minStake == 0 {
						if !prompts.IsInteractive() {
							return fmt.Errorf("--min-stake is required for PoS migration in non-interactive mode")
						}
						minStake, err = app.Prompt.CaptureUint64("Minimum stake required (in tokens)")
						if err != nil {
							return err
						}
					}

					rewardRate := upgradeRewardRate
					if rewardRate == 0 {
						if !prompts.IsInteractive() {
							return fmt.Errorf("--reward-rate is required for PoS migration in non-interactive mode")
						}
						rewardRate, err = app.Prompt.CaptureFloat("Annual reward rate (%)")
						if err != nil {
							return err
						}
					}

					enableDelegation := upgradeDelegation
					if !upgradeDelegation && !upgradeNoDelegation {
						if !prompts.IsInteractive() {
							return fmt.Errorf("--delegation or --no-delegation is required for PoS migration in non-interactive mode")
						}
						enableDelegation, err = app.Prompt.CaptureYesNo("Enable delegation?")
						if err != nil {
							return err
						}
					}

					// Update validator management in sidecar
					sc.ValidatorManagement = "proof-of-stake"
					sc.MinStake = minStake
					sc.RewardRate = rewardRate
					sc.DelegationEnabled = enableDelegation

					if err := app.UpdateSidecar(&sc); err != nil {
						return err
					}

					ux.Logger.PrintToUser("\nSuccessfully migrated to Proof of Stake!")
					ux.Logger.PrintToUser("Validators can now stake tokens to participate in consensus")
				}
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&upgradeToPoS, "to-pos", false, "Migrate to Proof of Stake")
	cmd.Flags().Uint64Var(&upgradeMinStake, "min-stake", 0, "Minimum stake required (in tokens)")
	cmd.Flags().Float64Var(&upgradeRewardRate, "reward-rate", 0, "Annual reward rate (%)")
	cmd.Flags().BoolVar(&upgradeDelegation, "delegation", false, "Enable delegation")
	cmd.Flags().BoolVar(&upgradeNoDelegation, "no-delegation", false, "Disable delegation")
	return cmd
}

var upgradeProtocol string

func newUpgradeProtocolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protocol [l1Name]",
		Short: "Add protocol support to L1",
		Long: `Add support for additional protocols to your L1:
- lux: Enable Lux subnet compatibility
- opstack: Enable OP Stack L2/L3 support
- cosmos: Enable IBC compatibility

NON-INTERACTIVE MODE:
  Use --protocol to specify which protocol to enable.
  Valid values: lux, opstack, cosmos, ethereum

EXAMPLES:
  lux l1 upgrade protocol mychain --protocol lux
  lux l1 upgrade protocol mychain --protocol opstack`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l1Name := args[0]

			protocols := []string{
				"Lux Compatibility",
				"OP Stack Support",
				"Cosmos IBC",
				"Ethereum Bridge",
			}

			choice := ""
			switch upgradeProtocol {
			case "lux":
				choice = "Lux Compatibility"
			case "opstack":
				choice = "OP Stack Support"
			case "cosmos":
				choice = "Cosmos IBC"
			case "ethereum":
				choice = "Ethereum Bridge"
			case "":
				if !prompts.IsInteractive() {
					return fmt.Errorf("--protocol is required in non-interactive mode (valid: lux, opstack, cosmos, ethereum)")
				}
				var err error
				choice, err = app.Prompt.CaptureList(
					"Choose protocol to add",
					protocols,
				)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown protocol: %s (valid: lux, opstack, cosmos, ethereum)", upgradeProtocol)
			}

			switch choice {
			case "Lux Compatibility":
				ux.Logger.PrintToUser("\nEnabling Lux compatibility...")
				ux.Logger.PrintToUser("This allows your L1 to:")
				ux.Logger.PrintToUser("- Accept Lux subnet validators")
				ux.Logger.PrintToUser("- Support Lux Warp messaging")
				ux.Logger.PrintToUser("- Bridge with Lux C-Chain")

				// Load and update sidecar
				sc, err := app.LoadSidecar(l1Name)
				if err != nil {
					return err
				}
				sc.LuxCompatible = true
				sc.WarpEnabled = true
				if err := app.UpdateSidecar(&sc); err != nil {
					return err
				}
				ux.Logger.PrintToUser("\nLux compatibility enabled!")

			case "OP Stack Support":
				ux.Logger.PrintToUser("\nEnabling OP Stack support...")
				ux.Logger.PrintToUser("This allows your L1 to:")
				ux.Logger.PrintToUser("- Host OP Stack L2s")
				ux.Logger.PrintToUser("- Use optimistic rollup technology")
				ux.Logger.PrintToUser("- Ethereum-compatible L2 scaling")

				// Load and update sidecar
				sc, err := app.LoadSidecar(l1Name)
				if err != nil {
					return err
				}
				sc.OPStackEnabled = true
				sc.RollupSupport = true
				if err := app.UpdateSidecar(&sc); err != nil {
					return err
				}
				ux.Logger.PrintToUser("\nOP Stack support enabled!")
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&upgradeProtocol, "protocol", "", "Protocol to enable (lux, opstack, cosmos, ethereum)")
	return cmd
}
