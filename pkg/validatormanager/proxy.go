// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package validatormanager

import (
	_ "embed"
	"math/big"

	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/evm"
	"github.com/luxfi/geth/core/types"

	"github.com/luxfi/crypto"
)

func SetupValidatorProxyImplementation(
	rpcURL string,
	proxyManagerPrivateKey string,
	validatorManager crypto.Address,
) (*types.Transaction, *types.Receipt, error) {
	return contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		proxyManagerPrivateKey,
		crypto.HexToAddress(ValidatorProxyAdminContractAddress),
		big.NewInt(0),
		"set validator proxy implementation",
		ErrorSignatureToError,
		"upgrade(address,address)",
		crypto.HexToAddress(ValidatorProxyContractAddress),
		validatorManager,
	)
}

func GetValidatorProxyImplementation(
	rpcURL string,
) (crypto.Address, error) {
	out, err := contract.CallToMethod(
		rpcURL,
		crypto.HexToAddress(ValidatorProxyAdminContractAddress),
		"getProxyImplementation(address)->(address)",
		crypto.HexToAddress(ValidatorProxyContractAddress),
	)
	if err != nil {
		return crypto.Address{}, err
	}
	return contract.GetSmartContractCallResult[crypto.Address]("getProxyImplementation", out)
}

func ValidatorProxyHasImplementationSet(
	rpcURL string,
) (bool, error) {
	validatorManagerAddress, err := GetValidatorProxyImplementation(rpcURL)
	if err != nil {
		return false, err
	}
	client, err := evm.GetClient(rpcURL)
	if err != nil {
		return false, err
	}
	return client.ContractAlreadyDeployed(
		validatorManagerAddress.Hex(),
	)
}

func GetSpecializedValidatorProxyImplementation(
	rpcURL string,
) (crypto.Address, error) {
	out, err := contract.CallToMethod(
		rpcURL,
		crypto.HexToAddress(SpecializationProxyAdminContractAddress),
		"getProxyImplementation(address)->(address)",
		crypto.HexToAddress(SpecializationProxyContractAddress),
	)
	if err != nil {
		return crypto.Address{}, err
	}
	return contract.GetSmartContractCallResult[crypto.Address]("getProxyImplementation", out)
}

func SetupSpecializationProxyImplementation(
	rpcURL string,
	proxyManagerPrivateKey string,
	validatorManager crypto.Address,
) (*types.Transaction, *types.Receipt, error) {
	return contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		proxyManagerPrivateKey,
		crypto.HexToAddress(SpecializationProxyAdminContractAddress),
		big.NewInt(0),
		"set specialization proxy implementation",
		ErrorSignatureToError,
		"upgrade(address,address)",
		crypto.HexToAddress(SpecializationProxyContractAddress),
		validatorManager,
	)
}
