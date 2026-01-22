// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package selfcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
)

func newUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a specific CLI version",
		Long: `Switch to a specific installed version of the Lux CLI.

Updates the ~/.lux/bin/lux symlink to point to the specified version.

Use 'dev' to switch back to your development build.

EXAMPLES:

  # Switch to a specific version
  lux self use v1.22.5

  # Switch back to development build
  lux self use dev`,
		Args: cobra.ExactArgs(1),
		RunE: runSelfUse,
	}

	return cmd
}

func runSelfUse(_ *cobra.Command, args []string) error {
	version := args[0]

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	binDir := filepath.Join(home, constants.BaseDirName, constants.BinDir)
	linkPath := filepath.Join(binDir, "lux")

	var targetPath string

	if version == "dev" || version == "development" {
		// Find the development build
		// Look in common locations
		devPaths := []string{
			filepath.Join(home, "work", "lux", "cli", "bin", "lux"),
			filepath.Join(home, "go", "bin", "lux"),
		}

		for _, p := range devPaths {
			if _, err := os.Stat(p); err == nil {
				targetPath = p
				break
			}
		}

		if targetPath == "" {
			return fmt.Errorf("could not find development build. Use 'lux self link' from your dev directory instead")
		}
	} else {
		// Normalize version
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}

		versionsDir := filepath.Join(home, constants.BaseDirName, "versions")
		versionBinary := filepath.Join(versionsDir, version, "lux")

		if _, err := os.Stat(versionBinary); os.IsNotExist(err) {
			ux.Logger.PrintToUser("Version %s not installed.", version)
			ux.Logger.PrintToUser("Use 'lux self install %s' to install it.", version)
			return fmt.Errorf("version not installed: %s", version)
		}

		targetPath = versionBinary
	}

	// Create bin directory if needed
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Remove existing symlink
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing link: %w", err)
		}
	}

	// Create new symlink
	if err := os.Symlink(targetPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	ux.Logger.PrintToUser("Switched to: %s", targetPath)
	ux.Logger.PrintToUser("")

	// Show version
	ux.Logger.PrintToUser("Now using:")
	/* #nosec G204 */
	// Version check intentionally limited to --version flag only
	out, _ := exec.Command(linkPath, "--version").Output()
	if len(out) > 0 {
		ux.Logger.PrintToUser("  %s", strings.TrimSpace(string(out)))
	}

	return nil
}
