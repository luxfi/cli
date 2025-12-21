// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vmcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var binaryPath string

func newLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <vm-name>",
		Short: "Link a VM binary to the plugins directory",
		Long: `Link a VM binary to the plugins directory.

Creates a symlink from ~/.lux/plugins/<vmid> to the specified binary path.
The VMID is calculated from the VM name (padded to 32 bytes, CB58 encoded).

The binary must exist and be executable.

Examples:
  lux vm link lux-evm --path ~/work/lux/evm/build/evm
  lux vm link "Lux EVM" --path /usr/local/bin/evm`,
		Args: cobra.ExactArgs(1),
		RunE: runLink,
	}

	cmd.Flags().StringVarP(&binaryPath, "path", "p", "", "Path to the VM binary (required)")
	_ = cmd.MarkFlagRequired("path")

	return cmd
}

func runLink(_ *cobra.Command, args []string) error {
	vmName := args[0]

	// Expand ~ in path
	expandedPath := utils.GetRealFilePath(binaryPath)

	// Resolve to absolute path
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate binary exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("binary not found: %s", absPath)
		}
		return fmt.Errorf("failed to stat binary: %w", err)
	}

	// Check if it's a regular file (not a directory)
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", absPath)
	}

	// Check if executable (user execute bit)
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("binary is not executable: %s", absPath)
	}

	// Calculate VMID
	vmID, err := utils.VMID(vmName)
	if err != nil {
		return fmt.Errorf("failed to calculate VMID: %w", err)
	}

	// Get plugins directory
	pluginDir := filepath.Join(app.GetBaseDir(), constants.PluginDir)

	// Ensure plugins directory exists
	if err := os.MkdirAll(pluginDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Symlink path
	symlinkPath := filepath.Join(pluginDir, vmID.String())

	// Atomic symlink update: remove old, create new
	// Using os.Remove + os.Symlink pattern (ln -sfn equivalent)
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	if err := os.Symlink(absPath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	ux.Logger.PrintToUser("VM linked successfully:")
	ux.Logger.PrintToUser("  Name:   %s", vmName)
	ux.Logger.PrintToUser("  VMID:   %s", vmID.String())
	ux.Logger.PrintToUser("  Path:   %s", absPath)
	ux.Logger.PrintToUser("  Plugin: %s", symlinkPath)

	return nil
}
