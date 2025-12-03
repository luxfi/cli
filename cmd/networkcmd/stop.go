// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/local"
	"github.com/luxfi/netrunner/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	forceStop   bool
	noSaveState bool
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the running local network and preserve state",
		Long: `The network stop command shuts down your local, multi-node network.

All deployed Subnets shutdown gracefully and save their state. If you provide the
--snapshot-name flag, the network saves its state under this named snapshot. You can
reload this snapshot with network start --snapshot-name <snapshotName>. Otherwise, the
network saves to the default snapshot, overwriting any existing state. You can reload the
default snapshot with network start.

This command is idempotent: if the network is not running, it reports this status
instead of failing.`,

		RunE:         StopNetwork,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", constants.DefaultSnapshotName, "name of snapshot to use to save network state into")
	cmd.Flags().BoolVar(&forceStop, "force", false, "force stop without saving snapshot")
	cmd.Flags().BoolVar(&noSaveState, "no-save", false, "stop without saving network state (alias for --force)")
	return cmd
}

func StopNetwork(*cobra.Command, []string) error {
	// Check if the server process is running first (idempotent check)
	checker := binutils.NewProcessChecker()
	isRunning, err := checker.IsServerProcessRunning(app)
	if err != nil {
		app.Log.Debug("could not check server process", zap.Error(err))
	}

	if !isRunning {
		ux.Logger.PrintToUser("Network is not running.")
		return nil
	}

	// Save network state unless --force or --no-save is specified
	if !forceStop && !noSaveState {
		if err := saveNetwork(); err != nil {
			// If network wasn't bootstrapped, that's fine - no state to save
			if !isNotBootstrappedError(err) {
				return err
			}
		}
	} else {
		ux.Logger.PrintToUser("Stopping without saving state (--force/--no-save specified)")
	}

	if err := binutils.KillgRPCServerProcess(app); err != nil {
		app.Log.Warn("failed killing server process", zap.Error(err))
		// Don't return error - process might already be dead
		ux.Logger.PrintToUser("Warning: %v", err)
	} else {
		ux.Logger.PrintToUser("Server shutdown gracefully")
	}

	return nil
}

// isNotBootstrappedError checks if the error indicates the network wasn't bootstrapped
func isNotBootstrappedError(err error) bool {
	if err == nil {
		return false
	}
	return server.IsServerError(err, server.ErrNotBootstrapped)
}

func saveNetwork() error {
	cli, err := binutils.NewGRPCClient(binutils.WithAvoidRPCVersionCheck(true))
	if err != nil {
		return err
	}

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
			return err
		}
	}

	_, err = cli.SaveSnapshot(ctx, snapshotName)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("Network stopped successfully.")

	return nil
}
