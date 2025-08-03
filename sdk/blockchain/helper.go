// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"github.com/luxfi/node/vms/platformvm"

	"github.com/luxfi/cli/sdk/network"
	"github.com/luxfi/cli/sdk/utils"

	"github.com/luxfi/ids"
)

func GetSubnet(subnetID ids.ID, network network.Network) (platformvm.GetSubnetClientResponse, error) {
	api := network.Endpoint
	pClient := platformvm.NewClient(api)
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	return pClient.GetSubnet(ctx, subnetID)
}
