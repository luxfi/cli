// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vmcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/api/admin"
	"github.com/spf13/cobra"
)

var (
	reloadEndpoint string
	reloadTimeout  time.Duration
)

func newReloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload VMs on network nodes",
		Long: `Reload VMs on network nodes by calling admin.loadVMs.

This triggers the node to scan the plugins directory and load any new VMs.

Examples:
  lux vm reload
  lux vm reload --endpoint http://127.0.0.1:9630`,
		Args: cobra.NoArgs,
		RunE: runReload,
	}

	cmd.Flags().StringVarP(&reloadEndpoint, "endpoint", "e", constants.DefaultNodeRunURL,
		"Node endpoint to call admin.loadVMs on")
	cmd.Flags().DurationVarP(&reloadTimeout, "timeout", "t", 30*time.Second,
		"Timeout for the reload request")

	return cmd
}

func runReload(_ *cobra.Command, _ []string) error {
	ux.Logger.PrintToUser("Reloading VMs on %s...", reloadEndpoint)

	client := admin.NewClient(reloadEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), reloadTimeout)
	defer cancel()

	// Call admin.loadVMs
	newVMs, failedVMs, err := client.LoadVMs(ctx)
	if err != nil {
		return fmt.Errorf("failed to reload VMs: %w", err)
	}

	if len(newVMs) == 0 && len(failedVMs) == 0 {
		ux.Logger.PrintToUser("No new VMs loaded (all VMs already loaded).")
		return nil
	}

	if len(newVMs) > 0 {
		ux.Logger.PrintToUser("Successfully loaded VMs:")
		for vmID, aliases := range newVMs {
			if len(aliases) > 0 {
				ux.Logger.PrintToUser("  %s (aliases: %v)", vmID, aliases)
			} else {
				ux.Logger.PrintToUser("  %s", vmID)
			}
		}
	}

	if len(failedVMs) > 0 {
		ux.Logger.PrintToUser("Failed to load VMs:")
		for vmID, errMsg := range failedVMs {
			ux.Logger.PrintToUser("  %s: %s", vmID, errMsg)
		}
		return fmt.Errorf("%d VM(s) failed to load", len(failedVMs))
	}

	return nil
}
