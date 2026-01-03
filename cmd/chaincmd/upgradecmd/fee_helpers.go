// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package upgradecmd

import (
	"math/big"

	"github.com/luxfi/evm/commontype"
)

const (
	// Gas limits for different configurations
	LowGasLimit    = int64(8000000)
	MediumGasLimit = int64(15000000)
	HighGasLimit   = int64(20000000)

	// Target gas for different configurations
	LowTargetGas    = int64(15000000)
	MediumTargetGas = int64(15000000)
	HighTargetGas   = int64(15000000)
)

// setLowGasConfig configures low throughput fee settings
func setLowGasConfig(feeConfig *commontype.FeeConfig, _ bool) {
	if feeConfig == nil {
		return
	}
	feeConfig.GasLimit = big.NewInt(LowGasLimit)
	feeConfig.TargetGas = big.NewInt(LowTargetGas)
	feeConfig.TargetBlockRate = 2
	feeConfig.MinBaseFee = big.NewInt(25000000000)
	feeConfig.BaseFeeChangeDenominator = big.NewInt(36)
	feeConfig.MinBlockGasCost = big.NewInt(0)
	feeConfig.MaxBlockGasCost = big.NewInt(1000000)
	feeConfig.BlockGasCostStep = big.NewInt(200000)
}

// setMediumGasConfig configures medium throughput fee settings
func setMediumGasConfig(feeConfig *commontype.FeeConfig, _ bool) {
	if feeConfig == nil {
		return
	}
	feeConfig.GasLimit = big.NewInt(MediumGasLimit)
	feeConfig.TargetGas = big.NewInt(MediumTargetGas)
	feeConfig.TargetBlockRate = 2
	feeConfig.MinBaseFee = big.NewInt(25000000000)
	feeConfig.BaseFeeChangeDenominator = big.NewInt(36)
	feeConfig.MinBlockGasCost = big.NewInt(0)
	feeConfig.MaxBlockGasCost = big.NewInt(1000000)
	feeConfig.BlockGasCostStep = big.NewInt(200000)
}

// setHighGasConfig configures high throughput fee settings
func setHighGasConfig(feeConfig *commontype.FeeConfig, _ bool) {
	if feeConfig == nil {
		return
	}
	feeConfig.GasLimit = big.NewInt(HighGasLimit)
	feeConfig.TargetGas = big.NewInt(HighTargetGas)
	feeConfig.TargetBlockRate = 2
	feeConfig.MinBaseFee = big.NewInt(25000000000)
	feeConfig.BaseFeeChangeDenominator = big.NewInt(36)
	feeConfig.MinBlockGasCost = big.NewInt(0)
	feeConfig.MaxBlockGasCost = big.NewInt(1000000)
	feeConfig.BlockGasCostStep = big.NewInt(200000)
}

// createFeeConfig creates a new fee configuration with the specified parameters
func createFeeConfig(
	gasLimit *big.Int,
	targetBlockRate uint64,
	minBaseFee *big.Int,
	targetGas *big.Int,
	baseFeeChangeDenominator *big.Int,
	minBlockGasCost *big.Int,
	maxBlockGasCost *big.Int,
	blockGasCostStep *big.Int,
) *commontype.FeeConfig {
	return &commontype.FeeConfig{
		GasLimit:                 gasLimit,
		TargetBlockRate:          targetBlockRate,
		MinBaseFee:               minBaseFee,
		TargetGas:                targetGas,
		BaseFeeChangeDenominator: baseFeeChangeDenominator,
		MinBlockGasCost:          minBlockGasCost,
		MaxBlockGasCost:          maxBlockGasCost,
		BlockGasCostStep:         blockGasCostStep,
	}
}
