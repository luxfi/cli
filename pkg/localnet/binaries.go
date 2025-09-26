// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"fmt"
	"path/filepath"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/dependencies"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/utils"
)

// SetupLuxdBinary:
// * checks if luxd is installed in the local binary path
// * if not, it downloads and installs it (os - and archive dependent)
// * returns the location of the luxd path
func SetupLuxdBinary(
	app *application.Lux,
	luxdVersion string,
	luxdBinaryPath string,
) (string, error) {
	var err error
	if luxdBinaryPath == "" {
		if luxdVersion == constants.DefaultLuxdVersion {
			luxdVersion, err = dependencies.GetLatestCLISupportedDependencyVersion(app, constants.LuxdRepoName, models.NewLocalNetwork(), nil)
			if err != nil {
				return "", err
			}
		}
		luxdDir, err := binutils.SetupLuxgo(app, luxdVersion)
		if err != nil {
			return "", fmt.Errorf("failed setting up luxd binary: %w", err)
		}
		luxdBinaryPath = filepath.Join(luxdDir, "luxd")
	}
	if !utils.IsExecutable(luxdBinaryPath) {
		return "", fmt.Errorf("luxd binary %s does not exist", luxdBinaryPath)
	}
	return luxdBinaryPath, nil
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
