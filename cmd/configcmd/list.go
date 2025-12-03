// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package configcmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/globalconfig"
	"github.com/spf13/cobra"
)

var listShowSources bool

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Long: `List all configuration values with their effective values.

Shows the merged configuration from all sources (defaults, global, project).
Use --sources to see where each value comes from.`,
		RunE: runList,
	}

	cmd.Flags().BoolVar(&listShowSources, "sources", false, "Show the source of each value")

	return cmd
}

func runList(_ *cobra.Command, _ []string) error {
	baseDir := app.GetBaseDir()

	merged, err := globalconfig.GetEffectiveConfig(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Configuration:")
	fmt.Println()

	// Network settings
	fmt.Println("[network]")
	printValue("defaultNetwork", merged.Config.Network.DefaultNetwork, merged.Sources.DefaultNetwork)
	printValue("luxdVersion", merged.Config.Network.LuxdVersion, merged.Sources.LuxdVersion)
	fmt.Println()

	// Local settings
	fmt.Println("[local]")
	if merged.Config.Local.NumNodes != nil {
		printValue("numNodes", fmt.Sprintf("%d", *merged.Config.Local.NumNodes), merged.Sources.NumNodes)
	}
	if merged.Config.Local.AutoTrackSubnets != nil {
		printValue("autoTrackSubnets", fmt.Sprintf("%t", *merged.Config.Local.AutoTrackSubnets), merged.Sources.AutoTrackSubnets)
	}
	fmt.Println()

	// EVM settings
	fmt.Println("[evm]")
	printValue("defaultTokenName", merged.Config.EVM.DefaultTokenName, merged.Sources.DefaultTokenName)
	printValue("defaultTokenSymbol", merged.Config.EVM.DefaultTokenSymbol, merged.Sources.DefaultTokenSymbol)
	printValue("defaultTokenSupply", merged.Config.EVM.DefaultTokenSupply, merged.Sources.DefaultTokenSupply)
	fmt.Println()

	// Staking settings
	fmt.Println("[staking]")
	if merged.Config.Staking.BootstrapValidatorBalance != nil {
		printValue("bootstrapValidatorBalance", fmt.Sprintf("%.2f", *merged.Config.Staking.BootstrapValidatorBalance), merged.Sources.BootstrapValidatorBalance)
	}
	if merged.Config.Staking.BootstrapValidatorWeight != nil {
		printValue("bootstrapValidatorWeight", fmt.Sprintf("%d", *merged.Config.Staking.BootstrapValidatorWeight), merged.Sources.BootstrapValidatorWeight)
	}
	fmt.Println()

	// Node settings
	fmt.Println("[node]")
	printValue("defaultInstanceType", merged.Config.Node.DefaultInstanceType, merged.Sources.DefaultInstanceType)
	printValue("defaultRegion", merged.Config.Node.DefaultRegion, merged.Sources.DefaultRegion)

	return nil
}

func printValue(key, value string, source globalconfig.ConfigSource) {
	if listShowSources {
		fmt.Printf("  %s = %s  (%s)\n", key, value, source)
	} else {
		fmt.Printf("  %s = %s\n", key, value)
	}
}
