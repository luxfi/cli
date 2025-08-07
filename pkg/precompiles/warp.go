// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package precompiles

import (
	_ "embed"

	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
)

func WarpPrecompileGetBlockchainID(
	rpcURL string,
) (ids.ID, error) {
	// Convert geth common.Address to crypto.Address
	warpAddr := crypto.BytesToAddress(WarpPrecompile.Bytes())
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
