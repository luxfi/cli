// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validatormanager

import (
	"github.com/luxfi/cli/pkg/contract"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/core/types"

	"github.com/luxfi/ids"
)

// PoAValidatorManagerInitialize initializes contract [managerAddress] at [rpcURL], to
// manage validators on [subnetID], with
// owner given by [ownerAddress]
func PoAValidatorManagerInitialize(
	rpcURL string,
	managerAddress crypto.Address,
	privateKey string,
	subnetID ids.ID,
	ownerAddress crypto.Address,
	useACP99 bool,
) (*types.Transaction, *types.Receipt, error) {
	const (
		defaultChurnPeriodSeconds     = uint64(0)
		defaultMaximumChurnPercentage = uint8(20)
	)
	if useACP99 {
		return contract.TxToMethod(
			rpcURL,
			false,
			crypto.Address{},
			privateKey,
			managerAddress,
			nil,
			"initialize PoA manager",
			ErrorSignatureToError,
			"initialize((address, bytes32,uint64,uint8))",
			ACP99ValidatorManagerSettings{
				Admin:                  ownerAddress,
				SubnetID:               subnetID,
				ChurnPeriodSeconds:     defaultChurnPeriodSeconds,
				MaximumChurnPercentage: defaultMaximumChurnPercentage,
			},
		)
	}
	return contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		privateKey,
		managerAddress,
		nil,
		"initialize PoA manager",
		ErrorSignatureToError,
		"initialize((bytes32,uint64,uint8),address)",
		ValidatorManagerSettings{
			SubnetID:               subnetID,
			ChurnPeriodSeconds:     defaultChurnPeriodSeconds,
			MaximumChurnPercentage: defaultMaximumChurnPercentage,
		},
		ownerAddress,
	)
}
