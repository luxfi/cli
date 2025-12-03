// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

const (
	// ConfigVersion is the current version of the config schema
	ConfigVersion = "1.0.0"

	// Default local network settings
	DefaultNumNodes = uint32(5)

	// Default staking settings
	DefaultBootstrapValidatorBalance = float64(1000)
	DefaultBootstrapValidatorWeight  = uint64(20)

	// Default EVM settings
	DefaultTokenName   = "TEST"
	DefaultTokenSymbol = "TST"
	DefaultTokenSupply = "1000000000000000000000000" // 1 million tokens with 18 decimals

	// Default network
	DefaultNetwork = "local"

	// Default node settings
	DefaultInstanceType = "default"
	DefaultRegion       = "us-east-1"
)

// DefaultGlobalConfig returns a new GlobalConfig with default values
func DefaultGlobalConfig() GlobalConfig {
	metricsEnabled := true
	numNodes := DefaultNumNodes
	autoTrack := true
	validatorBalance := DefaultBootstrapValidatorBalance
	validatorWeight := DefaultBootstrapValidatorWeight

	return GlobalConfig{
		Version:        ConfigVersion,
		MetricsEnabled: &metricsEnabled,
		Network: NetworkConfig{
			DefaultNetwork: DefaultNetwork,
			LuxdVersion:    "latest",
		},
		Local: LocalConfig{
			NumNodes:         &numNodes,
			AutoTrackSubnets: &autoTrack,
		},
		EVM: EVMConfig{
			DefaultTokenName:   DefaultTokenName,
			DefaultTokenSymbol: DefaultTokenSymbol,
			DefaultTokenSupply: DefaultTokenSupply,
		},
		Staking: StakingConfig{
			BootstrapValidatorBalance: &validatorBalance,
			BootstrapValidatorWeight:  &validatorWeight,
		},
		Node: NodeConfig{
			DefaultInstanceType: DefaultInstanceType,
			DefaultRegion:       DefaultRegion,
		},
	}
}
