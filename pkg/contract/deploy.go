// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package contract

import (
	_ "embed"
	"math/big"

	"github.com/luxfi/crypto"
)

//go:embed contracts/bin/Token.bin
var tokenBin []byte

func DeployERC20(
	rpcURL string,
	privateKey string,
	symbol string,
	funded crypto.Address,
	supply *big.Int,
) (crypto.Address, error) {
	return DeployContract(
		rpcURL,
		privateKey,
		tokenBin,
		"(string, address, uint256)",
		symbol,
		funded,
		supply,
	)
}
