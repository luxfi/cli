// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
	sdkutils "github.com/luxfi/sdk/utils"
)

// GetNetworkBalance returns the balance of an address on the P-chain
func GetNetworkBalance(address ids.ShortID, network models.Network) (uint64, error) {
	return sdkutils.GetNetworkBalance(address, network)
}
