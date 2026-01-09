// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package plugins

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kardianos/osext"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/config"
	"github.com/luxfi/constantsants"
	luxlog "github.com/luxfi/log"
	"github.com/shirou/gopsutil/process"
)

var (
	// env var for node data dir
	defaultUnexpandedDataDir = "$" + config.LuxNodeDataDirVar
	// expected file name for the config
	defaultConfigFileName = "config.json"
	// expected name of the plugins dir
	defaultPluginDir = "plugins"
	// default dir where the binary is usually found
	defaultLuxBuildDir = filepath.Join("go", "src", "github.com", constants.LuxOrg, constants.LuxRepoName, "build")
)

// This function needs to be called to initialize this package
//
// this init is partly "borrowed" from node/config/config.go
func getScanConfigDirs() ([]string, error) {
	folderPath, err := osext.ExecutableFolder()
	scanConfigDirs := []string{}
	if err == nil {
		scanConfigDirs = append(scanConfigDirs, folderPath)
		scanConfigDirs = append(scanConfigDirs, filepath.Dir(folderPath))
	}
	wd, err := os.Getwd()
	if err != nil {
		return []string{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}, err
	}
	// Add additional directories to scan for config files
	// Include common configuration locations
	additionalDirs := []string{
		filepath.Join("/", "etc", constants.LuxRepoName),
		filepath.Join("/", "usr", "local", "lib", constants.LuxRepoName),
		filepath.Join("/", "opt", constants.LuxRepoName),
		filepath.Join("/", "var", "lib", constants.LuxRepoName),
		wd,
		home,
		filepath.Join(home, constants.LuxRepoName),
		filepath.Join(home, defaultLuxBuildDir),
		filepath.Join(home, ".luxd"),
		filepath.Join(home, ".config", constants.LuxRepoName),
		filepath.Join(home, ".local", "share", constants.LuxRepoName),
		defaultUnexpandedDataDir,
	}

	// Only add directories that exist to avoid noise
	for _, dir := range additionalDirs {
		if _, err := os.Stat(dir); err == nil {
			scanConfigDirs = append(scanConfigDirs, dir)
		}
	}
	return scanConfigDirs, nil
}

func FindPluginDir() (string, error) {
	ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("Scanning your system for the plugin directory..."))
	scanConfigDirs, err := getScanConfigDirs()
	if err != nil {
		return "", err
	}
	dir := findByCommonDirs(defaultPluginDir, scanConfigDirs)
	if dir != "" {
		return dir, nil
	}
	ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("No plugin directory found on your system"))
	return "", nil
}

func FindLuxConfigPath() (string, error) {
	ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("Scanning your system for existing files..."))
	var path string
	// Attempt 1: Try the admin API
	if path = findByRunningProcesses(constants.LuxRepoName, config.ConfigFileKey); path != "" {
		return path, nil
	}
	// Attempt 2: find looking at some usual dirs
	scanConfigDirs, err := getScanConfigDirs()
	if err != nil {
		return "", err
	}
	if path = findByCommonDirs(defaultConfigFileName, scanConfigDirs); path != "" {
		return path, nil
	}
	ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("No config file has been found on your system"))
	return "", nil
}

func findByCommonDirs(filename string, scanDirs []string) string {
	for _, d := range scanDirs {
		if d == defaultUnexpandedDataDir {
			d = os.ExpandEnv(d)
		}
		path := filepath.Join(d, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func findByRunningProcesses(procName, key string) string {
	procs, err := process.Processes()
	if err != nil {
		return ""
	}
	regex, err := regexp.Compile(procName + ".*" + key)
	if err != nil {
		return ""
	}
	for _, p := range procs {
		name, err := p.Cmdline()
		if err != nil {
			// ignore errors for processes that just died (macos implementation)
			continue
		}
		if regex.MatchString(name) {
			// truncate at end of `--config-file` + 1 (ignores if = or space)
			trunc := name[strings.Index(name, key)+len(key)+1:]
			// there might be other params after the config file entry, so split those away
			// first entry is the value of configFileKey
			return strings.Split(trunc, " ")[0]
		}
	}
	return ""
}
