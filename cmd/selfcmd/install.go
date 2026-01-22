// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package selfcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
)

var currentVersion string

func newInstallCmd(version string) *cobra.Command {
	currentVersion = version

	cmd := &cobra.Command{
		Use:   "install [version]",
		Short: "Install a specific CLI version",
		Long: `Install a specific version of the Lux CLI.

Downloads and installs the specified version to ~/.lux/versions/<version>/.

If no version is specified, installs the latest version.

EXAMPLES:

  # Install latest version
  lux self install

  # Install specific version
  lux self install v1.22.5`,
		Args: cobra.MaximumNArgs(1),
		RunE: runSelfInstall,
	}

	return cmd
}

func runSelfInstall(_ *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Determine version to install
	var version string
	if len(args) > 0 {
		version = args[0]
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
	} else {
		// Get latest version
		url := binutils.GetGithubLatestReleaseURL(constants.LuxOrg, constants.CliRepoName)
		latest, err := app.Downloader.GetLatestReleaseVersion(url)
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
		version = latest
		ux.Logger.PrintToUser("Latest version: %s", version)
	}

	// Setup paths
	versionsDir := filepath.Join(home, constants.BaseDirName, "versions")
	versionDir := filepath.Join(versionsDir, version)

	// Check if already installed
	versionBinary := filepath.Join(versionDir, "lux")
	if _, err := os.Stat(versionBinary); err == nil {
		ux.Logger.PrintToUser("Version %s already installed at %s", version, versionDir)
		ux.Logger.PrintToUser("Use 'lux self use %s' to switch to it.", version)
		return nil
	}

	// Create version directory
	if err := os.MkdirAll(versionDir, 0o750); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	ux.Logger.PrintToUser("Installing Lux CLI %s...", version)

	// Download and install using the install script
	// curl -sSfL https://raw.githubusercontent.com/luxfi/cli/main/scripts/install.sh | sh -s -- -b <dir> <version>
	downloadCmd := exec.Command("curl", "-sSfL", constants.CliInstallationURL)
	installCmd := exec.Command("sh", "-s", "--", "-n", "-b", versionDir, version)

	installCmd.Stdin, err = downloadCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to setup pipe: %w", err)
	}

	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Start(); err != nil {
		return fmt.Errorf("failed to start install: %w", err)
	}

	if err := downloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	if err := installCmd.Wait(); err != nil {
		// Clean up failed install
		os.RemoveAll(versionDir)
		return fmt.Errorf("failed to install: %w", err)
	}

	ux.Logger.PrintToUser("Successfully installed Lux CLI %s", version)
	ux.Logger.PrintToUser("Use 'lux self use %s' to switch to it.", version)

	return nil
}
