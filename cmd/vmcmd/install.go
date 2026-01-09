// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vmcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	luxconfig "github.com/luxfi/config"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
)

const (
	vmNameEVM = "evm"
	orgLuxfi  = "luxfi"
)

var installVersion string

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <org/name>[@version]",
		Short: "Install a VM plugin from GitHub releases",
		Long: `Install a VM plugin from GitHub releases.

Downloads the latest (or specified) version from GitHub releases and installs it
to ~/.lux/plugins/packages/<org>/<name>/<version>/.

Package format: <org>/<name> or <org>/<name>@<version>

Examples:
  lux vm install luxfi/evm           # Install latest
  lux vm install luxfi/evm@v1.0.0    # Install specific version
  lux vm install myuser/myvm         # Install from any org`,
		Args: cobra.ExactArgs(1),
		RunE: runInstall,
	}

	cmd.Flags().StringVarP(&installVersion, "version", "v", "", "Version to install (default: latest)")

	return cmd
}

func runInstall(_ *cobra.Command, args []string) error {
	pkgRef := args[0]

	// Parse org/name[@version]
	var org, name, version string

	// Check for @version in package reference
	if atIdx := strings.LastIndex(pkgRef, "@"); atIdx != -1 {
		version = pkgRef[atIdx+1:]
		pkgRef = pkgRef[:atIdx]
	}

	// Override with flag if provided
	if installVersion != "" {
		version = installVersion
	}

	// Parse org/name
	parts := strings.SplitN(pkgRef, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid package reference: %s (expected org/name)", pkgRef)
	}
	org, name = parts[0], parts[1]

	// Determine VM name for VMID calculation
	vmName := name
	if name == vmNameEVM && org == orgLuxfi {
		vmName = "Lux EVM" // Canonical name for Lux EVM
	}

	// Calculate VMID
	vmID, err := utils.VMID(vmName)
	if err != nil {
		return fmt.Errorf("failed to calculate VMID: %w", err)
	}

	// Get version if not specified
	downloader := application.NewDownloader()
	if version == "" {
		releaseURL := binutils.GetGithubLatestReleaseURL(org, name)
		ux.Logger.PrintToUser("Fetching latest version from %s/%s...", org, name)
		version, err = downloader.GetLatestReleaseVersion(releaseURL)
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
	}

	// Ensure version has v prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	ux.Logger.PrintToUser("Installing %s/%s@%s...", org, name, version)

	// Build download URL based on platform
	downloadURL, ext := getDownloadURL(org, name, version)

	ux.Logger.PrintToUser("Downloading from %s...", downloadURL)

	// Download the archive
	archive, err := downloader.Download(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Create temp directory for extraction
	tmpDir, err := os.MkdirTemp("", "lux-vm-install-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Extract archive
	if err := binutils.InstallArchive(ext, archive, tmpDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Find the binary in extracted contents
	binaryPath, err := findBinary(tmpDir, name)
	if err != nil {
		return fmt.Errorf("failed to find binary in archive: %w", err)
	}

	// Create package manager
	pm, err := luxconfig.NewPluginPackageManager("")
	if err != nil {
		return fmt.Errorf("failed to create package manager: %w", err)
	}

	// Create manifest
	manifest := &luxconfig.PluginManifest{
		Name:        name,
		Org:         org,
		Version:     version,
		VMID:        vmID.String(),
		VMName:      vmName,
		Binary:      filepath.Base(binaryPath),
		Description: fmt.Sprintf("%s/%s VM plugin", org, name),
		Repository:  fmt.Sprintf("https://github.com/%s/%s", org, name),
		InstalledAt: time.Now(),
	}

	// Install
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := pm.Install(ctx, manifest, binaryPath); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	ux.Logger.PrintToUser("Plugin installed successfully:")
	ux.Logger.PrintToUser("  Package:  %s/%s@%s", org, name, version)
	ux.Logger.PrintToUser("  VMID:     %s", vmID.String())
	ux.Logger.PrintToUser("  Location: %s", pm.PackagePath(org, name, version))
	ux.Logger.PrintToUser("  Active:   %s", pm.ActivePath(vmID.String()))

	return nil
}

// getDownloadURL builds the download URL for a package
func getDownloadURL(org, name, version string) (string, string) {
	goarch := runtime.GOARCH
	goos := runtime.GOOS

	var url string
	ext := "tar.gz"

	// Handle known packages with specific naming conventions
	if org == orgLuxfi && name == vmNameEVM {
		// Lux EVM has specific naming: evm_<version>_<os>_<arch>.tar.gz
		versionWithoutV := strings.TrimPrefix(version, "v")
		url = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_%s_%s.tar.gz",
			constants.LuxOrg,
			constants.EVMRepoName,
			version,
			constants.EVMRepoName,
			versionWithoutV,
			goos,
			goarch,
		)
	} else {
		// Generic naming convention: <name>_<version>_<os>_<arch>.tar.gz
		versionWithoutV := strings.TrimPrefix(version, "v")
		url = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_%s_%s.tar.gz",
			org,
			name,
			version,
			name,
			versionWithoutV,
			goos,
			goarch,
		)
	}

	return url, ext
}

// findBinary locates the VM binary in the extracted archive
func findBinary(dir string, name string) (string, error) {
	// Common binary names to search for
	searchNames := []string{
		name,
		strings.TrimSuffix(name, "-vm"),
		vmNameEVM, // For Lux EVM
	}

	var foundPath string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Check if this is an executable
		if info.Mode()&0o111 == 0 {
			return nil
		}

		baseName := filepath.Base(path)
		for _, searchName := range searchNames {
			if baseName == searchName {
				foundPath = path
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", err
	}

	if foundPath == "" {
		// If specific binary not found, look for any executable
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if info.Mode()&0o111 != 0 {
				foundPath = path
				return filepath.SkipAll
			}
			return nil
		})
		if err != nil && !errors.Is(err, filepath.SkipAll) {
			return "", err
		}
	}

	if foundPath == "" {
		return "", fmt.Errorf("no executable binary found in archive")
	}

	return foundPath, nil
}
