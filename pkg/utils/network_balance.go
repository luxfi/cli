// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"fmt"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/sdk/models"
)

// GetNetworkBalance returns the balance of an address on the P-chain
func GetNetworkBalance(address ids.ShortID, network models.Network) (uint64, error) {
	pClient := platformvm.NewClient(network.Endpoint())
	ctx := context.Background()

	// Get the balance for the address
	response, err := pClient.GetBalance(ctx, []ids.ShortID{address})
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	// Check top-level unlocked first (for backward compatibility)
	if response.Unlocked > 0 {
		return uint64(response.Unlocked), nil
	}

	// If top-level unlocked is 0, sum all unlocked balances from the map
	// This handles custom networks where the LUX asset ID may not be set correctly
	var totalUnlocked uint64
	for _, balance := range response.Unlockeds {
		totalUnlocked += uint64(balance)
	}

	return totalUnlocked, nil
}
