// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package selfcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed CLI versions",
		Long: `List all installed versions of the Lux CLI.

Shows installed versions and indicates which one is currently active.

EXAMPLES:

  lux self list`,
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE:    runSelfList,
	}

	return cmd
}

func runSelfList(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	versionsDir := filepath.Join(home, constants.BaseDirName, "versions")
	binDir := filepath.Join(home, constants.BaseDirName, constants.BinDir)
	activePath := filepath.Join(binDir, "lux")

	// Get active version target
	activeTarget, _ := os.Readlink(activePath)
	activeVersion := ""
	if strings.Contains(activeTarget, "versions/") {
		parts := strings.Split(activeTarget, "versions/")
		if len(parts) > 1 {
			vParts := strings.Split(parts[1], "/")
			if len(vParts) > 0 {
				activeVersion = vParts[0]
			}
		}
	}

	// Check if versions directory exists
	if _, err := os.Stat(versionsDir); os.IsNotExist(err) {
		ux.Logger.PrintToUser("No versions installed yet.")
		ux.Logger.PrintToUser("Use 'lux self install <version>' to install a version.")
		return nil
	}

	// List versions
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return fmt.Errorf("failed to read versions directory: %w", err)
	}

	if len(entries) == 0 {
		ux.Logger.PrintToUser("No versions installed yet.")
		ux.Logger.PrintToUser("Use 'lux self install <version>' to install a version.")
		return nil
	}

	ux.Logger.PrintToUser("Installed versions:")
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			marker := "  "
			if version == activeVersion {
				marker = "* "
			}
			ux.Logger.PrintToUser("%s%s", marker, version)
		}
	}

	if activeTarget != "" && activeVersion == "" {
		// Linked to development build
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Currently using development build:")
		ux.Logger.PrintToUser("  %s", activeTarget)
	}

	return nil
}
