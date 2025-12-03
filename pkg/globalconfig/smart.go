// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

import (
	"os"
	"runtime"
)

// Environment represents the detected development environment
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvCI          Environment = "ci"
	EnvCodespace   Environment = "codespace"
	EnvProduction  Environment = "production"
)

// SmartDefaults provides intelligent default suggestions based on environment
type SmartDefaults struct {
	Environment       Environment
	SuggestedNumNodes uint32
	SuggestedInstance string
	IsTestnet         bool
}

// DetectEnvironment analyzes the current environment
func DetectEnvironment() Environment {
	// Check for CI environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("GITLAB_CI") != "" || os.Getenv("JENKINS_URL") != "" {
		return EnvCI
	}

	// Check for GitHub Codespaces
	if os.Getenv("CODESPACES") != "" || os.Getenv("CODESPACE_NAME") != "" {
		return EnvCodespace
	}

	// Check for production indicators
	if os.Getenv("PRODUCTION") != "" || os.Getenv("NODE_ENV") == "production" {
		return EnvProduction
	}

	return EnvDevelopment
}

// GetSmartDefaults returns environment-aware default configurations
func GetSmartDefaults() *SmartDefaults {
	env := DetectEnvironment()

	defaults := &SmartDefaults{
		Environment:       env,
		SuggestedNumNodes: DefaultNumNodes,
		SuggestedInstance: DefaultInstanceType,
		IsTestnet:         true,
	}

	switch env {
	case EnvCI:
		// CI environments typically need fewer nodes and faster execution
		defaults.SuggestedNumNodes = 3
		defaults.SuggestedInstance = "small"
	case EnvCodespace:
		// Codespaces have limited resources
		defaults.SuggestedNumNodes = 3
		defaults.SuggestedInstance = "small"
	case EnvProduction:
		// Production needs more robust setup
		defaults.SuggestedNumNodes = 5
		defaults.SuggestedInstance = "large"
		defaults.IsTestnet = false
	case EnvDevelopment:
		// Development uses standard defaults
		defaults.SuggestedNumNodes = 5
		defaults.SuggestedInstance = "default"
	}

	return defaults
}

// SuggestNumNodes returns the suggested number of nodes based on environment
func SuggestNumNodes() uint32 {
	smart := GetSmartDefaults()
	return smart.SuggestedNumNodes
}

// SuggestInstanceType returns the suggested instance type based on environment and resources
func SuggestInstanceType() string {
	smart := GetSmartDefaults()

	// Also consider available system resources
	numCPU := runtime.NumCPU()
	if numCPU <= 2 {
		return "small"
	} else if numCPU >= 8 {
		return "large"
	}

	return smart.SuggestedInstance
}

// SuggestTokenSupply returns the suggested token supply based on whether it's a testnet
func SuggestTokenSupply(isTestnet bool) string {
	if isTestnet {
		return DefaultTokenSupply // 1 million tokens
	}
	// For production, suggest a more conservative supply
	return "100000000000000000000000000" // 100 million tokens
}

// IsAutoTrackRecommended returns whether auto-tracking subnets is recommended
func IsAutoTrackRecommended() bool {
	env := DetectEnvironment()
	// Auto-track is recommended for development and CI
	return env == EnvDevelopment || env == EnvCI || env == EnvCodespace
}
