// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"fmt"

	"github.com/luxfi/sdk/models"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/platformvm"
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
	
	// Return the unlocked balance
	return uint64(response.Unlocked), nil
}