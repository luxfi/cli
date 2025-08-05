//go:build nowarp
// +build nowarp

// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayer

import (
	"errors"
	"github.com/luxfi/cli/pkg/application"
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

// Add other stub functions as needed
