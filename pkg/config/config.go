// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"encoding/json"

	"github.com/spf13/viper"
)

const (
	// LuxDataDirVar is the environment variable for the Lux data directory
	LuxDataDirVar = "LUXD_DATA_DIR"
)

type Config struct {
	MetricsEnabled bool `json:"metricsEnabled"`
}

func New() *Config {
	return &Config{}
}

func (*Config) LoadNodeConfig() (string, error) {
	globalConfigs := viper.GetStringMap("node-config")
	if len(globalConfigs) == 0 {
		return "", nil
	}
	configStr, err := json.Marshal(globalConfigs)
	if err != nil {
		return "", err
	}
	return string(configStr), nil
}

func (*Config) GetConfigStringValue(key string) string {
	return viper.GetString(key)
}

func (*Config) ConfigValueIsSet(key string) bool {
	return viper.IsSet(key)
}

func (*Config) ConfigFileExists() bool {
	return viper.ConfigFileUsed() != ""
}

func (*Config) GetConfigBoolValue(key string) bool {
	return viper.GetBool(key)
}

func (*Config) SetConfigValue(key string, value interface{}) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}

// GetConfigPath returns the path to the configuration file
func (*Config) GetConfigPath() string {
	return viper.ConfigFileUsed()
}
