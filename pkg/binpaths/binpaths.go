// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package binpaths provides utilities for resolving external binary paths
// from environment variables, config files, or default locations.
package binpaths

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luxfi/constantsants"
	"github.com/spf13/viper"
)

// GetNodePath returns the path to the luxd binary.
// Priority: ENV > config > PATH > default install location
func GetNodePath() string {
	// 1. Check environment variable
	if path := os.Getenv(constants.EnvNodePath); path != "" {
		return path
	}

	// 2. Check viper config
	if path := viper.GetString(constants.ConfigNodePath); path != "" {
		return path
	}

	// 3. Check if luxd is in PATH
	if path, err := exec.LookPath("luxd"); err == nil {
		return path
	}

	// 4. Default to ~/.lux/bin/luxd
	home, _ := os.UserHomeDir()
	return filepath.Join(home, constants.BaseDirName, constants.LuxCliBinDir, "luxd")
}

// GetNetrunnerPath returns the path to the netrunner binary.
// Priority: ENV > config > PATH > default install location
func GetNetrunnerPath() string {
	// 1. Check environment variable
	if path := os.Getenv(constants.EnvNetrunnerPath); path != "" {
		return path
	}

	// 2. Check viper config
	if path := viper.GetString(constants.ConfigNetrunnerPath); path != "" {
		return path
	}

	// 3. Check if netrunner is in PATH
	if path, err := exec.LookPath("netrunner"); err == nil {
		return path
	}

	// 4. Default to ~/.lux/bin/netrunner
	home, _ := os.UserHomeDir()
	return filepath.Join(home, constants.BaseDirName, constants.LuxCliBinDir, "netrunner")
}

// GetEVMPath returns the path to the EVM plugin binary.
// Priority: ENV > config > default install location
func GetEVMPath() string {
	// 1. Check environment variable
	if path := os.Getenv(constants.EnvEVMPath); path != "" {
		return path
	}

	// 2. Check viper config
	if path := viper.GetString(constants.ConfigEVMPath); path != "" {
		return path
	}

	// 3. Default to ~/.lux/bin/plugins/evm
	home, _ := os.UserHomeDir()
	return filepath.Join(home, constants.BaseDirName, constants.LuxCliBinDir, constants.PluginDir, constants.EVMBin)
}

// GetPluginsDir returns the path to the plugins directory.
// Priority: ENV > config > default location
func GetPluginsDir() string {
	// 1. Check environment variable
	if path := os.Getenv(constants.EnvPluginsDir); path != "" {
		return path
	}

	// 2. Check viper config
	if path := viper.GetString(constants.ConfigPluginsDir); path != "" {
		return path
	}

	// 3. Default to ~/.lux/bin/plugins
	home, _ := os.UserHomeDir()
	return filepath.Join(home, constants.BaseDirName, constants.LuxCliBinDir, constants.PluginDir)
}

// Exists checks if a binary exists at the given path
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureExecutable ensures the binary at path is executable
func EnsureExecutable(path string) error {
	return os.Chmod(path, 0o755) //nolint:gosec // G302: Executables need 0755 permissions
}

// GetBinDir returns the default binary installation directory
func GetBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, constants.BaseDirName, constants.LuxCliBinDir)
}
