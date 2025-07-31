// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"fmt"
	"path/filepath"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/dependencies"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/utils"
)

// SetupLuxGoBinary:
// * checks if luxd is installed in the local binary path
// * if not, it downloads and installs it (os - and archive dependent)
// * returns the location of the luxd path
func SetupLuxGoBinary(
	app *application.Lux,
	luxGoVersion string,
	luxGoBinaryPath string,
) (string, error) {
	var err error
	if luxGoBinaryPath == "" {
		if luxGoVersion == constants.DefaultLuxGoVersion {
			luxGoVersion, err = dependencies.GetLatestCLISupportedDependencyVersion(app, constants.LuxGoRepoName, models.NewLocalNetwork(), nil)
			if err != nil {
				return "", err
			}
		}
		_, luxGoDir, err := binutils.SetupLuxgo(app, luxGoVersion)
		if err != nil {
			return "", fmt.Errorf("failed setting up luxd binary: %w", err)
		}
		luxGoBinaryPath = filepath.Join(luxGoDir, "luxd")
	}
	if !utils.IsExecutable(luxGoBinaryPath) {
		return "", fmt.Errorf("luxGo binary %s does not exist", luxGoBinaryPath)
	}
	return luxGoBinaryPath, nil
}

// SetupVMBinary ensures a binary for [blockchainName]'s VM is locally available,
// and provides a path to it
func SetupVMBinary(
	app *application.Lux,
	blockchainName string,
) (string, error) {
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return "", err
	}
	var binaryPath string
	switch sc.VM {
	case models.SubnetEvm:
		_, binaryPath, err = binutils.SetupSubnetEVM(app, sc.VMVersion)
		if err != nil {
			return "", fmt.Errorf("failed to install subnet-evm: %w", err)
		}
	case models.CustomVM:
		binaryPath = binutils.SetupCustomBin(app, blockchainName)
	default:
		return "", fmt.Errorf("unknown vm: %s", sc.VM)
	}
	return binaryPath, nil
}
