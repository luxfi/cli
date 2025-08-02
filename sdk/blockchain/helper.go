// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"github.com/luxfi/node/v2/v2/vms/platformvm"

	"github.com/luxfi/cli/v2/sdk/network"
	"github.com/luxfi/cli/v2/sdk/utils"

	"github.com/luxfi/ids"
)

func GetSubnet(subnetID ids.ID, network network.Network) (platformvm.GetSubnetClientResponse, error) {
	api := network.Endpoint
	pClient := platformvm.NewClient(api)
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	return pClient.GetSubnet(ctx, subnetID)
}
