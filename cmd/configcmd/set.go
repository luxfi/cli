// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package configcmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/luxfi/cli/pkg/globalconfig"
	"github.com/spf13/cobra"
)

var setProject bool

func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value in the global or project config.

Keys use dot notation for nested values:
  local.numNodes       - Number of nodes for local network
  local.autoTrackSubnets - Auto-track subnets in dev mode
  network.defaultNetwork - Default network (local, testnet, mainnet)
  network.luxdVersion   - Default luxd version
  evm.defaultTokenName  - Default token name for EVM chains
  evm.defaultTokenSymbol - Default token symbol
  evm.defaultTokenSupply - Default token supply
  staking.bootstrapValidatorBalance - Bootstrap validator balance
  staking.bootstrapValidatorWeight  - Bootstrap validator weight
  node.defaultInstanceType - Default instance type
  node.defaultRegion      - Default region

Examples:
  lux config set local.numNodes 3
  lux config set evm.defaultTokenName "MyToken"
  lux config set --project local.autoTrackSubnets true`,
		Args: cobra.ExactArgs(2),
		RunE: runSet,
	}

	cmd.Flags().BoolVar(&setProject, "project", false, "Set value in project config instead of global")

	return cmd
}

func runSet(_ *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if setProject {
		return setProjectValue(key, value)
	}
	return setGlobalValue(key, value)
}

func setGlobalValue(key, value string) error {
	baseDir := app.GetBaseDir()

	config, err := globalconfig.LoadGlobalConfig(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if config == nil {
		config = &globalconfig.GlobalConfig{Version: globalconfig.ConfigVersion}
	}

	if err := applyConfigValue(config, key, value); err != nil {
		return err
	}

	if err := globalconfig.SaveGlobalConfig(baseDir, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s = %s in global config\n", key, value)
	return nil
}

func setProjectValue(key, value string) error {
	config, err := globalconfig.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if config == nil {
		config = &globalconfig.ProjectConfig{
			GlobalConfig: globalconfig.GlobalConfig{Version: globalconfig.ConfigVersion},
		}
	}

	if err := applyConfigValue(&config.GlobalConfig, key, value); err != nil {
		return err
	}

	if err := globalconfig.SaveProjectConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s = %s in project config\n", key, value)
	return nil
}

func applyConfigValue(config *globalconfig.GlobalConfig, key, value string) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid key format: use section.setting (e.g., local.numNodes)")
	}

	section := parts[0]
	setting := parts[1]

	switch section {
	case "local":
		return applyLocalSetting(config, setting, value)
	case "network":
		return applyNetworkSetting(config, setting, value)
	case "evm":
		return applyEVMSetting(config, setting, value)
	case "staking":
		return applyStakingSetting(config, setting, value)
	case "node":
		return applyNodeSetting(config, setting, value)
	default:
		return fmt.Errorf("unknown section: %s", section)
	}
}

func applyLocalSetting(config *globalconfig.GlobalConfig, setting, value string) error {
	switch setting {
	case "numNodes":
		n, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid numNodes value: %w", err)
		}
		num := uint32(n)
		config.Local.NumNodes = &num
	case "autoTrackSubnets":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid autoTrackSubnets value: %w", err)
		}
		config.Local.AutoTrackSubnets = &b
	default:
		return fmt.Errorf("unknown local setting: %s", setting)
	}
	return nil
}

func applyNetworkSetting(config *globalconfig.GlobalConfig, setting, value string) error {
	switch setting {
	case "defaultNetwork":
		config.Network.DefaultNetwork = value
	case "luxdVersion":
		config.Network.LuxdVersion = value
	default:
		return fmt.Errorf("unknown network setting: %s", setting)
	}
	return nil
}

func applyEVMSetting(config *globalconfig.GlobalConfig, setting, value string) error {
	switch setting {
	case "defaultTokenName":
		config.EVM.DefaultTokenName = value
	case "defaultTokenSymbol":
		config.EVM.DefaultTokenSymbol = value
	case "defaultTokenSupply":
		config.EVM.DefaultTokenSupply = value
	default:
		return fmt.Errorf("unknown evm setting: %s", setting)
	}
	return nil
}

func applyStakingSetting(config *globalconfig.GlobalConfig, setting, value string) error {
	switch setting {
	case "bootstrapValidatorBalance":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid bootstrapValidatorBalance value: %w", err)
		}
		config.Staking.BootstrapValidatorBalance = &f
	case "bootstrapValidatorWeight":
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid bootstrapValidatorWeight value: %w", err)
		}
		config.Staking.BootstrapValidatorWeight = &n
	default:
		return fmt.Errorf("unknown staking setting: %s", setting)
	}
	return nil
}

func applyNodeSetting(config *globalconfig.GlobalConfig, setting, value string) error {
	switch setting {
	case "defaultInstanceType":
		config.Node.DefaultInstanceType = value
	case "defaultRegion":
		config.Node.DefaultRegion = value
	default:
		return fmt.Errorf("unknown node setting: %s", setting)
	}
	return nil
}
