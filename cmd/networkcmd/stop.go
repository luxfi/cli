// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/local"
	"github.com/luxfi/netrunner/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var stopNetworkType string

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the running network and save a snapshot",
		Long: `The network stop command gracefully shuts down the running network and saves state.

SNAPSHOT BEHAVIOR:

  By default, the network saves its state to a snapshot when stopping. This includes:
  - Blockchain state (C-Chain, P-Chain, X-Chain, deployed chains)
  - Validator state
  - Database contents

  The snapshot allows you to resume exactly where you left off with:
    lux network start --<type> --snapshot-name <name>

OPTIONS:

  --snapshot-name     Name for the snapshot (default: default-snapshot)
  --network-type      Network to stop (mainnet/testnet/devnet/custom)
                      If not specified, auto-detects the running network

EXAMPLES:

  # Stop the running network (auto-detect type)
  lux network stop

  # Stop with named snapshot
  lux network stop --snapshot-name my-snapshot

  # Stop specific network type
  lux network stop --network-type devnet

  # Resume from snapshot later
  lux network start --devnet --snapshot-name my-snapshot

NOTES:

  - Snapshots preserve ALL network state including deployed chains
  - Chain configurations (in ~/.lux/chains/) are NOT affected
  - Use 'lux network clean' to wipe runtime data completely
  - Only the specified network type is stopped (others remain stopped)

SNAPSHOT vs CLEAN:

  lux network stop    - Saves state for resuming later
  lux network clean   - Deletes runtime data, preserves chain configs`,

		RunE:         StopNetwork,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", constants.DefaultSnapshotName, "name of snapshot to use to save network state into")
	cmd.Flags().StringVar(&stopNetworkType, "network-type", "", "network type to stop (mainnet, testnet, devnet, custom)")
	return cmd
}

func StopNetwork(*cobra.Command, []string) error {
	// Determine which network to stop
	networkType := stopNetworkType

	// If network type not specified via flag, try to determine from state
	if networkType == "" {
		// First check if there's a running network - prioritize custom over others
		// This ensures "lux network stop" without flags targets custom network by default
		for _, netType := range []string{"custom", "devnet", "testnet", "mainnet"} {
			state, err := app.LoadNetworkStateForType(netType)
			if err == nil && state != nil && state.Running {
				networkType = netType
				break
			}
		}
		// Fallback to custom if no running network found (not mainnet - user must explicitly specify)
		if networkType == "" {
			networkType = "custom"
		}
	}

	// Normalize "local" to "custom"
	if networkType == "local" {
		networkType = "custom"
	}

	ux.Logger.PrintToUser("Stopping network: %s", networkType)

	err := saveNetworkForType(networkType)

	if killErr := binutils.KillgRPCServerProcessForNetwork(app, networkType); killErr != nil {
		app.Log.Warn("failed killing server process", zap.Error(killErr))
		ux.Logger.PrintToUser("Warning: failed to shutdown server gracefully: %v", killErr)
	} else {
		ux.Logger.PrintToUser("Server (%s) shutdown gracefully", networkType)
	}

	// Clear network-specific state when stopping
	if clearErr := app.ClearNetworkStateForType(networkType); clearErr != nil {
		app.Log.Warn("failed to clear network state", zap.Error(clearErr))
	}

	return err
}

func saveNetwork() error {
	return saveNetworkForType("mainnet")
}

func saveNetworkForType(networkType string) error {
	cli, err := binutils.NewGRPCClient(binutils.WithAvoidRPCVersionCheck(true), binutils.WithNetworkType(networkType))
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := binutils.GetAsyncContext()

	_, err = cli.RemoveSnapshot(ctx, snapshotName)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			ux.Logger.PrintToUser("Network already stopped.")
			return nil
		}
		// it we try to stop a network with a new snapshot name, remove snapshot
		// will fail, so we cover here that expected case
		if !server.IsServerError(err, local.ErrSnapshotNotFound) {
			return fmt.Errorf("failed stop network with a snapshot: %w", err)
		}
	}

	_, err = cli.SaveSnapshot(ctx, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to stop network with a snapshot: %w", err)
	}
	ux.Logger.PrintToUser("Network stopped successfully.")

	return nil
}
