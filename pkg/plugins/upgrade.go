// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package plugins

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/models"
)

func ManualUpgrade(app *application.Lux, sc models.Sidecar, targetVersion string) error {
	vmid, err := sc.GetVMID()
	if err != nil {
		return err
	}
	pluginDir := app.GetTmpPluginDir()
	vmPath, err := CreatePluginFromVersion(app, sc.Name, sc.VM, targetVersion, vmid, pluginDir)
	if err != nil {
		return err
	}
	printUpgradeCmd(vmPath)
	return nil
}

func AutomatedUpgrade(app *application.Lux, sc models.Sidecar, targetVersion string, pluginDir string) error {
	// Attempt an automated update
	var err error
	if pluginDir == "" {
		pluginDir, err = FindPluginDir()
		if err != nil {
			return err
		}
		if pluginDir != "" {
			ux.Logger.PrintToUser(luxlog.Bold.Wrap(luxlog.Green.Wrap("Found the VM plugin directory at %s")), pluginDir)
			if !prompts.IsInteractive() {
				// In non-interactive mode, use the found directory
				ux.Logger.PrintToUser("Using found plugin directory (use --plugin-dir to specify a different path)")
			} else {
				yes, err := app.Prompt.CaptureYesNo("Is this where we should upgrade the VM?")
				if err != nil {
					return err
				}
				if yes {
					ux.Logger.PrintToUser("Will use plugin directory at %s to upgrade the VM", pluginDir)
				} else {
					pluginDir = ""
				}
			}
		}
		if pluginDir == "" {
			if !prompts.IsInteractive() {
				return fmt.Errorf("--plugin-dir is required when plugin directory cannot be auto-detected in non-interactive mode")
			}
			pluginDir, err = app.Prompt.CaptureString("Path to your node plugin dir (likely ~/.luxd/plugins)")
			if err != nil {
				return err
			}
		}
	}

	pluginDir, err = SanitizePath(pluginDir)
	if err != nil {
		return err
	}

	vmid, err := sc.GetVMID()
	if err != nil {
		return err
	}
	vmPath, err := CreatePluginFromVersion(app, sc.Name, sc.VM, targetVersion, vmid, pluginDir)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("VM binary written to %s", vmPath)

	return nil
}

func printUpgradeCmd(vmPath string) {
	msg := `
To upgrade your node, you must do three things:

1. Stop your node
2. Replace your VM binary in your node's plugin directory
3. Restart your node

To add the VM to your plugin directory, copy or scp from %s

If you installed node with the install script, your plugin directory is likely
~/.luxd/plugins.
`

	ux.Logger.PrintToUser(msg, vmPath)
}
