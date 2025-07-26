// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"fmt"
	"os/exec"

	"github.com/luxfi/geth/common"
)

// CreateSubnetEvmConfigWithChainID creates a subnet EVM configuration with specific chain ID
func CreateSubnetEvmConfigWithChainID(subnetName string, genesisPath string, chainID int) {
	cmd := exec.Command(
		CLIBinary,
		"subnet",
		"create",
		subnetName,
		"--evm",
		"--genesis",
		genesisPath,
		"--chain-id",
		fmt.Sprintf("%d", chainID),
		"--force",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error creating subnet config: %s\n", string(output))
		panic(err)
	}
}

// CreateSubnetEvmConfigWithAllocation creates a subnet EVM configuration with custom allocations
func CreateSubnetEvmConfigWithAllocation(
	subnetName string,
	genesisPath string,
	chainID int,
	allocations map[common.Address]string,
) {
	// First create the subnet
	CreateSubnetEvmConfigWithChainID(subnetName, genesisPath, chainID)
	
	// Then update the allocation using subnet configure
	for addr, balance := range allocations {
		cmd := exec.Command(
			CLIBinary,
			"subnet",
			"configure",
			subnetName,
			"--allocation",
			fmt.Sprintf("%s:%s", addr.Hex(), balance),
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error configuring allocation: %s\n", string(output))
			panic(err)
		}
	}
}

// StartNetwork starts the local network
func StartNetwork(snapshotName string) error {
	args := []string{"network", "start"}
	if snapshotName != "" {
		args = append(args, "--snapshot-name", snapshotName)
	}
	
	cmd := exec.Command(CLIBinary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error starting network: %s\n", string(output))
		return err
	}
	return nil
}

// StopNetwork stops the local network
func StopNetwork() error {
	cmd := exec.Command(CLIBinary, "network", "stop")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error stopping network: %s\n", string(output))
		return err
	}
	return nil
}