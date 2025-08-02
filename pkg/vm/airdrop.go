// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"math/big"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/core/types"
	"github.com/luxfi/geth/common"
)

const (
	defaultAirdrop = "Airdrop 1 million tokens to the default address (do not use in production)"
	customAirdrop  = "Customize your airdrop"
	extendAirdrop  = "Would you like to airdrop more tokens?"
)

func getDefaultAllocation(defaultAirdropAmount string) (core.GenesisAlloc, error) {
	allocation := core.GenesisAlloc{}
	defaultAmount, ok := new(big.Int).SetString(defaultAirdropAmount, 10)
	if !ok {
		return allocation, errors.New("unable to decode default allocation")
	}

	allocation[PrefundedEwoqAddress] = types.GenesisAccount{
		Balance: defaultAmount,
	}
	return allocation, nil
}

func getAllocation(
	app *application.Lux,
	defaultAirdropAmount string,
	multiplier *big.Int,
	captureAmountLabel string,
) (core.GenesisAlloc, statemachine.StateDirection, error) {
	allocation := core.GenesisAlloc{}

	airdropType, err := app.Prompt.CaptureList(
		"How would you like to distribute funds",
		[]string{defaultAirdrop, customAirdrop, goBackMsg},
	)
	if err != nil {
		return allocation, statemachine.Stop, err
	}

	if airdropType == defaultAirdrop {
		alloc, err := getDefaultAllocation(defaultAirdropAmount)
		return alloc, statemachine.Forward, err
	}

	if airdropType == goBackMsg {
		return allocation, statemachine.Backward, nil
	}

	var addressHex common.Address

	for {
		addressHex, err = app.Prompt.CaptureAddress("Address to airdrop to")
		if err != nil {
			return nil, statemachine.Stop, err
		}

		amount, err := app.Prompt.CapturePositiveBigInt(captureAmountLabel)
		if err != nil {
			return nil, statemachine.Stop, err
		}

		amount = amount.Mul(amount, multiplier)

		account := types.GenesisAccount{
			Balance: amount,
		}

		allocation[addressHex] = account

		continueAirdrop, err := app.Prompt.CaptureNoYes(extendAirdrop)
		if err != nil {
			return nil, statemachine.Stop, err
		}
		if !continueAirdrop {
			return allocation, statemachine.Forward, nil
		}
	}
}
