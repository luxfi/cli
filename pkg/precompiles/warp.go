// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package precompiles

import (
	luxcommon "github.com/luxfi/crypto/common"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/contract"
)

func WarpPrecompileGetBlockchainID(
	rpcURL string,
) (ids.ID, error) {
	// Convert geth common.Address to luxcommon.Address
	warpAddr := luxcommon.BytesToAddress(WarpPrecompile.Bytes())
	out, err := contract.CallToMethod(
		rpcURL,
		warpAddr,
		"getBlockchainID()->(bytes32)",
	)
	if err != nil {
		return ids.Empty, err
	}
	return contract.GetSmartContractCallResult[[32]byte]("getBlockchainID", out)
}
