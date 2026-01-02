// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binpaths"
	"github.com/luxfi/cli/pkg/constants"
)

const (
	netrunnerBinPrefix = "netrunner-"
)

// SetupNetrunner downloads and sets up the netrunner binary.
// Returns the path to the installed binary.
func SetupNetrunner(app *application.Lux, version string) (string, error) {
	binDir := binpaths.GetBinDir()

	installer := NewInstaller()
	downloader := NewNetrunnerDownloader()
	return InstallBinary(
		app,
		version,
		binDir,
		binDir,
		netrunnerBinPrefix,
		constants.LuxOrg,
		constants.NetrunnerRepoName,
		downloader,
		installer,
	)
}

// EnsureNetrunnerBinary ensures the netrunner binary is available, downloading if necessary.
// Returns the path to the binary.
func EnsureNetrunnerBinary(app *application.Lux, version string) (string, error) {
	// First check if binary already exists at expected location
	path := binpaths.GetNetrunnerPath()
	if binpaths.Exists(path) {
		return path, nil
	}

	// Download the binary
	installedPath, err := SetupNetrunner(app, version)
	if err != nil {
		return "", err
	}

	// Find the actual binary within the installed directory
	binaryPath := filepath.Join(installedPath, "netrunner")
	if !binpaths.Exists(binaryPath) {
		// Binary might be at the installed path directly
		if binpaths.Exists(installedPath) {
			binaryPath = installedPath
		}
	}

	if err := os.Chmod(binaryPath, 0o755); err != nil { //nolint:gosec // G302: Executable needs 0755
		return "", err
	}

	return binaryPath, nil
}
