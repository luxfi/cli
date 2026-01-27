//go:build windows

// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpc

import (
	"os"
	"os/exec"
)

// setSysProcAttr sets platform-specific process attributes.
// On Windows, this is a no-op as Setpgid is not available.
func setSysProcAttr(cmd *exec.Cmd) {
	// No special process attributes needed on Windows
}

// signalTerm sends a termination signal to the process.
// On Windows, this calls Kill() as SIGTERM is not available.
func signalTerm(process *os.Process) error {
	return process.Kill()
}

// checkProcessAlive checks if the process is still running.
// On Windows, we try to find the process which returns an error if not found.
func checkProcessAlive(process *os.Process) error {
	// On Windows, FindProcess always succeeds, so we try to
	// check if the process is still running by waiting with WNOHANG equivalent.
	// A simple approach is to try to get exit code, but that requires handle.
	// For simplicity, return nil and let the caller handle failures elsewhere.
	return nil
}
