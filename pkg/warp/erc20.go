// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package warp

import (
	_ "embed"

	"github.com/luxfi/crypto"
	"github.com/luxfi/erc20-go/erc20"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
)

func GetTokenParams(endpoint string, tokenAddress crypto.Address) (string, string, uint8, error) {
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return "", "", 0, err
	}
	// Convert crypto.Address to geth common.Address
	gethAddr := common.BytesToAddress(tokenAddress.Bytes())
	token, err := erc20.NewGGToken(gethAddr, client)
	if err != nil {
		return "", "", 0, err
	}
	tokenName, err := token.Name(nil)
	if err != nil {
		return "", "", 0, err
	}
	tokenSymbol, err := token.Symbol(nil)
	if err != nil {
		return "", "", 0, err
	}
	tokenDecimals, err := token.Decimals(nil)
	if err != nil {
		return "", "", 0, err
	}
	return tokenSymbol, tokenName, tokenDecimals, nil
}

func GetTokenDecimals(endpoint string, tokenAddress crypto.Address) (uint8, error) {
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return 0, err
	}
	// Convert crypto.Address to geth common.Address
	gethAddr := common.BytesToAddress(tokenAddress.Bytes())
	token, err := erc20.NewGGToken(gethAddr, client)
	if err != nil {
		return 0, err
	}
	return token.Decimals(nil)
}
