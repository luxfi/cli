// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// TestFlags represents test-specific flags as a map for flexible flag passing
type TestFlags map[string]interface{}

// GlobalFlags represents global command flags as a map for flexible flag passing
type GlobalFlags map[string]interface{}

// TestCommand executes a command with the given flags for testing
func TestCommand(cmd, subCmd string, args []string, globalFlags GlobalFlags, testFlags TestFlags) (string, error) {
	// Build the command
	cmdArgs := []string{subCmd}
	cmdArgs = append(cmdArgs, args...)

	// Add global flags
	for key, value := range globalFlags {
		flagName := "--" + key
		switch v := value.(type) {
		case bool:
			if v {
				cmdArgs = append(cmdArgs, flagName)
			}
		case string:
			if v != "" {
				cmdArgs = append(cmdArgs, flagName, v)
			}
		case int, int64, float64:
			cmdArgs = append(cmdArgs, flagName, fmt.Sprintf("%v", v))
		default:
			if v != nil {
				cmdArgs = append(cmdArgs, flagName, fmt.Sprintf("%v", v))
			}
		}
	}

	// Add test flags
	for key, value := range testFlags {
		// Skip env as it's handled separately
		if key == "env" {
			continue
		}

		flagName := "--" + key
		// Handle short flags
		if key == "verbose" || key == "v" {
			flagName = "-v"
		}

		switch v := value.(type) {
		case bool:
			if v {
				cmdArgs = append(cmdArgs, flagName)
			}
		case string:
			if v != "" {
				cmdArgs = append(cmdArgs, flagName, v)
			}
		case int, int64, float64:
			cmdArgs = append(cmdArgs, flagName, fmt.Sprintf("%v", v))
		default:
			if v != nil {
				cmdArgs = append(cmdArgs, flagName, fmt.Sprintf("%v", v))
			}
		}
	}

	// Build exec command
	execCmd := exec.Command(cmd, cmdArgs...)

	// Set environment variables
	if envMap, ok := testFlags["env"].(map[string]string); ok {
		for k, v := range envMap {
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
