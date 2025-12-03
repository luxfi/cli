// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

// GetNumNodes returns the effective number of nodes considering config hierarchy
// Priority: flagValue (if flagChanged) > project config > global config > smart defaults
func GetNumNodes(baseDir string, flagValue uint32, flagChanged bool) (uint32, error) {
	if flagChanged {
		return flagValue, nil
	}

	merged, err := GetEffectiveConfig(baseDir)
	if err != nil {
		return SuggestNumNodes(), nil
	}

	if merged.Config.Local.NumNodes != nil {
		return *merged.Config.Local.NumNodes, nil
	}

	return SuggestNumNodes(), nil
}

// GetLuxdVersion returns the effective luxd version considering config hierarchy
func GetLuxdVersion(baseDir string, flagValue string, flagChanged bool) (string, error) {
	if flagChanged && flagValue != "" {
		return flagValue, nil
	}

	merged, err := GetEffectiveConfig(baseDir)
	if err != nil {
		return "latest", nil
	}

	if merged.Config.Network.LuxdVersion != "" {
		return merged.Config.Network.LuxdVersion, nil
	}

	return "latest", nil
}

// GetAutoTrackSubnets returns whether subnets should be auto-tracked
func GetAutoTrackSubnets(baseDir string) (bool, error) {
	merged, err := GetEffectiveConfig(baseDir)
	if err != nil {
		return IsAutoTrackRecommended(), nil
	}

	if merged.Config.Local.AutoTrackSubnets != nil {
		return *merged.Config.Local.AutoTrackSubnets, nil
	}

	return IsAutoTrackRecommended(), nil
}

// GetDefaultTokenSupply returns the default token supply from config
func GetDefaultTokenSupply(baseDir string) (string, error) {
	merged, err := GetEffectiveConfig(baseDir)
	if err != nil {
		smart := GetSmartDefaults()
		return SuggestTokenSupply(smart.IsTestnet), nil
	}

	if merged.Config.EVM.DefaultTokenSupply != "" {
		return merged.Config.EVM.DefaultTokenSupply, nil
	}

	smart := GetSmartDefaults()
	return SuggestTokenSupply(smart.IsTestnet), nil
}

// GetBootstrapValidatorBalance returns the default bootstrap validator balance
func GetBootstrapValidatorBalance(baseDir string) (float64, error) {
	merged, err := GetEffectiveConfig(baseDir)
	if err != nil {
		return DefaultBootstrapValidatorBalance, nil
	}

	if merged.Config.Staking.BootstrapValidatorBalance != nil {
		return *merged.Config.Staking.BootstrapValidatorBalance, nil
	}

	return DefaultBootstrapValidatorBalance, nil
}

// GetBootstrapValidatorWeight returns the default bootstrap validator weight
func GetBootstrapValidatorWeight(baseDir string) (uint64, error) {
	merged, err := GetEffectiveConfig(baseDir)
	if err != nil {
		return DefaultBootstrapValidatorWeight, nil
	}

	if merged.Config.Staking.BootstrapValidatorWeight != nil {
		return *merged.Config.Staking.BootstrapValidatorWeight, nil
	}

	return DefaultBootstrapValidatorWeight, nil
}

// GetDefaultInstanceType returns the default instance type for node deployment
func GetDefaultInstanceType(baseDir string) (string, error) {
	merged, err := GetEffectiveConfig(baseDir)
	if err != nil {
		return SuggestInstanceType(), nil
	}

	if merged.Config.Node.DefaultInstanceType != "" {
		return merged.Config.Node.DefaultInstanceType, nil
	}

	return SuggestInstanceType(), nil
}
