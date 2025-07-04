// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package plugins

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/config"
	"github.com/luxfi/node/utils/logging"
	"github.com/kardianos/osext"
	"github.com/shirou/gopsutil/process"
)

var (
	// env var for node data dir
	defaultUnexpandedDataDir = "$" + config.LuxGoDataDirVar
	// expected file name for the config
	// TODO should other file names be supported? e.g. conf.json, etc.
	defaultConfigFileName = "config.json"
	// expected name of the plugins dir
	defaultPluginDir = "plugins"
	// default dir where the binary is usually found
	defaultLuxgoBuildDir = filepath.Join("go", "src", "github.com", constants.AvaLabsOrg, constants.LuxGoRepoName, "build")
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
	// TODO: Any other dirs we want to scan?
	scanConfigDirs = append(scanConfigDirs,
		filepath.Join("/", "etc", constants.LuxGoRepoName),
		filepath.Join("/", "usr", "local", "lib", constants.LuxGoRepoName),
		wd,
		home,
		filepath.Join(home, constants.LuxGoRepoName),
		filepath.Join(home, defaultLuxgoBuildDir),
		filepath.Join(home, ".node"),
		defaultUnexpandedDataDir,
	)
	return scanConfigDirs, nil
}

func FindPluginDir() (string, error) {
	ux.Logger.PrintToUser(logging.Yellow.Wrap("Scanning your system for the plugin directory..."))
	scanConfigDirs, err := getScanConfigDirs()
	if err != nil {
		return "", err
	}
	dir := findByCommonDirs(defaultPluginDir, scanConfigDirs)
	if dir != "" {
		return dir, nil
	}
	ux.Logger.PrintToUser(logging.Yellow.Wrap("No plugin directory found on your system"))
	return "", nil
}

func FindLuxConfigPath() (string, error) {
	ux.Logger.PrintToUser(logging.Yellow.Wrap("Scanning your system for existing files..."))
	var path string
	// Attempt 1: Try the admin API
	if path = findByRunningProcesses(constants.LuxGoRepoName, config.ConfigFileKey); path != "" {
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
	ux.Logger.PrintToUser(logging.Yellow.Wrap("No config file has been found on your system"))
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
