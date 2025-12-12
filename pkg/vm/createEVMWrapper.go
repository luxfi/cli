// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/warp"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/geth/common"
)

// CreateEVMGenesisWithParams creates EVM genesis with extended parameters
func CreateEVMGenesisWithParams(
	app *application.Lux,
	params SubnetEVMGenesisParams,
	warpInfo *warp.WarpInfo,
	addWarpRegistryToGenesis bool,
	proxyContractOwner string,
	rewardBasisPoints uint64,
	useACP99 bool,
) ([]byte, error) {
	// Get chain ID from params
	chainIDBig := new(big.Int).SetUint64(params.ChainID)

	// Create allocations with prefunded addresses
	allocations := make(core.GenesisAlloc)

	// Add allocation for any prefunded addresses from params
	for _, alloc := range params.Allocations {
		addr := common.HexToAddress(alloc.Address)
		balance, ok := new(big.Int).SetString(alloc.Balance, 10)
		if !ok {
			// Try hex format without 0x prefix
			balanceStr := strings.TrimPrefix(alloc.Balance, "0x")
			balance, ok = new(big.Int).SetString(balanceStr, 16)
			if !ok {
				balance = new(big.Int).SetUint64(0)
			}
		}
		allocations[addr] = core.GenesisAccount{
			Balance: balance,
		}
	}

	// Add default allocation using local key if no allocations provided
	if len(allocations) == 0 {
		// Use local key for default allocation (generated per machine, or from env vars)
		const localNetworkID = 12345
		localKey, err := key.GetOrCreateLocalKey(localNetworkID)
		if err == nil {
			localAddr := common.HexToAddress(localKey.C())
			balance := new(big.Int)
			balance.SetString("1000000000000000000000000000", 10) // 1 billion with 18 decimals
			allocations[localAddr] = core.GenesisAccount{
				Balance: balance,
			}
		}
	}

	// Add warp prefunded address if warpInfo is available
	if warpInfo != nil && warpInfo.FundedAddress != "" {
		warpAddr := common.HexToAddress(warpInfo.FundedAddress)
		if warpInfo.FundedBalance != nil {
			allocations[warpAddr] = core.GenesisAccount{
				Balance: warpInfo.FundedBalance,
			}
		}
	}

	timestamps := make(map[string]uint64)

	// Create genesis map
	genesisMap := CreateEVMGenesis(chainIDBig, allocations, timestamps)

	// Serialize to JSON
	genesisBytes, err := json.MarshalIndent(genesisMap, "", "    ")
	if err != nil {
		return nil, err
	}

	return genesisBytes, nil
}
