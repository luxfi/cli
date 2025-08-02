// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	MetricsEnabled bool              `json:"metricsEnabled"`
	ConfigFile     string            // Path to config file
	ConfigData     map[string]interface{} // Stores configuration data
}

func New() *Config {
	return &Config{
		ConfigData: make(map[string]interface{}),
	}
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

// ConfigFileExists checks if the config file exists
func (c *Config) ConfigFileExists() bool {
	return c.ConfigFile != "" && fileExists(c.ConfigFile)
}

// ConfigValueIsSet checks if a config value is set
func (c *Config) ConfigValueIsSet(key string) bool {
	_, exists := c.ConfigData[key]
	return exists
}

// SetConfigValue sets a configuration value
func (c *Config) SetConfigValue(key string, value interface{}) error {
	c.ConfigData[key] = value
	return nil
}

// GetConfigStringValue gets a string configuration value
func (c *Config) GetConfigStringValue(key string) (string, error) {
	val, exists := c.ConfigData[key]
	if !exists {
		return "", nil
	}
	strVal, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("config value %s is not a string", key)
	}
	return strVal, nil
}

// GetConfigBoolValue gets a boolean configuration value
func (c *Config) GetConfigBoolValue(key string) (bool, error) {
	val, exists := c.ConfigData[key]
	if !exists {
		return false, nil
	}
	boolVal, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("config value %s is not a boolean", key)
	}
	return boolVal, nil
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
