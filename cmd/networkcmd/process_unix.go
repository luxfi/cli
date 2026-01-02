// Copyright (C) 2025, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build !windows

package networkcmd

import "syscall"

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// On Unix, sending signal 0 checks if we can signal the process
	// without actually sending a signal
	err := syscall.Kill(pid, 0)
	return err == nil
}
