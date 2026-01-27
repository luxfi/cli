//go:build !windows

// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpc

import (
	"os"
	"os/exec"
	"syscall"
)

// setSysProcAttr sets platform-specific process attributes.
// On Unix systems, this sets Setpgid to create a new process group.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// signalTerm sends SIGTERM to the process for graceful shutdown.
func signalTerm(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}

// checkProcessAlive checks if the process is still running.
func checkProcessAlive(process *os.Process) error {
	return process.Signal(syscall.Signal(0))
}
