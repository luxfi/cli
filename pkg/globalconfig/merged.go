// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

// ConfigSource indicates where a config value came from
type ConfigSource string

const (
	SourceDefault ConfigSource = "default"
	SourceGlobal  ConfigSource = "global"
	SourceProject ConfigSource = "project"
	SourceFlag    ConfigSource = "flag"
)

// MergedConfig holds the final merged configuration with source tracking
type MergedConfig struct {
	Config  GlobalConfig
	Sources ConfigSources
}

// ConfigSources tracks where each config value originated
type ConfigSources struct {
	NumNodes                  ConfigSource
	AutoTrackSubnets          ConfigSource
	DefaultNetwork            ConfigSource
	LuxdVersion               ConfigSource
	DefaultTokenName          ConfigSource
	DefaultTokenSymbol        ConfigSource
	DefaultTokenSupply        ConfigSource
	BootstrapValidatorBalance ConfigSource
	BootstrapValidatorWeight  ConfigSource
	DefaultInstanceType       ConfigSource
	DefaultRegion             ConfigSource
}

// Merge combines default, global, and project configs with proper precedence
// Hierarchy: project > global > defaults
func Merge(global *GlobalConfig, project *ProjectConfig) *MergedConfig {
	defaults := DefaultGlobalConfig()
	merged := &MergedConfig{
		Config: defaults,
		Sources: ConfigSources{
			NumNodes:                  SourceDefault,
			AutoTrackSubnets:          SourceDefault,
			DefaultNetwork:            SourceDefault,
			LuxdVersion:               SourceDefault,
			DefaultTokenName:          SourceDefault,
			DefaultTokenSymbol:        SourceDefault,
			DefaultTokenSupply:        SourceDefault,
			BootstrapValidatorBalance: SourceDefault,
			BootstrapValidatorWeight:  SourceDefault,
			DefaultInstanceType:       SourceDefault,
			DefaultRegion:             SourceDefault,
		},
	}

	// Apply global config
	if global != nil {
		mergeGlobalConfig(merged, global)
	}

	// Apply project config (highest precedence)
	if project != nil {
		mergeProjectConfig(merged, project)
	}

	return merged
}

func mergeGlobalConfig(merged *MergedConfig, global *GlobalConfig) {
	if global.MetricsEnabled != nil {
		merged.Config.MetricsEnabled = global.MetricsEnabled
	}

	// Network settings
	if global.Network.DefaultNetwork != "" {
		merged.Config.Network.DefaultNetwork = global.Network.DefaultNetwork
		merged.Sources.DefaultNetwork = SourceGlobal
	}
	if global.Network.LuxdVersion != "" {
		merged.Config.Network.LuxdVersion = global.Network.LuxdVersion
		merged.Sources.LuxdVersion = SourceGlobal
	}

	// Local settings
	if global.Local.NumNodes != nil {
		merged.Config.Local.NumNodes = global.Local.NumNodes
		merged.Sources.NumNodes = SourceGlobal
	}
	if global.Local.AutoTrackSubnets != nil {
		merged.Config.Local.AutoTrackSubnets = global.Local.AutoTrackSubnets
		merged.Sources.AutoTrackSubnets = SourceGlobal
	}

	// EVM settings
	if global.EVM.DefaultTokenName != "" {
		merged.Config.EVM.DefaultTokenName = global.EVM.DefaultTokenName
		merged.Sources.DefaultTokenName = SourceGlobal
	}
	if global.EVM.DefaultTokenSymbol != "" {
		merged.Config.EVM.DefaultTokenSymbol = global.EVM.DefaultTokenSymbol
		merged.Sources.DefaultTokenSymbol = SourceGlobal
	}
	if global.EVM.DefaultTokenSupply != "" {
		merged.Config.EVM.DefaultTokenSupply = global.EVM.DefaultTokenSupply
		merged.Sources.DefaultTokenSupply = SourceGlobal
	}

	// Staking settings
	if global.Staking.BootstrapValidatorBalance != nil {
		merged.Config.Staking.BootstrapValidatorBalance = global.Staking.BootstrapValidatorBalance
		merged.Sources.BootstrapValidatorBalance = SourceGlobal
	}
	if global.Staking.BootstrapValidatorWeight != nil {
		merged.Config.Staking.BootstrapValidatorWeight = global.Staking.BootstrapValidatorWeight
		merged.Sources.BootstrapValidatorWeight = SourceGlobal
	}

	// Node settings
	if global.Node.DefaultInstanceType != "" {
		merged.Config.Node.DefaultInstanceType = global.Node.DefaultInstanceType
		merged.Sources.DefaultInstanceType = SourceGlobal
	}
	if global.Node.DefaultRegion != "" {
		merged.Config.Node.DefaultRegion = global.Node.DefaultRegion
		merged.Sources.DefaultRegion = SourceGlobal
	}
}

func mergeProjectConfig(merged *MergedConfig, project *ProjectConfig) {
	if project.MetricsEnabled != nil {
		merged.Config.MetricsEnabled = project.MetricsEnabled
	}

	// Network settings
	if project.Network.DefaultNetwork != "" {
		merged.Config.Network.DefaultNetwork = project.Network.DefaultNetwork
		merged.Sources.DefaultNetwork = SourceProject
	}
	if project.Network.LuxdVersion != "" {
		merged.Config.Network.LuxdVersion = project.Network.LuxdVersion
		merged.Sources.LuxdVersion = SourceProject
	}

	// Local settings
	if project.Local.NumNodes != nil {
		merged.Config.Local.NumNodes = project.Local.NumNodes
		merged.Sources.NumNodes = SourceProject
	}
	if project.Local.AutoTrackSubnets != nil {
		merged.Config.Local.AutoTrackSubnets = project.Local.AutoTrackSubnets
		merged.Sources.AutoTrackSubnets = SourceProject
	}

	// EVM settings
	if project.EVM.DefaultTokenName != "" {
		merged.Config.EVM.DefaultTokenName = project.EVM.DefaultTokenName
		merged.Sources.DefaultTokenName = SourceProject
	}
	if project.EVM.DefaultTokenSymbol != "" {
		merged.Config.EVM.DefaultTokenSymbol = project.EVM.DefaultTokenSymbol
		merged.Sources.DefaultTokenSymbol = SourceProject
	}
	if project.EVM.DefaultTokenSupply != "" {
		merged.Config.EVM.DefaultTokenSupply = project.EVM.DefaultTokenSupply
		merged.Sources.DefaultTokenSupply = SourceProject
	}

	// Staking settings
	if project.Staking.BootstrapValidatorBalance != nil {
		merged.Config.Staking.BootstrapValidatorBalance = project.Staking.BootstrapValidatorBalance
		merged.Sources.BootstrapValidatorBalance = SourceProject
	}
	if project.Staking.BootstrapValidatorWeight != nil {
		merged.Config.Staking.BootstrapValidatorWeight = project.Staking.BootstrapValidatorWeight
		merged.Sources.BootstrapValidatorWeight = SourceProject
	}

	// Node settings
	if project.Node.DefaultInstanceType != "" {
		merged.Config.Node.DefaultInstanceType = project.Node.DefaultInstanceType
		merged.Sources.DefaultInstanceType = SourceProject
	}
	if project.Node.DefaultRegion != "" {
		merged.Config.Node.DefaultRegion = project.Node.DefaultRegion
		merged.Sources.DefaultRegion = SourceProject
	}
}

// GetEffectiveConfig loads and merges all config sources
func GetEffectiveConfig(baseDir string) (*MergedConfig, error) {
	global, err := LoadGlobalConfig(baseDir)
	if err != nil {
		return nil, err
	}

	project, err := LoadProjectConfig()
	if err != nil {
		return nil, err
	}

	return Merge(global, project), nil
}
