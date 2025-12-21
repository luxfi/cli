// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vmcmd

import (
	"fmt"
	"os"
	"path/filepath"

	luxconfig "github.com/luxfi/config"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newUnlinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unlink <vm-name>",
		Aliases: []string{"rm", "remove"},
		Short:   "Remove a VM symlink from the plugins directory",
		Long: `Remove a VM symlink from the plugins directory.

Removes the symlink at ~/.lux/plugins/<vmid> for the given VM name.

Examples:
  lux vm unlink lux-evm
  lux vm unlink "Lux EVM"`,
		Args: cobra.ExactArgs(1),
		RunE: runUnlink,
	}

	return cmd
}

func runUnlink(_ *cobra.Command, args []string) error {
	vmName := args[0]

	// Calculate VMID
	vmID, err := utils.VMID(vmName)
	if err != nil {
		return fmt.Errorf("failed to calculate VMID: %w", err)
	}

	// Get plugins directory using unified config
	pluginDir := luxconfig.ResolvePluginDir()

	// Symlink path
	symlinkPath := filepath.Join(pluginDir, vmID.String())

	// Check if symlink exists
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("VM '%s' (VMID: %s) is not linked", vmName, vmID.String())
		}
		return fmt.Errorf("failed to check symlink: %w", err)
	}

	// Verify it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("plugin at %s is not a symlink", symlinkPath)
	}

	// Get target before removing (for display)
	target, _ := os.Readlink(symlinkPath)

	// Remove the symlink
	if err := os.Remove(symlinkPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	ux.Logger.PrintToUser("VM unlinked successfully:")
	ux.Logger.PrintToUser("  Name:   %s", vmName)
	ux.Logger.PrintToUser("  VMID:   %s", vmID.String())
	if target != "" {
		ux.Logger.PrintToUser("  Was:    %s", target)
	}

	return nil
}
