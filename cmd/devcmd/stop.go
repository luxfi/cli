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

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop local dev node",
		Long: `Stop the running Lux dev node.

This gracefully terminates the luxd process started by 'lux dev start'.`,
		RunE:         stopDevNode,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	return cmd
}

func stopDevNode(*cobra.Command, []string) error {
	ux.Logger.PrintToUser("Stopping Lux dev node...")

	// Try to find PID file first
	pidFile := filepath.Join(os.Getenv("HOME"), ".lux", "dev", "luxd.pid")
	if pidData, err := os.ReadFile(pidFile); err == nil { //nolint:gosec // G304: Reading from app's data directory
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err == nil {
			process, err := os.FindProcess(pid)
			if err == nil {
				if err := process.Signal(os.Interrupt); err == nil {
					ux.Logger.PrintToUser("Sent interrupt signal to PID %d", pid)
					_ = os.Remove(pidFile)
					return nil
				}
			}
		}
	}

	// Fallback: use pkill (only for luxd, and only in dev context)
	cmd := exec.Command("pkill", "-f", "luxd.*--dev")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// pkill returns error if no process found - that's ok
		if strings.Contains(string(output), "no process found") || cmd.ProcessState.ExitCode() == 1 {
			ux.Logger.PrintToUser("No dev node running")
			return nil
		}
		return fmt.Errorf("failed to stop dev node: %w", err)
	}

	ux.Logger.PrintToUser("Dev node stopped")
	return nil
}
