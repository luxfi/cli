// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayercmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
)

// saveRelayerConfig saves the relayer configuration to a file
func saveRelayerConfig(configPath string, config interface{}) error {
	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, constants.WriteReadReadPerms); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// deployRelayerProcess starts the relayer process with the given configuration
func deployRelayerProcess(configPath, logPath string) error {
	// Get relayer binary path
	relayerBin := filepath.Join(app.GetBaseDir(), "bin", "warp-relayer")

	// Check if relayer binary exists
	if _, err := os.Stat(relayerBin); os.IsNotExist(err) {
		return fmt.Errorf("relayer binary not found at %s", relayerBin)
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, constants.WriteReadReadPerms)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// Start relayer process
	cmd := exec.Command(relayerBin, "--config", configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start relayer: %w", err)
	}

	// Detach the process
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to detach relayer process: %w", err)
	}

	ux.Logger.PrintToUser("✅ Relayer deployed successfully (PID: %d)", cmd.Process.Pid)
	return nil
}
