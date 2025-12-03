// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

// GlobalConfig represents the global configuration stored in ~/.lux/config.json
type GlobalConfig struct {
	Version        string         `json:"version"`
	MetricsEnabled *bool          `json:"metricsEnabled,omitempty"`
	Network        NetworkConfig  `json:"network"`
	Local          LocalConfig    `json:"local"`
	EVM            EVMConfig      `json:"evm"`
	Staking        StakingConfig  `json:"staking"`
	Node           NodeConfig     `json:"node"`
}

// NetworkConfig contains network-related settings
type NetworkConfig struct {
	DefaultNetwork string `json:"defaultNetwork,omitempty"`
	LuxdVersion    string `json:"luxdVersion,omitempty"`
}

// LocalConfig contains local network settings
type LocalConfig struct {
	NumNodes         *uint32 `json:"numNodes,omitempty"`
	AutoTrackSubnets *bool   `json:"autoTrackSubnets,omitempty"`
}

// EVMConfig contains EVM chain defaults
type EVMConfig struct {
	DefaultTokenName   string `json:"defaultTokenName,omitempty"`
	DefaultTokenSymbol string `json:"defaultTokenSymbol,omitempty"`
	DefaultTokenSupply string `json:"defaultTokenSupply,omitempty"`
	DefaultChainID     uint64 `json:"defaultChainId,omitempty"`
}

// StakingConfig contains staking defaults
type StakingConfig struct {
	BootstrapValidatorBalance *float64 `json:"bootstrapValidatorBalance,omitempty"`
	BootstrapValidatorWeight  *uint64  `json:"bootstrapValidatorWeight,omitempty"`
}

// NodeConfig contains node deployment defaults
type NodeConfig struct {
	DefaultInstanceType string `json:"defaultInstanceType,omitempty"`
	DefaultRegion       string `json:"defaultRegion,omitempty"`
}

// ProjectConfig represents project-local configuration in .luxconfig.json
type ProjectConfig struct {
	GlobalConfig
	ProjectName string `json:"projectName,omitempty"`
}
