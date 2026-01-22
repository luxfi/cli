// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package netrunnercmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
)

const netrunnerBinaryName = "netrunner"

var autoDetect bool

func newLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link [path]",
		Short: "Symlink netrunner binary to ~/.lux/bin/",
		Long: `Link netrunner binary for the CLI to use.

Creates ~/.lux/bin directory if needed and symlinks the netrunner binary.

EXAMPLES:

  # Link netrunner (auto-detect from ../netrunner/bin/netrunner)
  lux netrunner link --auto

  # Link specific path
  lux netrunner link /path/to/netrunner`,
		Args: cobra.MaximumNArgs(1),
		RunE: runLinkNetrunner,
	}

	cmd.Flags().BoolVar(&autoDetect, "auto", false, "auto-detect netrunner from standard locations")

	return cmd
}

func runLinkNetrunner(_ *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	binDir := filepath.Join(home, constants.BaseDirName, constants.BinDir)

	// Create ~/.lux/bin directory
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		return fmt.Errorf("failed to create %s: %w", binDir, err)
	}

	var binaryPath string

	if len(args) >= 1 {
		binaryPath = utils.GetRealFilePath(args[0])
		binaryPath, err = filepath.Abs(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path: %w", err)
		}
	} else if autoDetect {
		// Auto-detect: look relative to CLI executable
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get CLI executable path: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return fmt.Errorf("failed to resolve CLI symlinks: %w", err)
		}
		// CLI is at cli/bin/lux, netrunner is at netrunner/bin/netrunner
		cliDir := filepath.Dir(filepath.Dir(execPath))
		binaryPath = filepath.Join(cliDir, "..", "netrunner", "bin", netrunnerBinaryName)
		binaryPath, err = filepath.Abs(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path: %w", err)
		}
	} else {
		return fmt.Errorf("specify path to netrunner binary or use --auto")
	}

	// Validate binary exists and is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("netrunner binary not found: %s", binaryPath)
		}
		return fmt.Errorf("failed to stat binary: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", binaryPath)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("binary is not executable: %s", binaryPath)
	}

	// Create symlink
	linkPath := filepath.Join(binDir, netrunnerBinaryName)

	// Remove existing symlink/file if present
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing %s: %w", linkPath, err)
		}
	}

	if err := os.Symlink(binaryPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	ux.Logger.PrintToUser("netrunner linked successfully:")
	ux.Logger.PrintToUser("  Source: %s", binaryPath)
	ux.Logger.PrintToUser("  Link:   %s", linkPath)

	return nil
}
