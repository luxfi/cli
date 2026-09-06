// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package devcmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// Deprecated. New code uses `lux down <brand>/<env>` (cmd/downcmd) which
// identifies the node by service (port + networkID handshake) rather
// than by PID file. This subcommand stays as the anvil-compat shortcut
// for `lux dev start`.
func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the anvil-compat dev node started by `lux dev start`",
		Long: `Stops the node whose PID ` + "`lux dev start`" + ` recorded in its data-dir.
Pass the same --data-dir you started with. For sovereign-L1 nodes booted
via ` + "`lux up <brand>/<env>`" + `, use ` + "`lux down <brand>/<env>`" + ` instead.`,
		RunE:         stopDevNode,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "luxd data-dir the node was started with (default ~/.lux/devnet)")
	return cmd
}

func stopDevNode(*cobra.Command, []string) error {
	ux.Logger.PrintToUser("Stopping Lux dev node...")

	// The node to stop is the one this data-dir names. Matching on a command
	// line instead reaches nodes this command never started: the pattern it
	// used, a quorum size of one, fits every dev node on the machine. So the
	// PID file is the only address, and its absence is an answer rather than
	// a reason to guess.
	pidFile := devPIDFile()
	pidData, err := os.ReadFile(pidFile) //nolint:gosec // the caller's own data-dir
	if err != nil {
		ux.Logger.PrintToUser("No dev node recorded at %s", pidFile)
		return nil
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return fmt.Errorf("unreadable PID in %s: %w", pidFile, err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("no process %d from %s: %w", pid, pidFile, err)
	}
	if err := process.Signal(os.Interrupt); err != nil {
		_ = os.Remove(pidFile)
		ux.Logger.PrintToUser("PID %d from %s is already gone", pid, pidFile)
		return nil
	}
	_ = os.Remove(pidFile)
	ux.Logger.PrintToUser("Sent interrupt to PID %d", pid)
	return nil
}
