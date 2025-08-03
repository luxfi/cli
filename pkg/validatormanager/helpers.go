// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package validatormanager

import (
	"context"
	"math/big"

	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/sdk/evm"
	"github.com/luxfi/ids"
	warpMessage "github.com/luxfi/warp"
	warpPayload "github.com/luxfi/warp/payload"
	"github.com/luxfi/evm/interfaces"
	subnetEvmWarp "github.com/luxfi/evm/precompile/contracts/warp"

	"github.com/ethereum/go-ethereum/common"
)

func GetValidatorNonce(
	ctx context.Context,
	rpcURL string,
	validationID ids.ID,
) (uint64, error) {
	client, err := evm.GetClient(rpcURL)
	if err != nil {
		return 0, err
	}
	height, err := client.BlockNumber()
	if err != nil {
		return 0, err
	}
	count := uint64(0)
	maxBlock := int64(height)
	minBlock := int64(0)
	blockStep := int64(5000)
	for blockNumber := maxBlock; blockNumber >= minBlock; blockNumber -= blockStep {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		fromBlock := big.NewInt(blockNumber - blockStep)
		if fromBlock.Sign() < 0 {
			fromBlock = big.NewInt(0)
		}
		toBlock := big.NewInt(blockNumber)
		logs, err := client.FilterLogs(interfaces.FilterQuery{
			FromBlock: fromBlock,
			ToBlock:   toBlock,
			Addresses: []common.Address{subnetEvmWarp.Module.Address},
		})
		if err != nil {
			return 0, err
		}
		msgs := evm.GetWarpMessagesFromLogs(utils.PointersSlice(logs))
		for _, msg := range msgs {
			payload := msg.Payload
			addressedCall, err := warpPayload.ParseAddressedCall(payload)
			if err == nil {
				weightMsg, err := warpMessage.ParseL1ValidatorWeight(addressedCall.Payload)
				if err == nil {
					if weightMsg.ValidationID == validationID {
						count++
					}
				}
				regMsg, err := warpMessage.ParseRegisterL1Validator(addressedCall.Payload)
				if err == nil {
					if regMsg.ValidationID() == validationID {
						return count, nil
					}
				}
			}
		}
	}
	return count, nil
}
