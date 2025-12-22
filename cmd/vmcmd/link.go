// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vmcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	luxconfig "github.com/luxfi/config"
	"github.com/spf13/cobra"
)

var linkVersion string

func newLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <org/name> <path>",
		Short: "Link a local VM binary for development",
		Long: `Link a local VM binary to the plugins directory for development.

Creates a proper package entry and VMID symlink for a locally built VM binary.
Use this during development to test local builds with the node.

Package format: <org>/<name> (e.g., luxfi/evm, myuser/myvm)

The binary must exist and be executable.

Examples:
  lux vm link luxfi/evm ~/work/lux/evm/build/evm
  lux vm link luxfi/evm ~/work/lux/evm/build/evm --version v1.2.3-dev
  lux vm link myuser/myvm /path/to/myvm/build/myvm`,
		Args: cobra.ExactArgs(2),
		RunE: runLink,
	}

	cmd.Flags().StringVarP(&linkVersion, "version", "v", "", "Version label (default: v0.0.0-local)")

	return cmd
}

func runLink(_ *cobra.Command, args []string) error {
	pkgRef := args[0]
	binaryPath := args[1]

	// Parse org/name
	parts := strings.SplitN(pkgRef, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid package reference: %s (expected org/name)", pkgRef)
	}
	org, name := parts[0], parts[1]

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

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", absPath)
	}

	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("binary is not executable: %s", absPath)
	}

	// Determine version
	version := linkVersion
	if version == "" {
		version = "v0.0.0-local"
	}

	// Determine VM name for VMID calculation
	// Use canonical name based on common VMs, or default to package name
	vmName := name
	if name == "evm" && org == "luxfi" {
		vmName = "Lux EVM" // Canonical name for Lux EVM
	}

	// Calculate VMID
	vmID, err := utils.VMID(vmName)
	if err != nil {
		return fmt.Errorf("failed to calculate VMID: %w", err)
	}

	// Create package manager
	pm, err := luxconfig.NewPluginPackageManager("")
	if err != nil {
		return fmt.Errorf("failed to create package manager: %w", err)
	}

	// Create manifest (link creates a symlink, not a copy)
	manifest := &luxconfig.PluginManifest{
		Name:        name,
		Org:         org,
		Version:     version,
		VMID:        vmID.String(),
		VMName:      vmName,
		Binary:      filepath.Base(absPath),
		Description: fmt.Sprintf("%s/%s VM plugin (linked)", org, name),
		Repository:  fmt.Sprintf("https://github.com/%s/%s", org, name),
		InstalledAt: time.Now(),
	}

	// Link (creates symlink instead of copying)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pm.Link(ctx, manifest, absPath); err != nil {
		return fmt.Errorf("failed to link plugin: %w", err)
	}

	ux.Logger.PrintToUser("Plugin linked successfully:")
	ux.Logger.PrintToUser("  Package:  %s/%s@%s", org, name, version)
	ux.Logger.PrintToUser("  VMID:     %s", vmID.String())
	ux.Logger.PrintToUser("  Binary:   %s", absPath)
	ux.Logger.PrintToUser("  Active:   %s", pm.ActivePath(vmID.String()))

	return nil
}
