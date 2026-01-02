// Copyright (C) 2025, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build windows

package networkcmd

import (
	"golang.org/x/sys/windows"
)

// STILL_ACTIVE is the exit code for a running process on Windows
const stillActive = 259

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// On Windows, we open the process handle to check if it exists
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	// Check if process has exited
	var exitCode uint32
	err = windows.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}

	// stillActive (259) means process is running
	return exitCode == stillActive
}
