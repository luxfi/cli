// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"

	"github.com/luxfi/sdk/fees"
)

const (
	defaultEvmAirdropAmount = "1000000000000000000000000"
	goBackMsg               = "Go back to previous step"
)

var (
	Difficulty = big.NewInt(0)

	// This is the current c-chain gas config
	StarterFeeConfig = fees.DefaultFeeConfig

	oneLux = new(big.Int).SetUint64(1000000000000000000)
)
