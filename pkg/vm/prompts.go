// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/sdk/models"
)

// SubnetEVMGenesisParams contains parameters for Subnet EVM genesis
type SubnetEVMGenesisParams struct {
	UseDefaults         bool
	Interop             bool
	UseWarp             bool
	UseExternalGasToken bool
}

// PromptVMType prompts the user to select a VM type
func PromptVMType(app *application.Lux, useSubnetEVM bool, useCustom bool) (models.VMType, error) {
	if useSubnetEVM {
		return models.EVM, nil
	}
	if useCustom {
		return models.CustomVM, nil
	}
	// Default to EVM for now
	return models.EVM, nil
}

// PromptDefaults prompts the user for default configuration
func PromptDefaults(app *application.Lux, defaultsKind DefaultsKind, vmType models.VMType) (DefaultsKind, error) {
	return defaultsKind, nil
}

// PromptSubnetEVMVersion prompts for Subnet EVM version
func PromptSubnetEVMVersion(app *application.Lux, vmType models.VMType, version string) (string, error) {
	if version != "" {
		return version, nil
	}
	// Return latest version
	return "latest", nil
}

// PromptTokenSymbol prompts for token symbol
func PromptTokenSymbol(app *application.Lux, state statemachine.StateType, token string) (string, error) {
	if token != "" {
		return token, nil
	}
	// Default token symbol
	return "TKN", nil
}

// PromptInterop prompts for interoperability configuration
func PromptInterop(app *application.Lux, vmType models.VMType, version string, chainID uint64, interop bool) (bool, error) {
	return interop, nil
}

// PromptSubnetEVMGenesisParams prompts for Subnet EVM genesis parameters
func PromptSubnetEVMGenesisParams(
	app *application.Lux,
	params SubnetEVMGenesisParams,
	vmType models.VMType,
	version string,
	chainID uint64,
	symbol string,
	interop bool,
) (*SubnetEVMGenesisParams, error) {
	return &SubnetEVMGenesisParams{
		UseDefaults: params.UseDefaults,
		Interop:     interop,
	}, nil
}
