// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"os"
	"strings"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/server"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var verbose bool

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Prints the status of the local network",
		Long: `The network status command prints whether or not a local Lux
network is running and some basic stats about the network.`,

		RunE:         networkStatus,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

func networkStatus(*cobra.Command, []string) error {
	ux.Logger.PrintToUser("Requesting network status...")

	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	ctx := binutils.GetAsyncContext()
	status, err := cli.Status(ctx)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			ux.Logger.PrintToUser("No local network running")
			return nil
		}
		return err
	}

	// Use adaptive layout for different screen sizes
	const maxWidth = 100
	separator := strings.Repeat("=", min(maxWidth, getTerminalWidth()))
	nodeSeparator := strings.Repeat("-", min(maxWidth/2, getTerminalWidth()/2))

	if status != nil && status.ClusterInfo != nil {
		ux.Logger.PrintToUser("Network is Up. Network information:")
		ux.Logger.PrintToUser(separator)
		ux.Logger.PrintToUser("Healthy: %t", status.ClusterInfo.Healthy)
		ux.Logger.PrintToUser("Custom VMs healthy: %t", status.ClusterInfo.CustomChainsHealthy)
		ux.Logger.PrintToUser("Number of nodes: %d", len(status.ClusterInfo.NodeNames))
		ux.Logger.PrintToUser("Number of custom VMs: %d", len(status.ClusterInfo.CustomChains))
		ux.Logger.PrintToUser("%s Node information %s", nodeSeparator, nodeSeparator)
		for n, nodeInfo := range status.ClusterInfo.NodeInfos {
			ux.Logger.PrintToUser("%s has ID %s and endpoint %s ", n, nodeInfo.Id, nodeInfo.Uri)
		}
		if len(status.ClusterInfo.CustomChains) > 0 {
			ux.Logger.PrintToUser("%s Custom VM information %s", nodeSeparator, nodeSeparator)
			for _, nodeInfo := range status.ClusterInfo.NodeInfos {
				for blockchainID := range status.ClusterInfo.CustomChains {
					ux.Logger.PrintToUser("Endpoint at %s for blockchain %q: %s/ext/bc/%s/rpc", nodeInfo.Name, blockchainID, nodeInfo.GetUri(), blockchainID)
				}
			}
		}
	} else {
		ux.Logger.PrintToUser("No local network running")
	}

	// Show verbose output if flag is set
	if verbose {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Verbose output:")
		ux.Logger.PrintToUser(status.String())
	}

	return nil
}

// getTerminalWidth returns the current terminal width, or a default if unable to determine
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // Default width
	}
	return width
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
