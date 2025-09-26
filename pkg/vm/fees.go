// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"
	
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/params/extras"
)

func GetFeeConfig(config params.ChainConfig, app *application.Lux) (
	params.ChainConfig,
	statemachine.StateDirection,
	error,
) {
	const (
		useFast   = "High disk use   / High Throughput   5 mil   gas/s"
		useMedium = "Medium disk use / Medium Throughput 2 mil   gas/s"
		useSlow   = "Low disk use    / Low Throughput    1.5 mil gas/s (C-Chain's setting)"
		customFee = "Customize fee config"

		setGasLimit                 = "Set gas limit"
		setBlockRate                = "Set target block rate"
		setMinBaseFee               = "Set min base fee"
		setTargetGas                = "Set target gas"
		setBaseFeeChangeDenominator = "Set base fee change denominator"
		setMinBlockGas              = "Set min block gas cost"
		setMaxBlockGas              = "Set max block gas cost"
		setGasStep                  = "Set block gas cost step"
		
		// Default gas targets
		fastTarget   = uint64(5000000)
		mediumTarget = uint64(2000000) 
		slowTarget   = uint64(1500000)
	)

	feeConfigOptions := []string{useSlow, useMedium, useFast, customFee, goBackMsg}

	feeDefault, err := app.Prompt.CaptureList(
		"How would you like to set fees",
		feeConfigOptions,
	)
	if err != nil {
		return config, statemachine.Stop, err
	}

	// Get the extras config or create a new one
	extrasConfig := params.GetExtra(&config)
	if extrasConfig == nil {
		extrasConfig = &extras.ChainConfig{}
	}
	
	// Initialize FeeConfig if not present (check if GasLimit is nil as indicator)
	if extrasConfig.FeeConfig.GasLimit == nil {
		extrasConfig.FeeConfig = commontype.FeeConfig{
			GasLimit: big.NewInt(8000000),
		}
	}

	switch feeDefault {
	case useFast:
		extrasConfig.FeeConfig.TargetGas = new(big.Int).SetUint64(fastTarget)
		return *params.WithExtra(&config, extrasConfig), statemachine.Forward, nil
	case useMedium:
		extrasConfig.FeeConfig.TargetGas = new(big.Int).SetUint64(mediumTarget)
		return *params.WithExtra(&config, extrasConfig), statemachine.Forward, nil
	case useSlow:
		extrasConfig.FeeConfig.TargetGas = new(big.Int).SetUint64(slowTarget)
		return *params.WithExtra(&config, extrasConfig), statemachine.Forward, nil
	case goBackMsg:
		return config, statemachine.Backward, nil
	default:
		ux.Logger.PrintToUser("Customizing fee config")
	}

	gasLimit, err := app.Prompt.CapturePositiveBigInt(setGasLimit)
	if err != nil {
		return config, statemachine.Stop, err
	}

	blockRate, err := app.Prompt.CapturePositiveBigInt(setBlockRate)
	if err != nil {
		return config, statemachine.Stop, err
	}

	minBaseFee, err := app.Prompt.CapturePositiveBigInt(setMinBaseFee)
	if err != nil {
		return config, statemachine.Stop, err
	}

	targetGas, err := app.Prompt.CapturePositiveBigInt(setTargetGas)
	if err != nil {
		return config, statemachine.Stop, err
	}

	baseDenominator, err := app.Prompt.CapturePositiveBigInt(setBaseFeeChangeDenominator)
	if err != nil {
		return config, statemachine.Stop, err
	}

	minBlockGas, err := app.Prompt.CapturePositiveBigInt(setMinBlockGas)
	if err != nil {
		return config, statemachine.Stop, err
	}

	maxBlockGas, err := app.Prompt.CapturePositiveBigInt(setMaxBlockGas)
	if err != nil {
		return config, statemachine.Stop, err
	}

	gasStep, err := app.Prompt.CapturePositiveBigInt(setGasStep)
	if err != nil {
		return config, statemachine.Stop, err
	}

	// Set the custom fee configuration
	extrasConfig.FeeConfig = commontype.FeeConfig{
		GasLimit:                 gasLimit,
		TargetBlockRate:          blockRate.Uint64(),
		MinBaseFee:               minBaseFee,
		TargetGas:                targetGas,
		BaseFeeChangeDenominator: baseDenominator,
		MinBlockGasCost:          minBlockGas,
		MaxBlockGasCost:          maxBlockGas,
		BlockGasCostStep:         gasStep,
	}

	// Return the config with the updated extras
	return *params.WithExtra(&config, extrasConfig), statemachine.Forward, nil
}
