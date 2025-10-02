// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
)

// StartLedgerSim simulates ledger interaction for testing
// Returns two channels: interactionEndCh and ledgerSimEndCh
func StartLedgerSim(interactionCount int, seed string, verbose bool) (chan struct{}, chan struct{}) {
	interactionEndCh := make(chan struct{})
	ledgerSimEndCh := make(chan struct{})

	// Start a goroutine to simulate ledger interaction
	go func() {
		defer close(ledgerSimEndCh)

		if verbose {
			fmt.Printf("Starting ledger simulation with seed %s for %d interactions\n", seed, interactionCount)
		}

		// Wait for interaction signal or timeout
		select {
		case <-interactionEndCh:
			if verbose {
				fmt.Println("Ledger simulation completed")
			}
		case <-time.After(30 * time.Second):
			if verbose {
				fmt.Println("Ledger simulation timed out")
			}
		}
	}()

	return interactionEndCh, ledgerSimEndCh
}

// Note: FundLedgerAddress is implemented in helpers.go

// GetLedgerAddress retrieves the address from a ledger device
func GetLedgerAddress(network models.Network, index uint32) (string, error) {
	// This is a stub implementation
	// In real implementation, this would:
	// 1. Connect to the ledger device
	// 2. Derive the address at the given index
	// 3. Return the formatted address for the given network

	// Return a test address for now
	testAddresses := map[uint32]string{
		0: "P-custom1wu9sae0z2s80lv2x5gt5ys57y5yasqtnt6n2hs",
		1: "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p",
		2: "P-custom18g2tekxzt60j3sn8ymjx6qvk96xunhctkyzckt",
	}

	if addr, ok := testAddresses[index]; ok {
		return addr, nil
	}

	return fmt.Sprintf("P-custom1test%d", index), nil
}

// GetSubnetValidators returns the validators for a subnet
func GetSubnetValidators(subnetID ids.ID) ([]string, error) {
	// This is a stub implementation
	// In real implementation, this would query the P-Chain
	// to get the current validator set for the subnet

	// Return test validators for now
	validators := []string{
		"NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
		"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
		"NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
		"NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
		"NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
	}

	return validators, nil
}

// GetTmpFilePath creates and returns a temporary file path with the given prefix
func GetTmpFilePath(prefix string) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", prefix+"*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Close the file but keep the path
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	return tmpPath, nil
}

// GetSubnetEVMMainneChainID retrieves the mainnet chain ID for a SubnetEVM
func GetSubnetEVMMainneChainID(subnetName string) (uint, error) {
	// This is a stub implementation
	// In real implementation, this would:
	// 1. Read the subnet configuration
	// 2. Extract the mainnet chain ID from the genesis or sidecar

	// For testing, return 0 initially (not configured)
	// This allows tests to verify that chain ID can be set
	return 0, nil
}
