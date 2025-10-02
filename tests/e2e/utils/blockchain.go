// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ParseBlockchainIDFromOutput parses the blockchain ID from deployment output
func ParseBlockchainIDFromOutput(output string) (string, error) {
	// Look for blockchain ID pattern in output
	// Common patterns:
	// "Blockchain ID: <id>"
	// "BlockchainID: <id>"
	// "blockchain id: <id>"
	patterns := []string{
		`(?i)blockchain[\s-_]*id[:\s]+([a-zA-Z0-9]+)`,
		`(?i)blockchain[\s]+([2-9A-HJ-NP-Za-km-z]{50,60})`, // Base58 encoded ID
		`(?i)deployed.*blockchain[\s]+([2-9A-HJ-NP-Za-km-z]{50,60})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	// Try to find it in a different format
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "blockchain") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				id := strings.TrimSpace(parts[len(parts)-1])
				if len(id) > 20 && len(id) < 100 {
					return id, nil
				}
			}
		}
	}

	return "", fmt.Errorf("blockchain ID not found in output")
}

// GetLocalClusterNodesInfo retrieves information about local cluster nodes
func GetLocalClusterNodesInfo() (map[string]NodeInfo, error) {
	nodesInfo := make(map[string]NodeInfo)

	// Default local network configuration
	// Check for lux-cli run directory
	runDir := os.ExpandEnv("$HOME/.lux-cli/runs/network_current")
	if _, err := os.Stat(runDir); os.IsNotExist(err) {
		// Try luxd data directory
		runDir = os.ExpandEnv("$HOME/.luxd")
	}

	// Look for node directories
	entries, err := os.ReadDir(runDir)
	if err != nil {
		// Return default single node configuration if directory doesn't exist
		nodesInfo["node1"] = NodeInfo{
			ID:         "NodeID-111111111111111111116DBWJs",
			PluginDir:  filepath.Join(runDir, "plugins"),
			ConfigFile: filepath.Join(runDir, "config.json"),
			URI:        "http://127.0.0.1:9630",
			LogDir:     filepath.Join(runDir, "logs"),
		}
		return nodesInfo, nil
	}

	nodeCount := 0
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "node") {
			nodeCount++
			nodeName := entry.Name()
			nodeDir := filepath.Join(runDir, nodeName)

			// Get node ID from staking certificate if available
			nodeID := fmt.Sprintf("NodeID-%d", nodeCount)
			stakingCertPath := filepath.Join(nodeDir, "staking", "staker.crt")
			if _, err := os.Stat(stakingCertPath); err == nil {
				// For simplicity, just use a placeholder ID
				// In a real implementation, you'd parse the certificate
				nodeID = fmt.Sprintf("NodeID-%s", strings.ToUpper(nodeName))
			}

			// Calculate HTTP port (9630 + nodeNumber - 1)
			httpPort := 9630 + nodeCount - 1

			nodesInfo[nodeName] = NodeInfo{
				ID:         nodeID,
				PluginDir:  filepath.Join(nodeDir, "plugins"),
				ConfigFile: filepath.Join(nodeDir, "config.json"),
				URI:        fmt.Sprintf("http://127.0.0.1:%d", httpPort),
				LogDir:     filepath.Join(nodeDir, "logs"),
			}
		}
	}

	// If no nodes found, return default configuration
	if len(nodesInfo) == 0 {
		nodesInfo["node1"] = NodeInfo{
			ID:         "NodeID-111111111111111111116DBWJs",
			PluginDir:  filepath.Join(runDir, "plugins"),
			ConfigFile: filepath.Join(runDir, "config.json"),
			URI:        "http://127.0.0.1:9630",
			LogDir:     filepath.Join(runDir, "logs"),
		}
	}

	return nodesInfo, nil
}

// BlockchainConfigExists checks if a blockchain configuration exists
func BlockchainConfigExists(blockchainName string) (bool, error) {
	// Check if the blockchain config exists in the standard location
	configDir := os.ExpandEnv("$HOME/.lux-cli/blockchains")
	configPath := filepath.Join(configDir, blockchainName, "config.json")

	if _, err := os.Stat(configPath); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// GetBlockchainID returns the blockchain ID for a given blockchain name
func GetBlockchainID(blockchainName string) (string, error) {
	// This would typically read from a config file or make an API call
	// For now, return a placeholder
	return fmt.Sprintf("blockchain-%s-id", blockchainName), nil
}

// CleanupLogs removes log files for a specific blockchain from all nodes
func CleanupLogs(nodesInfo map[string]NodeInfo, blockchainID string) {
	for _, nodeInfo := range nodesInfo {
		// Remove blockchain-specific log file
		blockchainLogFile := filepath.Join(nodeInfo.LogDir, blockchainID+".log")
		os.Remove(blockchainLogFile)

		// Also try to clear main.log if it exists (for subnet configs)
		mainLogFile := filepath.Join(nodeInfo.LogDir, "main.log")
		if _, err := os.Stat(mainLogFile); err == nil {
			// Truncate the file instead of deleting it to avoid issues with open file handles
			os.Truncate(mainLogFile, 0)
		}
	}
}