// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package configcmd

import (
	"fmt"
	"strings"

	"github.com/luxfi/cli/pkg/globalconfig"
	"github.com/spf13/cobra"
)

var getShowSource bool

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value, showing the effective value after merging all config sources.

Keys use dot notation for nested values:
  local.numNodes       - Number of nodes for local network
  local.autoTrackSubnets - Auto-track subnets in dev mode
  network.defaultNetwork - Default network
  evm.defaultTokenName  - Default token name for EVM chains

Use --source to also show where the value came from (default, global, project).

Examples:
  lux config get local.numNodes
  lux config get --source evm.defaultTokenName`,
		Args: cobra.ExactArgs(1),
		RunE: runGet,
	}

	cmd.Flags().BoolVar(&getShowSource, "source", false, "Show the source of the value")

	return cmd
}

func runGet(_ *cobra.Command, args []string) error {
	key := args[0]
	baseDir := app.GetBaseDir()

	merged, err := globalconfig.GetEffectiveConfig(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	value, source, err := getConfigValue(merged, key)
	if err != nil {
		return err
	}

	if getShowSource {
		fmt.Printf("%s = %s (source: %s)\n", key, value, source)
	} else {
		fmt.Printf("%s = %s\n", key, value)
	}

	return nil
}

func getConfigValue(merged *globalconfig.MergedConfig, key string) (string, globalconfig.ConfigSource, error) {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid key format: use section.setting (e.g., local.numNodes)")
	}

	section := parts[0]
	setting := parts[1]

	switch section {
	case "local":
		return getLocalValue(merged, setting)
	case "network":
		return getNetworkValue(merged, setting)
	case "evm":
		return getEVMValue(merged, setting)
	case "staking":
		return getStakingValue(merged, setting)
	case "node":
		return getNodeValue(merged, setting)
	default:
		return "", "", fmt.Errorf("unknown section: %s", section)
	}
}

func getLocalValue(merged *globalconfig.MergedConfig, setting string) (string, globalconfig.ConfigSource, error) {
	switch setting {
	case "numNodes":
		if merged.Config.Local.NumNodes != nil {
			return fmt.Sprintf("%d", *merged.Config.Local.NumNodes), merged.Sources.NumNodes, nil
		}
		return fmt.Sprintf("%d", globalconfig.DefaultNumNodes), globalconfig.SourceDefault, nil
	case "autoTrackSubnets":
		if merged.Config.Local.AutoTrackSubnets != nil {
			return fmt.Sprintf("%t", *merged.Config.Local.AutoTrackSubnets), merged.Sources.AutoTrackSubnets, nil
		}
		return "true", globalconfig.SourceDefault, nil
	default:
		return "", "", fmt.Errorf("unknown local setting: %s", setting)
	}
}

func getNetworkValue(merged *globalconfig.MergedConfig, setting string) (string, globalconfig.ConfigSource, error) {
	switch setting {
	case "defaultNetwork":
		return merged.Config.Network.DefaultNetwork, merged.Sources.DefaultNetwork, nil
	case "luxdVersion":
		return merged.Config.Network.LuxdVersion, merged.Sources.LuxdVersion, nil
	default:
		return "", "", fmt.Errorf("unknown network setting: %s", setting)
	}
}

func getEVMValue(merged *globalconfig.MergedConfig, setting string) (string, globalconfig.ConfigSource, error) {
	switch setting {
	case "defaultTokenName":
		return merged.Config.EVM.DefaultTokenName, merged.Sources.DefaultTokenName, nil
	case "defaultTokenSymbol":
		return merged.Config.EVM.DefaultTokenSymbol, merged.Sources.DefaultTokenSymbol, nil
	case "defaultTokenSupply":
		return merged.Config.EVM.DefaultTokenSupply, merged.Sources.DefaultTokenSupply, nil
	default:
		return "", "", fmt.Errorf("unknown evm setting: %s", setting)
	}
}

func getStakingValue(merged *globalconfig.MergedConfig, setting string) (string, globalconfig.ConfigSource, error) {
	switch setting {
	case "bootstrapValidatorBalance":
		if merged.Config.Staking.BootstrapValidatorBalance != nil {
			return fmt.Sprintf("%.2f", *merged.Config.Staking.BootstrapValidatorBalance), merged.Sources.BootstrapValidatorBalance, nil
		}
		return fmt.Sprintf("%.2f", globalconfig.DefaultBootstrapValidatorBalance), globalconfig.SourceDefault, nil
	case "bootstrapValidatorWeight":
		if merged.Config.Staking.BootstrapValidatorWeight != nil {
			return fmt.Sprintf("%d", *merged.Config.Staking.BootstrapValidatorWeight), merged.Sources.BootstrapValidatorWeight, nil
		}
		return fmt.Sprintf("%d", globalconfig.DefaultBootstrapValidatorWeight), globalconfig.SourceDefault, nil
	default:
		return "", "", fmt.Errorf("unknown staking setting: %s", setting)
	}
}

func getNodeValue(merged *globalconfig.MergedConfig, setting string) (string, globalconfig.ConfigSource, error) {
	switch setting {
	case "defaultInstanceType":
		return merged.Config.Node.DefaultInstanceType, merged.Sources.DefaultInstanceType, nil
	case "defaultRegion":
		return merged.Config.Node.DefaultRegion, merged.Sources.DefaultRegion, nil
	default:
		return "", "", fmt.Errorf("unknown node setting: %s", setting)
	}
}
