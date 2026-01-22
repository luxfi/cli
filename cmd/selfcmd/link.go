// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package selfcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Symlink current CLI to ~/.lux/bin/lux",
		Long: `Link the currently running CLI binary to ~/.lux/bin/lux.

This makes the development build available system-wide when ~/.lux/bin
is in your PATH.

EXAMPLES:

  # Link current binary
  lux self link`,
		Args: cobra.NoArgs,
		RunE: runSelfLink,
	}

	return cmd
}

func runSelfLink(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	binDir := filepath.Join(home, constants.BaseDirName, constants.BinDir)

	// Create ~/.lux/bin directory
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		return fmt.Errorf("failed to create %s: %w", binDir, err)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks to get real path
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Make absolute
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Validate binary exists and is executable
	info, err := os.Stat(execPath)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", execPath)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("binary is not executable: %s", execPath)
	}

	// Create symlink
	linkPath := filepath.Join(binDir, "lux")

	// Check if we're already linked correctly
	if existingTarget, err := os.Readlink(linkPath); err == nil {
		if existingTarget == execPath {
			ux.Logger.PrintToUser("Already linked: %s -> %s", linkPath, execPath)
			return nil
		}
	}

	// Remove existing symlink/file if present
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing %s: %w", linkPath, err)
		}
	}

	if err := os.Symlink(execPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	ux.Logger.PrintToUser("lux CLI linked successfully:")
	ux.Logger.PrintToUser("  Source: %s", execPath)
	ux.Logger.PrintToUser("  Link:   %s", linkPath)

	return nil
}

// SelfLinkOnFirstRun checks if this is first run and links if needed.
// Call this from root command initialization.
func SelfLinkOnFirstRun() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil // Don't fail on first run check
	}

	linkPath := filepath.Join(home, constants.BaseDirName, constants.BinDir, "lux")

	// If link already exists, we're not on first run
	if _, err := os.Lstat(linkPath); err == nil {
		return nil
	}

	// Create the link silently on first run
	return runSelfLink(nil, nil)
}
