// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package devcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// Deprecated. New code uses `lux down <brand>/<env>` (cmd/downcmd) which
// identifies the node by service (port + networkID handshake) rather
// than by PID file. This subcommand stays as the anvil-compat shortcut
// for `lux dev start` — it stops a node booted with that default
// dataDir (~/.lux/devnet) only.
func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the anvil-compat dev node started by `lux dev start`",
		Long: `Stops the dev node that `+"`lux dev start`"+` writes to its default
data-dir (~/.lux/devnet). For sovereign-L1 nodes booted via
`+"`lux up <brand>/<env>`"+`, use `+"`lux down <brand>/<env>`"+` instead.`,
		RunE:         stopDevNode,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	return cmd
}

func stopDevNode(*cobra.Command, []string) error {
	ux.Logger.PrintToUser("Stopping Lux dev node...")

	// dev start writes its PID into ~/.lux/devnet/luxd.pid.
	pidFile := filepath.Join(os.Getenv("HOME"), ".lux", "devnet", "luxd.pid")
	if pidData, err := os.ReadFile(pidFile); err == nil { //nolint:gosec
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				if err := process.Signal(os.Interrupt); err == nil {
					ux.Logger.PrintToUser("Sent interrupt signal to PID %d", pid)
					_ = os.Remove(pidFile)
					return nil
				}
			}
		}
	}

	// Fallback: identify dev profile by its K=1 quorum flag.
	cmd := exec.Command("pkill", "-f", "luxd.*--consensus-quorum-size=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "no process found") || cmd.ProcessState.ExitCode() == 1 {
			ux.Logger.PrintToUser("No dev node running")
			return nil
		}
		return fmt.Errorf("failed to stop dev node: %w", err)
	}
	ux.Logger.PrintToUser("Dev node stopped")
	return nil
}
