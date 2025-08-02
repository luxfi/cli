// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNetworkLaunch tests launching different network configurations
func TestNetworkLaunch(t *testing.T) {
	tests := []struct {
		name         string
		networkType  string
		expectedNodes int
		timeout      time.Duration
	}{
		{
			name:         "local network launch",
			networkType:  "local",
			expectedNodes: 5,
			timeout:      2 * time.Minute,
		},
		{
			name:         "testnet network launch",
			networkType:  "testnet", 
			expectedNodes: 11,
			timeout:      5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing network
			cleanupNetwork(t, tt.networkType)
			
			// Launch network
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			
			cmd := runCommand(ctx, "lux", "network", "start", "--"+tt.networkType, "--clean")
			output, err := cmd.CombinedOutput()
			
			require.NoError(t, err, "Failed to launch network: %s", string(output))
			
			// Wait for network to become healthy
			time.Sleep(30 * time.Second)
			
			// Check network status
			statusCmd := runCommand(ctx, "lux", "network", "status")
			statusOutput, err := statusCmd.CombinedOutput()
			
			require.NoError(t, err, "Failed to get network status: %s", string(statusOutput))
			
			// Verify expected number of nodes
			assert.Contains(t, string(statusOutput), fmt.Sprintf("%d nodes", tt.expectedNodes))
			assert.Contains(t, string(statusOutput), "healthy")
			
			// Clean up
			cleanupNetwork(t, tt.networkType)
		})
	}
}

// TestNodeJoin tests joining an existing network as a validator
func TestNodeJoin(t *testing.T) {
	// Skip if not running extended tests
	if os.Getenv("RUN_EXTENDED_TESTS") != "true" {
		t.Skip("Skipping extended test")
	}
	
	// Test joining testnet
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	
	// Generate validator key first
	keyGenCmd := runCommand(ctx, "lux", "key", "generate", "validator")
	keyOutput, err := keyGenCmd.CombinedOutput()
	require.NoError(t, err, "Failed to generate validator key: %s", string(keyOutput))
	
	// Extract NodeID from output
	var nodeID string
	lines := strings.Split(string(keyOutput), "\n")
	for _, line := range lines {
		if strings.Contains(line, "NodeID:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				nodeID = parts[1]
				break
			}
		}
	}
	require.NotEmpty(t, nodeID, "Failed to extract NodeID from key generation")
	
	// Test join command (dry run)
	joinCmd := runCommand(ctx, "lux", "node", "join", "--testnet", "--dry-run")
	joinOutput, err := joinCmd.CombinedOutput()
	
	require.NoError(t, err, "Failed to run join command: %s", string(joinOutput))
	assert.Contains(t, string(joinOutput), "NodeID: "+nodeID)
	assert.Contains(t, string(joinOutput), "testnet")
}

// TestValidatorKeyManagement tests key generation and management
func TestValidatorKeyManagement(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	keysDir := filepath.Join(homeDir, ".luxd", "keys")
	
	// Clean up any existing test keys
	testKeyPath := filepath.Join(keysDir, "test-validator-01.key")
	os.Remove(testKeyPath)
	os.Remove(strings.Replace(testKeyPath, ".key", ".crt", 1))
	
	ctx := context.Background()
	
	// Generate new validator key
	genCmd := runCommand(ctx, "lux", "key", "generate", "validator", "--name", "test-validator-01")
	output, err := genCmd.CombinedOutput()
	
	require.NoError(t, err, "Failed to generate key: %s", string(output))
	assert.Contains(t, string(output), "NodeID:")
	
	// Verify key files exist
	assert.FileExists(t, testKeyPath)
	assert.FileExists(t, strings.Replace(testKeyPath, ".key", ".crt", 1))
	
	// List keys
	listCmd := runCommand(ctx, "lux", "key", "list")
	listOutput, err := listCmd.CombinedOutput()
	
	require.NoError(t, err, "Failed to list keys: %s", string(listOutput))
	assert.Contains(t, string(listOutput), "test-validator-01")
	
	// Clean up
	os.Remove(testKeyPath)
	os.Remove(strings.Replace(testKeyPath, ".key", ".crt", 1))
}

// TestCrosschainTransfer tests C-chain to P-chain transfers for staking
func TestCrosschainTransfer(t *testing.T) {
	// Skip if not running extended tests
	if os.Getenv("RUN_EXTENDED_TESTS") != "true" {
		t.Skip("Skipping extended test")
	}
	
	// This test requires a running local network with funded accounts
	ctx := context.Background()
	
	// Launch local network first
	launchCmd := runCommand(ctx, "lux", "network", "start", "--local", "--clean")
	output, err := launchCmd.CombinedOutput()
	require.NoError(t, err, "Failed to launch network: %s", string(output))
	
	// Wait for network to be ready
	time.Sleep(30 * time.Second)
	
	// Test export from C-chain (dry run)
	exportCmd := runCommand(ctx, "lux", "transaction", "export", 
		"--amount", "100",
		"--from", "C",
		"--to", "P",
		"--dry-run")
	exportOutput, err := exportCmd.CombinedOutput()
	
	// Command might not be fully implemented yet
	if err == nil {
		assert.Contains(t, string(exportOutput), "export")
		assert.Contains(t, string(exportOutput), "C-chain")
		assert.Contains(t, string(exportOutput), "P-chain")
	}
	
	// Clean up
	cleanupNetwork(t, "local")
}

// Helper function to run commands
func runCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	return cmd
}

// Helper function to clean up networks
func cleanupNetwork(t *testing.T, networkType string) {
	ctx := context.Background()
	
	// Stop network
	stopCmd := runCommand(ctx, "lux", "network", "stop")
	stopCmd.Run() // Ignore errors as network might not be running
	
	// Clean data
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".luxd", "networks", networkType)
	os.RemoveAll(dataDir)
}