// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/local"
	"github.com/luxfi/netrunner/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const networkTypeLocal = "local"

var (
	stopNetworkType string
	forceStop       bool
)

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
                      REQUIRED if multiple networks are running
  --force             Force stop without confirmation (use with caution)

SAFETY CHECKS:

  If multiple networks are running, you MUST specify which one to stop:
    lux network stop --network-type devnet
    lux network stop --network-type custom

  Stopping mainnet or testnet requires explicit --network-type flag.
  This prevents accidental disruption of production deployments.

EXAMPLES:

  # Stop the running network (when only one is running)
  lux network stop

  # Stop specific network type (required when multiple running)
  lux network stop --network-type devnet

  # Stop with named snapshot
  lux network stop --network-type custom --snapshot-name my-snapshot

  # Resume from snapshot later
  lux network start --devnet --snapshot-name my-snapshot

NOTES:

  - Snapshots preserve ALL network state including deployed chains
  - Chain configurations (in ~/.lux/chains/) are NOT affected
  - Use 'lux network clean' to wipe runtime data completely
  - Only the specified network type is stopped (others remain running)
  - Use 'lux dev stop' for the dev mode node (separate from network command)

SNAPSHOT vs CLEAN:

  lux network stop    - Saves state for resuming later
  lux network clean   - Deletes runtime data, preserves chain configs`,

		RunE:         StopNetwork,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", constants.DefaultSnapshotName, "name of snapshot to use to save network state into")
	cmd.Flags().StringVar(&stopNetworkType, "network-type", "", "network type to stop (mainnet, testnet, devnet, custom) - REQUIRED if multiple networks running")
	cmd.Flags().BoolVar(&forceStop, "force", false, "force stop without confirmation (use with caution for mainnet/testnet)")
	return cmd
}

func StopNetwork(*cobra.Command, []string) error {
	// Get all running networks
	runningNetworks := app.GetAllRunningNetworks()
	devRunning := isDevModeRunning()

	// If network type not specified, apply safety checks
	if stopNetworkType == "" {
		// If multiple networks are running, require explicit --network-type
		if len(runningNetworks) > 1 {
			ux.Logger.PrintToUser("Multiple networks are running: %s", strings.Join(runningNetworks, ", "))
			if devRunning {
				ux.Logger.PrintToUser("Dev mode is also running (use 'lux dev stop' to stop it)")
			}
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Please specify which network to stop:")
			for _, net := range runningNetworks {
				ux.Logger.PrintToUser("  lux network stop --network-type %s", net)
			}
			return fmt.Errorf("ambiguous: multiple networks running. Use --network-type to specify which one to stop")
		}

		// If dev mode + one network, warn but allow stopping the network
		if devRunning && len(runningNetworks) == 1 {
			ux.Logger.PrintToUser("Note: Dev mode is also running. Use 'lux dev stop' to stop it separately.")
		}

		// Auto-detect the single running network
		if len(runningNetworks) == 1 {
			stopNetworkType = runningNetworks[0]
		} else if len(runningNetworks) == 0 {
			// No network running
			if devRunning {
				return fmt.Errorf("no network running. Dev mode is running - use 'lux dev stop' to stop it")
			}
			ux.Logger.PrintToUser("No network is currently running.")
			return nil
		}
	}

	// Normalize "local" to "custom"
	if stopNetworkType == networkTypeLocal {
		stopNetworkType = networkTypeCustom
	}

	// Safety check for mainnet/testnet: require explicit flag or force
	if (stopNetworkType == "mainnet" || stopNetworkType == "testnet") && !forceStop {
		// Check if this is a production-like network that needs protection
		ux.Logger.PrintToUser("WARNING: You are about to stop %s network.", strings.ToUpper(stopNetworkType))
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("This could disrupt production services. Are you sure?")
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("To confirm, run:")
		ux.Logger.PrintToUser("  lux network stop --network-type %s --force", stopNetworkType)
		return fmt.Errorf("stopping %s requires --force flag for safety", stopNetworkType)
	}

	// Check if the specified network is actually running
	isRunning := false
	for _, net := range runningNetworks {
		if net == stopNetworkType {
			isRunning = true
			break
		}
	}
	if !isRunning {
		ux.Logger.PrintToUser("Network '%s' is not currently running.", stopNetworkType)
		if len(runningNetworks) > 0 {
			ux.Logger.PrintToUser("Running networks: %s", strings.Join(runningNetworks, ", "))
		}
		return nil
	}

	ux.Logger.PrintToUser("Stopping network: %s", stopNetworkType)

	err := saveNetworkForType(stopNetworkType)

	if killErr := binutils.KillgRPCServerProcessForNetwork(app, stopNetworkType); killErr != nil {
		app.Log.Warn("failed killing server process", zap.Error(killErr))
		ux.Logger.PrintToUser("Warning: failed to shutdown server gracefully: %v", killErr)
	} else {
		ux.Logger.PrintToUser("Server (%s) shutdown gracefully", stopNetworkType)
	}

	// Clear network-specific state when stopping
	if clearErr := app.ClearNetworkStateForType(stopNetworkType); clearErr != nil {
		app.Log.Warn("failed to clear network state", zap.Error(clearErr))
	}

	return err
}

// isDevModeRunning checks if a dev mode node is currently running
func isDevModeRunning() bool {
	pidFile := filepath.Join(os.Getenv("HOME"), constants.BaseDirName, constants.DevDir, "luxd.pid")
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return false
	}

	// Check if process exists using os.FindProcess
	// Note: On Unix, FindProcess always succeeds, but we verify by checking /proc
	// On Windows, we check if we can open the process
	return isProcessRunning(pid)
}

func saveNetworkForType(networkType string) error {
	cli, err := binutils.NewGRPCClient(binutils.WithAvoidRPCVersionCheck(true), binutils.WithNetworkType(networkType))
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

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
