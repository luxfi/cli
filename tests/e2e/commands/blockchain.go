// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"fmt"
	"os/exec"

	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/onsi/gomega"
)

const strTrue = "true"

// ConfigureBlockchain configures a blockchain with the given flags
func ConfigureBlockchain(blockchainName string, flags utils.TestFlags) (string, error) {
	// Convert flags to args
	args := []string{"blockchain", "configure", blockchainName}
	for flag, value := range flags {
		args = append(args, "--"+flag, fmt.Sprintf("%v", value))
	}

	// Run the command
	/* #nosec G204 */
	cmd := exec.Command(CLIBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DeployBlockchain deploys a blockchain with the given flags
func DeployBlockchain(blockchainName string, flags utils.TestFlags) (string, error) {
	// Convert flags to args
	args := []string{"blockchain", "deploy", blockchainName}
	for flag, value := range flags {
		strValue := fmt.Sprintf("%v", value)
		if flag == "local" && strValue == strTrue {
			args = append(args, "--local")
		} else if flag == "skip-warp-deploy" && strValue == strTrue {
			args = append(args, "--skip-warp-deploy")
		} else if flag == "skip-update-check" && strValue == strTrue {
			args = append(args, "--skip-update-check")
		} else {
			args = append(args, "--"+flag, strValue)
		}
	}

	// Run the command
	/* #nosec G204 */
	cmd := exec.Command(CLIBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// CreateBlockchain creates a blockchain with the given VM type and version
func CreateBlockchain(blockchainName string, vmType string, version string) (string, error) {
	args := []string{"blockchain", "create", blockchainName, "--vm", vmType}
	if version != "" {
		args = append(args, "--version", version)
	}

	/* #nosec G204 */
	cmd := exec.Command(CLIBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DeleteBlockchain deletes a blockchain
func DeleteBlockchain(blockchainName string) error {
	exists, err := utils.BlockchainConfigExists(blockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	if !exists {
		return nil
	}

	args := []string{"blockchain", "delete", blockchainName, "--force"}
	/* #nosec G204 */
	cmd := exec.Command(CLIBinary, args...)
	_, err = cmd.CombinedOutput()
	return err
}

// BlockchainStatus gets the status of a blockchain
func BlockchainStatus(blockchainName string) (string, error) {
	args := []string{"blockchain", "status", blockchainName}
	/* #nosec G204 */
	cmd := exec.Command(CLIBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
