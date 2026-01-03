// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
)

const (
	extendAirdrop = "Would you like to airdrop more tokens?"
)

// getAllocation prompts the user to specify addresses and amounts for the airdrop.
// There is no default option - users must always provide their own addresses.
func getAllocation(
	app *application.Lux,
	_ string,
	multiplier *big.Int,
	captureAmountLabel string,
) (types.GenesisAlloc, statemachine.StateDirection, error) {
	allocation := types.GenesisAlloc{}

	var addressHex crypto.Address
	var err error

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

		account := types.Account{
			Balance: amount,
		}

		// Convert crypto.Address to common.Address
		commonAddr := common.Address(addressHex)
		allocation[commonAddr] = account

		continueAirdrop, err := app.Prompt.CaptureNoYes(extendAirdrop)
		if err != nil {
			return nil, statemachine.Stop, err
		}
		if !continueAirdrop {
			return allocation, statemachine.Forward, nil
		}
	}
}
