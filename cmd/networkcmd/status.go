// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/server"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	verbose        bool
	statusMainnet  bool
	statusTestnet  bool
	statusDevnet   bool
	statusAll      bool
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Prints the status of the local network",
		Long: `The network status command prints whether or not a local Lux
network is running and some basic stats about the network.

By default, it auto-detects the running network. Use flags to specify a network:
  --mainnet   Check mainnet status (gRPC port 8369)
  --testnet   Check testnet status (gRPC port 8368)
  --devnet    Check devnet status (gRPC port 8370)
  --all       Check all networks`,

		RunE:         networkStatus,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show detailed cluster info including raw protobuf response")
	cmd.Flags().BoolVar(&statusMainnet, "mainnet", false, "check mainnet network status")
	cmd.Flags().BoolVar(&statusTestnet, "testnet", false, "check testnet network status")
	cmd.Flags().BoolVar(&statusDevnet, "devnet", false, "check devnet network status")
	cmd.Flags().BoolVar(&statusAll, "all", false, "check status of all networks")

	return cmd
}

func networkStatus(*cobra.Command, []string) error {
	// Count how many network-specific flags are set
	flagCount := 0
	if statusMainnet {
		flagCount++
	}
	if statusTestnet {
		flagCount++
	}
	if statusDevnet {
		flagCount++
	}

	// Check for conflicting flags (but --all can override)
	if flagCount > 1 && !statusAll {
		return fmt.Errorf("cannot use multiple network flags together (use --all to check all networks)")
	}

	// If --all is set, check all networks
	if statusAll {
		networks := []string{"mainnet", "testnet", "devnet", "custom"}
		anyRunning := false
		for _, netType := range networks {
			if err := checkNetworkStatus(netType); err == nil {
				anyRunning = true
			}
		}
		if !anyRunning {
			ux.Logger.PrintToUser("No networks are currently running")
		}
		return nil
	}

	// Determine network type to check
	var networkType string
	switch {
	case statusMainnet:
		networkType = "mainnet"
	case statusTestnet:
		networkType = "testnet"
	case statusDevnet:
		networkType = "devnet"
	default:
		// Auto-detect from running network state
		networkType = app.GetRunningNetworkType()
		if networkType == "" || networkType == "local" {
			networkType = "custom" // Default fallback ("local" is deprecated)
		}
	}

	return checkNetworkStatus(networkType)
}

// checkNetworkStatus checks the status of a specific network type
func checkNetworkStatus(networkType string) error {
	ux.Logger.PrintToUser("Checking %s network status...", networkType)

	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(networkType))
	if err != nil {
		ux.Logger.PrintToUser("%s: Not running (failed to connect)", networkType)
		return err
	}
	defer cli.Close()

	ctx := binutils.GetAsyncContext()
	status, err := cli.Status(ctx)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			ux.Logger.PrintToUser("%s: Not running", networkType)
			return err
		}
		ux.Logger.PrintToUser("%s: Error - %v", networkType, err)
		return err
	}

	// Use adaptive layout for different screen sizes
	const maxWidth = 100
	separator := strings.Repeat("=", min(maxWidth, getTerminalWidth()))
	nodeSeparator := strings.Repeat("-", min(maxWidth/2, getTerminalWidth()/2))

	if status != nil && status.ClusterInfo != nil {
		// Get port info from gRPC ports config
		grpcPorts := binutils.GetGRPCPorts(networkType)

		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("%s Network is Up (gRPC port: %d)", strings.ToUpper(networkType[:1])+networkType[1:], grpcPorts.Server)
		ux.Logger.PrintToUser("%s", separator)
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

		// Show verbose output if flag is set
		if verbose {
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Verbose output:")
			ux.Logger.PrintToUser("%s", status.String())
		}
	} else {
		ux.Logger.PrintToUser("%s: No network running", networkType)
		return fmt.Errorf("no %s network running", networkType)
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
