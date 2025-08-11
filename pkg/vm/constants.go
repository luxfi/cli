// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Units
var OneLux = big.NewInt(1_000_000_000) // 1 LUX = 1e9 nLUX

// Gas limits
const (
	LowGasLimit    = uint64(12_000_000)
	MediumGasLimit = uint64(15_000_000)
	HighGasLimit   = uint64(20_000_000)
	
	LowTargetGas    = uint64(1_500_000)
	MediumTargetGas = uint64(2_000_000)
	HighTargetGas   = uint64(5_000_000)
	
	NoDynamicFeesGasLimitToTargetGasFactor = uint64(10)
)

// GetVMBinaryProtocolVersion returns the protocol version for the VM binary
func GetVMBinaryProtocolVersion(vmBinaryPath string) (int, error) {
	// Try to detect protocol version by running the VM binary with version flag
	if vmBinaryPath == "" {
		return 37, nil // Return default if no path provided
	}
	
	// Check if binary exists
	if _, err := os.Stat(vmBinaryPath); err != nil {
		return 37, fmt.Errorf("VM binary not found: %w", err)
	}
	
	// Try to run the binary with --version flag to get protocol info
	cmd := exec.Command(vmBinaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If version flag doesn't work, try other common flags
		cmd = exec.Command(vmBinaryPath, "-v")
		output, err = cmd.CombinedOutput()
		if err != nil {
			// Fall back to default if version detection fails
			return 37, nil
		}
	}
	
	// Parse output looking for protocol version
	// Expected format: "protocol-version: X" or "rpc-version: X"
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(line, "protocol") || strings.Contains(line, "rpc") {
			// Extract version number
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(part, "version") && i+1 < len(parts) {
					// Try to parse the next part as version number
					versionStr := strings.TrimPrefix(parts[i+1], "v")
					versionStr = strings.TrimSuffix(versionStr, ",")
					versionStr = strings.TrimSuffix(versionStr, ":")
					if version, err := strconv.Atoi(versionStr); err == nil {
						return version, nil
					}
				}
			}
		}
	}
	
	// Default to latest known RPC version if detection fails
	return 37, nil
}