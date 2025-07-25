// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/precompile/allowlist"
	"github.com/luxfi/evm/precompile/contracts/deployerallowlist"
	"github.com/luxfi/evm/precompile/contracts/feemanager"
	"github.com/luxfi/evm/precompile/contracts/nativeminter"
	"github.com/luxfi/evm/precompile/contracts/rewardmanager"
	"github.com/luxfi/evm/precompile/contracts/txallowlist"
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/geth/common"
)

type Precompile string

const (
	NativeMint        = "Native Minting"
	ContractAllowList = "Contract Deployment Allow List"
	TxAllowList       = "Transaction Allow List"
	FeeManager        = "Manage Fee Settings"
	RewardManager     = "RewardManagerConfig"
)

func PrecompileToUpgradeString(p Precompile) string {
	switch p {
	case NativeMint:
		return "contractNativeMinterConfig"
	case ContractAllowList:
		return "contractDeployerAllowListConfig"
	case TxAllowList:
		return "txAllowListConfig"
	case FeeManager:
		return "feeManagerConfig"
	case RewardManager:
		return "rewardManagerConfig"
	default:
		return ""
	}
}

func configureRewardManager(app *application.Lux) (rewardmanager.Config, bool, error) {
	config := rewardmanager.Config{}
	adminPrompt := "Configure reward manager admins"
	enabledPrompt := "Configure reward manager enabled addresses"
	info := "\nThis precompile allows to configure the fee reward mechanism " +
		"on your subnet, including burning or sending fees.\nFor more information visit " +
		"https://docs.lux.network/subnets/customize-a-subnet#changing-fee-reward-mechanisms\n\n"

	admins, cancelled, err := getAddressList(adminPrompt, info, app)
	if err != nil || cancelled {
		return config, false, err
	}

	enabled, cancelled, err := getAddressList(enabledPrompt, info, app)
	if err != nil {
		return config, false, err
	}

	config.AllowListConfig = allowlist.AllowListConfig{
		AdminAddresses:   admins,
		EnabledAddresses: enabled,
	}
	zero := uint64(0)
	config.Upgrade = precompileconfig.Upgrade{
		BlockTimestamp: &zero,
	}
	config.InitialRewardConfig, err = ConfigureInitialRewardConfig(app)
	if err != nil {
		return config, false, err
	}

	return config, cancelled, nil
}

func ConfigureInitialRewardConfig(app *application.Lux) (*rewardmanager.InitialRewardConfig, error) {
	config := &rewardmanager.InitialRewardConfig{}

	burnPrompt := "Should fees be burnt?"
	burnFees, err := app.Prompt.CaptureYesNo(burnPrompt)
	if err != nil {
		return config, err
	}
	if burnFees {
		return config, nil
	}

	feeRcpdPrompt := "Allow block producers to claim fees?"
	allowFeeRecipients, err := app.Prompt.CaptureYesNo(feeRcpdPrompt)
	if err != nil {
		return config, err
	}
	if allowFeeRecipients {
		config.AllowFeeRecipients = true
		return config, nil
	}

	rewardPrompt := "Provide the address to which fees will be sent to"
	rewardAddress, err := app.Prompt.CaptureAddress(rewardPrompt)
	if err != nil {
		return config, err
	}
	config.RewardAddress = rewardAddress
	return config, nil
}

func getAddressList(initialPrompt string, info string, app *application.Lux) ([]common.Address, bool, error) {
	label := "Address"

	return prompts.CaptureListDecision(
		app.Prompt,
		initialPrompt,
		app.Prompt.CaptureAddress,
		"Enter Address ",
		label,
		info,
	)
}

func configureContractAllowList(app *application.Lux) (deployerallowlist.Config, bool, error) {
	config := deployerallowlist.Config{}
	adminPrompt := "Configure contract deployment admin allow list"
	enabledPrompt := "Configure contract deployment enabled addresses list"
	info := "\nThis precompile restricts who has the ability to deploy contracts " +
		"on your subnet.\nFor more information visit " +
		"https://docs.lux.network/subnets/customize-a-subnet/#restricting-smart-contract-deployers\n\n"

	admins, cancelled, err := getAddressList(adminPrompt, info, app)
	if err != nil || cancelled {
		return config, false, err
	}

	enabled, cancelled, err := getAddressList(enabledPrompt, info, app)
	if err != nil {
		return config, false, err
	}

	config.AllowListConfig = allowlist.AllowListConfig{
		AdminAddresses:   admins,
		EnabledAddresses: enabled,
	}
	zero := uint64(0)
	config.Upgrade = precompileconfig.Upgrade{
		BlockTimestamp: &zero,
	}

	return config, cancelled, nil
}

func configureTransactionAllowList(app *application.Lux) (txallowlist.Config, bool, error) {
	config := txallowlist.Config{}
	adminPrompt := "Configure transaction allow list admin addresses"
	enabledPrompt := "Configure transaction allow list enabled addresses"
	info := "\nThis precompile restricts who has the ability to issue transactions " +
		"on your subnet.\nFor more information visit " +
		"https://docs.lux.network/subnets/customize-a-subnet/#restricting-who-can-submit-transactions\n\n"

	admins, cancelled, err := getAddressList(adminPrompt, info, app)
	if err != nil || cancelled {
		return config, false, err
	}

	enabled, cancelled, err := getAddressList(enabledPrompt, info, app)
	if err != nil {
		return config, false, err
	}

	config.AllowListConfig = allowlist.AllowListConfig{
		AdminAddresses:   admins,
		EnabledAddresses: enabled,
	}
	zero := uint64(0)
	config.Upgrade = precompileconfig.Upgrade{
		BlockTimestamp: &zero,
	}

	return config, cancelled, nil
}

func configureMinterList(app *application.Lux) (nativeminter.Config, bool, error) {
	config := nativeminter.Config{}
	adminPrompt := "Configure native minting allow list"
	enabledPrompt := "Configure native minting enabled addresses"
	info := "\nThis precompile allows admins to permit designated contracts to mint the native token " +
		"on your subnet.\nFor more information visit " +
		"https://docs.lux.network/subnets/customize-a-subnet#minting-native-coins\n\n"

	admins, cancelled, err := getAddressList(adminPrompt, info, app)
	if err != nil || cancelled {
		return config, false, err
	}

	enabled, cancelled, err := getAddressList(enabledPrompt, info, app)
	if err != nil {
		return config, false, err
	}

	config.AllowListConfig = allowlist.AllowListConfig{
		AdminAddresses:   admins,
		EnabledAddresses: enabled,
	}
	zero := uint64(0)
	config.Upgrade = precompileconfig.Upgrade{
		BlockTimestamp: &zero,
	}

	return config, cancelled, nil
}

func configureFeeConfigAllowList(app *application.Lux) (feemanager.Config, bool, error) {
	config := feemanager.Config{}
	adminPrompt := "Configure fee manager allow list"
	enabledPrompt := "Configure native minting enabled addresses"
	info := "\nThis precompile allows admins to adjust chain gas and fee parameters without " +
		"performing a hardfork.\nFor more information visit " +
		"https://docs.lux.network/subnets/customize-a-subnet#configuring-dynamic-fees\n\n"

	admins, cancelled, err := getAddressList(adminPrompt, info, app)
	if err != nil || cancelled {
		return config, cancelled, err
	}

	enabled, cancelled, err := getAddressList(enabledPrompt, info, app)
	if err != nil {
		return config, false, err
	}

	config.AllowListConfig = allowlist.AllowListConfig{
		AdminAddresses:   admins,
		EnabledAddresses: enabled,
	}
	zero := uint64(0)
	config.Upgrade = precompileconfig.Upgrade{
		BlockTimestamp: &zero,
	}

	return config, cancelled, nil
}

func removePrecompile(arr []string, s string) ([]string, error) {
	for i, val := range arr {
		if val == s {
			return append(arr[:i], arr[i+1:]...), nil
		}
	}
	return arr, errors.New("string not in array")
}

func getPrecompiles(config params.ChainConfig, app *application.Lux) (
	params.ChainConfig,
	statemachine.StateDirection,
	error,
) {
	const cancel = "Cancel"

	first := true

	remainingPrecompiles := []string{NativeMint, ContractAllowList, TxAllowList, FeeManager, RewardManager, cancel}

	for {
		firstStr := "Advanced: Would you like to add a custom precompile to modify the EVM?"
		secondStr := "Would you like to add additional precompiles?"

		var promptStr string
		if promptStr = secondStr; first {
			promptStr = firstStr
			first = false
		}

		addPrecompile, err := app.Prompt.CaptureList(promptStr, []string{prompts.No, prompts.Yes, goBackMsg})
		if err != nil {
			return config, statemachine.Stop, err
		}

		switch addPrecompile {
		case prompts.No:
			return config, statemachine.Forward, nil
		case goBackMsg:
			return config, statemachine.Backward, nil
		}

		precompileDecision, err := app.Prompt.CaptureList(
			"Choose precompile",
			remainingPrecompiles,
		)
		if err != nil {
			return config, statemachine.Stop, err
		}

		switch precompileDecision {
		case NativeMint:
			mintConfig, cancelled, err := configureMinterList(app)
			if err != nil {
				return config, statemachine.Stop, err
			}
			if !cancelled {
				config.GenesisPrecompiles[nativeminter.ConfigKey] = &mintConfig
				remainingPrecompiles, err = removePrecompile(remainingPrecompiles, NativeMint)
				if err != nil {
					return config, statemachine.Stop, err
				}
			}
		case ContractAllowList:
			contractConfig, cancelled, err := configureContractAllowList(app)
			if err != nil {
				return config, statemachine.Stop, err
			}
			if !cancelled {
				config.GenesisPrecompiles[deployerallowlist.ConfigKey] = &contractConfig
				remainingPrecompiles, err = removePrecompile(remainingPrecompiles, ContractAllowList)
				if err != nil {
					return config, statemachine.Stop, err
				}
			}
		case TxAllowList:
			txConfig, cancelled, err := configureTransactionAllowList(app)
			if err != nil {
				return config, statemachine.Stop, err
			}
			if !cancelled {
				config.GenesisPrecompiles[txallowlist.ConfigKey] = &txConfig
				remainingPrecompiles, err = removePrecompile(remainingPrecompiles, TxAllowList)
				if err != nil {
					return config, statemachine.Stop, err
				}
			}
		case FeeManager:
			feeConfig, cancelled, err := configureFeeConfigAllowList(app)
			if err != nil {
				return config, statemachine.Stop, err
			}
			if !cancelled {
				config.GenesisPrecompiles[feemanager.ConfigKey] = &feeConfig
				remainingPrecompiles, err = removePrecompile(remainingPrecompiles, FeeManager)
				if err != nil {
					return config, statemachine.Stop, err
				}
			}
		case RewardManager:
			rewardManagerConfig, cancelled, err := configureRewardManager(app)
			if err != nil {
				return config, statemachine.Stop, err
			}
			if !cancelled {
				config.GenesisPrecompiles[rewardmanager.ConfigKey] = &rewardManagerConfig
				remainingPrecompiles, err = removePrecompile(remainingPrecompiles, RewardManager)
				if err != nil {
					return config, statemachine.Stop, err
				}
			}

		case cancel:
			return config, statemachine.Forward, nil
		}

		// When all precompiles have been added, the len of remainingPrecompiles will be 1
		// (the cancel option stays in the list). Safe to return.
		if len(remainingPrecompiles) == 1 {
			return config, statemachine.Forward, nil
		}
	}
}
