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

	// DefaultFundedAddress is the default prefunded address for local development
	// This corresponds to the mainnet-ops key in ~/.lux/keys/mainnet-ops
	// Private keys should be loaded from key files, not hardcoded in source code
	DefaultFundedAddress = common.HexToAddress("0xe495301d73ea70fa87fb5e5f2b337f1656d22f3f")

	oneLux = new(big.Int).SetUint64(1000000000000000000)
)
