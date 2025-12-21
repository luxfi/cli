// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vmcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// VMInfo holds information about a linked VM
type VMInfo struct {
	Name   string `json:"name"`
	VMID   string `json:"vmid"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

var jsonOutput bool

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"list", "ls"},
		Short:   "Show all linked VMs",
		Long: `Show all linked VMs in the plugins directory.

Displays VMID, name (if known), target path, and whether the target exists.

Examples:
  lux vm status
  lux vm status --json`,
		Args: cobra.NoArgs,
		RunE: runStatus,
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func runStatus(_ *cobra.Command, _ []string) error {
	pluginDir := filepath.Join(app.GetBaseDir(), constants.PluginDir)

	// Check if plugins directory exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		ux.Logger.PrintToUser("No plugins directory found at %s", pluginDir)
		return nil
	}

	// Read all entries in the plugins directory
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	if len(entries) == 0 {
		ux.Logger.PrintToUser("No VMs linked.")
		ux.Logger.PrintToUser("Use 'lux vm link <vm-name> --path <path>' to link a VM.")
		return nil
	}

	var vms []VMInfo

	for _, entry := range entries {
		vmid := entry.Name()
		symlinkPath := filepath.Join(pluginDir, vmid)

		// Get the symlink target
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			// Not a symlink, skip
			continue
		}

		// Check if target exists
		_, statErr := os.Stat(target)
		exists := statErr == nil

		vms = append(vms, VMInfo{
			Name:   "", // We don't store the name, just the VMID
			VMID:   vmid,
			Path:   target,
			Exists: exists,
		})
	}

	if len(vms) == 0 {
		ux.Logger.PrintToUser("No VMs linked.")
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(vms, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Table output using tablewriter v1.0.9 API
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("VMID", "Path", "Status")

	for _, vm := range vms {
		status := "OK"
		if !vm.Exists {
			status = "MISSING"
		}
		_ = table.Append([]string{vm.VMID, vm.Path, status})
	}

	_ = table.Render()

	return nil
}
