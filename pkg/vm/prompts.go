// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/sdk/models"
)

// AllocationEntry represents an allocation entry in genesis
type AllocationEntry struct {
	Address string
	Balance string
}

// EVMGenesisParams contains parameters for Chain EVM genesis
type EVMGenesisParams struct {
	UseDefaults         bool
	Interop             bool
	UseWarp             bool
	UseExternalGasToken bool
	ChainID             uint64
	TokenSymbol         string
	Allocations         []AllocationEntry
}

// PromptVMType prompts the user to select a VM type
func PromptVMType(app *application.Lux, useEVM bool, useCustom bool, usePars bool, useSession bool) (models.VMType, error) {
	if useEVM {
		return models.EVM, nil
	}
	if useSession {
		return models.SessionVM, nil
	}
	if usePars {
		return models.ParsVM, nil
	}
	if useCustom {
		return models.CustomVM, nil
	}

	// Prompt user to select VM type
	vmOptions := []string{
		"Lux EVM - Standard EVM with precompiles",
		"Session VM - Post-quantum secure messaging",
		"Pars VM - Pars network (EVM + SessionVM)",
		"Custom - Use your own VM binary",
	}

	selected, err := app.Prompt.CaptureList("Which VM would you like to use?", vmOptions)
	if err != nil {
		return models.EVM, err
	}

	switch selected {
	case vmOptions[0]:
		return models.EVM, nil
	case vmOptions[1]:
		return models.SessionVM, nil
	case vmOptions[2]:
		return models.ParsVM, nil
	case vmOptions[3]:
		return models.CustomVM, nil
	default:
		return models.EVM, nil
	}
}

// PromptDefaults prompts the user for default configuration
func PromptDefaults(app *application.Lux, defaultsKind DefaultsKind, vmType models.VMType) (DefaultsKind, error) {
	return defaultsKind, nil
}

// PromptEVMVersion prompts for Chain EVM version
func PromptEVMVersion(app *application.Lux, vmType models.VMType, version string) (string, error) {
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

// PromptEVMGenesisParams prompts for Chain EVM genesis parameters
func PromptEVMGenesisParams(
	app *application.Lux,
	params EVMGenesisParams,
	vmType models.VMType,
	version string,
	chainID uint64,
	symbol string,
	interop bool,
) (*EVMGenesisParams, error) {
	return &EVMGenesisParams{
		UseDefaults: params.UseDefaults,
		Interop:     interop,
		ChainID:     chainID,
		TokenSymbol: symbol,
	}, nil
}
