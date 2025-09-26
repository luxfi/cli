//go:build nowarp
// +build nowarp

// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayer

import (
	"errors"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/models"
)

// Stub implementations when warp is not available

func GenerateProposerConfig(
	app *application.Lux,
	network models.Network,
	subnetName string,
	blockchainName string,
	fullname string,
	multisig *models.MultisigTxInfo,
) (string, error) {
	return "", errors.New("relayer functionality not available in this build")
}

// GetDefaultRelayerKeyInfo returns the default relayer key information
func GetDefaultRelayerKeyInfo(app *application.Lux, subnetName string) (string, string, string, error) {
	// Return empty values for now - this would typically read from sidecar
	return "", "", "", nil
}

// Add other stub functions as needed
