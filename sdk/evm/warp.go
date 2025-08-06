// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package evm

import (
	"fmt"

	subnetEvmWarp "github.com/luxfi/evm/precompile/contracts/warp"
	"github.com/luxfi/geth/core/types"
	warp "github.com/luxfi/node/vms/platformvm/warp"
)

// get all unsigned warp messages contained in [logs]
func GetWarpMessagesFromLogs(
	logs []*types.Log,
) []*warp.UnsignedMessage {
	messages := []*warp.UnsignedMessage{}
	for _, txLog := range logs {
		msg, err := subnetEvmWarp.UnpackSendWarpEventDataToMessage(txLog.Data)
		if err == nil {
			messages = append(messages, msg)
		}
	}
	return messages
}

// get first unsigned warp message contained in [logs]
func ExtractWarpMessageFromLogs(
	logs []*types.Log,
) (*warp.UnsignedMessage, error) {
	messages := GetWarpMessagesFromLogs(logs)
	if len(messages) == 0 {
		return nil, fmt.Errorf("no warp message is present in evm logs")
	}
	return messages[0], nil
}

// get first unsigned warp message contained in [receipt]
func ExtractWarpMessageFromReceipt(
	receipt *types.Receipt,
) (*warp.UnsignedMessage, error) {
	if receipt == nil {
		return nil, fmt.Errorf("empty receipt was given")
	}
	return ExtractWarpMessageFromLogs(receipt.Logs)
}
