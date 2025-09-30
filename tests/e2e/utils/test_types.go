// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// TestFlags represents test-specific flags
type TestFlags struct {
	Verbose   bool
	Timeout   string
	SkipSetup bool
	Env       map[string]string
}

// GlobalFlags represents global command flags
type GlobalFlags struct {
	Network  string
	Config   string
	LogLevel string
}

// TestCommand executes a command with the given flags for testing
func TestCommand(cmd, subCmd string, args []string, globalFlags GlobalFlags, testFlags TestFlags) (string, error) {
	// Build the command
	cmdArgs := []string{subCmd}
	cmdArgs = append(cmdArgs, args...)

	// Add global flags
	if globalFlags.Network != "" {
		cmdArgs = append(cmdArgs, "--network", globalFlags.Network)
	}
	if globalFlags.Config != "" {
		cmdArgs = append(cmdArgs, "--config", globalFlags.Config)
	}
	if globalFlags.LogLevel != "" {
		cmdArgs = append(cmdArgs, "--log-level", globalFlags.LogLevel)
	}

	// Add test flags
	if testFlags.Verbose {
		cmdArgs = append(cmdArgs, "-v")
	}
	if testFlags.Timeout != "" {
		cmdArgs = append(cmdArgs, "--timeout", testFlags.Timeout)
	}

	// Build exec command
	execCmd := exec.Command(cmd, cmdArgs...)

	// Set environment variables
	if testFlags.Env != nil {
		for k, v := range testFlags.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Execute command
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()
	if err != nil {
		return "", fmt.Errorf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Test key constants
const (
	// LatestEVM2LuxdKey represents the latest EVM to Luxd compatibility key
	LatestEVM2LuxdKey = "v0.14.0"

	// LatestLuxd2EVMKey represents the latest Luxd to EVM compatibility key
	LatestLuxd2EVMKey = "v1.12.0"

	// EwoqEVMAddress is the EVM address for the Ewoq test key
	EwoqEVMAddress = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
)
