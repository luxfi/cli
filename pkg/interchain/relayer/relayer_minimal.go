// Copyright (C) 2025, Lux Industries Inc. All rights reserved
// See the file LICENSE for licensing terms.
package relayer

import (
	"errors"
	"os/exec"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/models"
)

// Minimal stub implementation until warp packages are available

func GenerateProposerConfig(
	app *application.Lux,
	network models.Network,
	subnetName string,
	blockchainName string,
	fullname string,
	multisig *models.MultisigTxInfo,
) (string, error) {
	return "", errors.New("relayer functionality temporarily disabled")
}

func DeployRelayer(
	binDir string,
	configPath string,
	logLevel string,
	logDisplayLevel string,
	networkID uint32,
	metricsPort uint16,
) *exec.Cmd {
	return nil
}

func DeployProposer(
	binDir string,
	configPath string,
	logLevel string,
	storageLocation string,
	disableSignatureAggregator bool,
	offchainRegistryAPIEndpoint string,
) *exec.Cmd {
	return nil
}
