// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	GlobalConfigFile  = "config.json"
	ProjectConfigFile = ".luxconfig.json"
)

var (
	globalConfigCache *GlobalConfig
	cacheMu           sync.RWMutex
)

// LoadGlobalConfig loads the global config from ~/.lux/config.json
func LoadGlobalConfig(baseDir string) (*GlobalConfig, error) {
	cacheMu.RLock()
	if globalConfigCache != nil {
		defer cacheMu.RUnlock()
		return globalConfigCache, nil
	}
	cacheMu.RUnlock()

	configPath := filepath.Join(baseDir, GlobalConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	cacheMu.Lock()
	globalConfigCache = &config
	cacheMu.Unlock()

	return &config, nil
}

// SaveGlobalConfig saves the global config to ~/.lux/config.json
func SaveGlobalConfig(baseDir string, config *GlobalConfig) error {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}

	configPath := filepath.Join(baseDir, GlobalConfigFile)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	cacheMu.Lock()
	globalConfigCache = config
	cacheMu.Unlock()

	return os.WriteFile(configPath, data, 0o644)
}

// LoadProjectConfig loads the project config by searching upward from cwd
func LoadProjectConfig() (*ProjectConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	projectRoot, err := FindProjectRoot(cwd)
	if err != nil {
		return nil, nil // No project config found
	}

	configPath := filepath.Join(projectRoot, ProjectConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveProjectConfig saves the project config to .luxconfig.json in cwd
func SaveProjectConfig(config *ProjectConfig) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	configPath := filepath.Join(cwd, ProjectConfigFile)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o644)
}

// FindProjectRoot searches upward from startDir to find .luxconfig.json
func FindProjectRoot(startDir string) (string, error) {
	dir := startDir
	for {
		configPath := filepath.Join(dir, ProjectConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// ClearCache clears the global config cache
func ClearCache() {
	cacheMu.Lock()
	globalConfigCache = nil
	cacheMu.Unlock()
}
