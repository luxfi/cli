// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package utils

import (
	"fmt"
	"os"
	"os/exec"
)

// GenerateDockerHostIPs generates IP addresses for Docker hosts
func GenerateDockerHostIPs(nodeCount int) ([]string, error) {
	ips := make([]string, nodeCount)
	for i := 0; i < nodeCount; i++ {
		// Generate IPs in the 172.18.0.x range for Docker network
		ips[i] = fmt.Sprintf("172.18.0.%d", i+10)
	}
	return ips, nil
}

// GenerateDockerHostIDs generates unique IDs for Docker hosts
func GenerateDockerHostIDs(nodeCount int) ([]string, error) {
	ids := make([]string, nodeCount)
	for i := 0; i < nodeCount; i++ {
		ids[i] = fmt.Sprintf("node-%d", i+1)
	}
	return ids, nil
}

// SaveDockerComposeFile saves Docker Compose configuration to a file
func SaveDockerComposeFile(content []byte, path string) error {
	return os.WriteFile(path, content, 0o644)
}

// StartDockerCompose starts Docker Compose with the given configuration file
func StartDockerCompose(dockerComposeFile string) error {
	// Execute docker-compose up command
	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start docker-compose: %w, output: %s", err, string(output))
	}
	return nil
}
