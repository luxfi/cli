// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package precompiles

import (
	_ "embed"

	"github.com/luxfi/cli/pkg/contract"
	"github.com/luxfi/ids"
)

func WarpPrecompileGetBlockchainID(
	rpcURL string,
) (ids.ID, error) {
	out, err := contract.CallToMethod(
		rpcURL,
		WarpPrecompile,
		"getBlockchainID()->(bytes32)",
	)
	if err != nil {
		return ids.Empty, err
	}
	return contract.GetSmartContractCallResult[[32]byte]("getBlockchainID", out)
}
