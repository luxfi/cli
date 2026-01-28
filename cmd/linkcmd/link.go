// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package linkcmd provides a unified link command for all Lux binaries.
package linkcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
)

// Binary definitions
type binaryDef struct {
	name     string
	autoPath string // relative to workspace root
}

var binaries = map[string]binaryDef{
	"lux": {
		name:     "lux",
		autoPath: "cli/bin/lux",
	},
	"luxd": {
		name:     constants.NodeBinaryName,
		autoPath: "node/bin/" + constants.NodeBinaryName,
	},
	"netrunner": {
		name:     "netrunner",
		autoPath: "netrunner/bin/netrunner",
	},
}

// NewCmd creates the link command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link [binary] [path]",
		Short: "Link Lux binaries to ~/.lux/bin/",
		Long: `Link Lux binaries for system-wide use.

Creates ~/.lux/bin directory if needed and symlinks binaries.
Add ~/.lux/bin to your PATH for easy access.

SUPPORTED BINARIES:
  all        - All binaries (lux, luxd, netrunner)
  lux        - CLI binary
  luxd       - Node binary
  netrunner  - Network runner

EXAMPLES:

  # Link all binaries (auto-detect from workspace)
  lux link all

  # Link specific binary (auto-detect)
  lux link luxd
  lux link netrunner

  # Link specific binary with explicit path
  lux link luxd /path/to/luxd`,
		Args: cobra.MaximumNArgs(2),
		RunE: runLink,
	}

	return cmd
}

func runLink(_ *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	binDir := filepath.Join(home, constants.BaseDirName, constants.BinDir)

	// Create ~/.lux/bin directory
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		return fmt.Errorf("failed to create %s: %w", binDir, err)
	}

	// No args = show help
	if len(args) == 0 {
		return fmt.Errorf("specify binary to link: all, lux, luxd, netrunner")
	}

	// Link all binaries
	if args[0] == "all" {
		ux.Logger.PrintToUser("Linking all Lux binaries...")
		successCount := 0

		for name := range binaries {
			if err := linkBinary(name, "", binDir); err != nil {
				ux.Logger.PrintToUser("  %s: failed - %v", name, err)
			} else {
				successCount++
			}
		}

		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Linked %d/%d binaries to %s", successCount, len(binaries), binDir)
		return nil
	}

	// Link specific binary
	binaryName := args[0]
	var binaryPath string

	if _, ok := binaries[binaryName]; !ok {
		// First arg might be a path (backward compatible for luxd)
		binaryPath = args[0]
		binaryName = "luxd"
	} else if len(args) >= 2 {
		binaryPath = args[1]
	}

	return linkBinary(binaryName, binaryPath, binDir)
}

func linkBinary(name, explicitPath, binDir string) error {
	def, ok := binaries[name]
	if !ok {
		return fmt.Errorf("unknown binary: %s", name)
	}

	var binaryPath string
	var err error

	if explicitPath != "" {
		binaryPath, err = filepath.Abs(explicitPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
	} else {
		// Auto-detect: look relative to CLI executable
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get CLI executable path: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return fmt.Errorf("failed to resolve CLI symlinks: %w", err)
		}

		// CLI is at cli/bin/lux or ~/.lux/bin/lux, workspace is parent of cli
		cliDir := filepath.Dir(filepath.Dir(execPath))
		workspaceRoot := filepath.Dir(cliDir)

		// Try workspace relative path first
		binaryPath = filepath.Join(workspaceRoot, def.autoPath)
		binaryPath, err = filepath.Abs(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Validate binary exists and is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not found: %s", binaryPath)
		}
		return fmt.Errorf("failed to stat: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory: %s", binaryPath)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("not executable: %s", binaryPath)
	}

	// Create symlink
	linkPath := filepath.Join(binDir, def.name)

	// Check if already linked correctly
	if existingTarget, err := os.Readlink(linkPath); err == nil {
		if existingTarget == binaryPath {
			ux.Logger.PrintToUser("  %s: already linked", name)
			return nil
		}
	}

	// Remove existing symlink/file
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing: %w", err)
		}
	}

	if err := os.Symlink(binaryPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	ux.Logger.PrintToUser("  %s: %s -> %s", name, linkPath, binaryPath)
	return nil
}
