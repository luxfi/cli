// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package precompiles

import (
	_ "embed"
	"math/big"

	"github.com/luxfi/crypto"
	"github.com/luxfi/sdk/contract"
)

func SetAdmin(
	rpcURL string,
	precompile crypto.Address,
	privateKey string,
	toSet crypto.Address,
) error {
	_, _, err := contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		privateKey,
		precompile,
		nil,
		"set precompile admin",
		nil,
		"setAdmin(address)",
		toSet,
	)
	return err
}

func SetManager(
	rpcURL string,
	precompile crypto.Address,
	privateKey string,
	toSet crypto.Address,
) error {
	_, _, err := contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		privateKey,
		precompile,
		nil,
		"set precompile manager",
		nil,
		"setManager(address)",
		toSet,
	)
	return err
}

func SetEnabled(
	rpcURL string,
	precompile crypto.Address,
	privateKey string,
	toSet crypto.Address,
) error {
	_, _, err := contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		privateKey,
		precompile,
		nil,
		"set precompile enabled",
		nil,
		"setEnabled(address)",
		toSet,
	)
	return err
}

func SetNone(
	rpcURL string,
	precompile crypto.Address,
	privateKey string,
	toSet crypto.Address,
) error {
	_, _, err := contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		privateKey,
		precompile,
		nil,
		"set precompile none",
		nil,
		"setNone(address)",
		toSet,
	)
	return err
}

func ReadAllowList(
	rpcURL string,
	precompile crypto.Address,
	toQuery crypto.Address,
) (*big.Int, error) {
	out, err := contract.CallToMethod(
		rpcURL,
		precompile,
		"readAllowList(address)->(uint256)",
		toQuery,
	)
	if err != nil {
		return nil, err
	}
	return contract.GetSmartContractCallResult[*big.Int]("readAllowList", out)
}
