// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/sdk/fees"
)

const (
	defaultEvmAirdropAmount = "1000000000000000000000000"
	goBackMsg               = "Go back to previous step"
)

var (
	Difficulty = big.NewInt(0)

	slowTarget   = big.NewInt(15_000_000)
	mediumTarget = big.NewInt(20_000_000)
	fastTarget   = big.NewInt(50_000_000)

	// This is the current c-chain gas config
	StarterFeeConfig = fees.DefaultFeeConfig

	PrefundedEwoqAddress = common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")
	PrefundedEwoqPrivate = "56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027"

	oneLux = new(big.Int).SetUint64(1000000000000000000)
)
