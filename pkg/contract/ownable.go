// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package contract

import "github.com/luxfi/crypto"

// GetContractOwner gets owner for https://docs.openzeppelin.com/contracts/2.x/api/ownership#Ownable-owner contracts
func GetContractOwner(
	rpcURL string,
	contractAddress crypto.Address,
) (crypto.Address, error) {
	out, err := CallToMethod(
		rpcURL,
		contractAddress,
		"owner()->(address)",
	)
	if err != nil {
		return crypto.Address{}, err
	}
	return GetSmartContractCallResult[crypto.Address]("owner", out)
}

func TransferOwnership(
	rpcURL string,
	contractAddress crypto.Address,
	ownerPrivateKey string,
	newOwner crypto.Address,
) error {
	_, _, err := TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		ownerPrivateKey,
		contractAddress,
		nil,
		"transfer ownership",
		nil,
		"transferOwnership(address)",
		newOwner,
	)
	return err
}
