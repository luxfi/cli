// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"fmt"
	"os/exec"

	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	"github.com/onsi/gomega"
)

// SimulateMainnetDeploySOV simulates sovereign subnet deployment on mainnet
/* #nosec G204 */
func SimulateMainnetDeploySOV(subnetName string, chainID int, skipPrompt bool) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		SubnetCmd,
		"deploy",
		subnetName,
		"--mainnet",
		"--sovereign",
		"--" + constants.SkipUpdateFlag,
	}

	// Add chain ID if specified
	if chainID > 0 {
		cmdArgs = append(cmdArgs, "--chain-id", fmt.Sprintf("%d", chainID))
	}

	// Add skip prompt flag if specified
	if skipPrompt {
		cmdArgs = append(cmdArgs, "--yes")
	}

	// Execute command
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(outputStr)
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	return outputStr
}

// SimulateMultisigMainnetDeploySOV simulates multisig sovereign subnet deployment on mainnet
/* #nosec G204 */
func SimulateMultisigMainnetDeploySOV(
	subnetName string,
	controlKeys []string,
	subnetAuthKeys []string,
	txPath string,
	expectError bool,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		SubnetCmd,
		"deploy",
		subnetName,
		"--mainnet",
		"--sovereign",
		"--multisig",
		"--tx-path", txPath,
		"--" + constants.SkipUpdateFlag,
	}

	// Add control keys
	for _, key := range controlKeys {
		cmdArgs = append(cmdArgs, "--control-keys", key)
	}

	// Add subnet auth keys
	for _, key := range subnetAuthKeys {
		cmdArgs = append(cmdArgs, "--subnet-auth-keys", key)
	}

	// Execute command
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if expectError {
		gomega.Expect(err).ShouldNot(gomega.BeNil())
	} else {
		if err != nil {
			fmt.Println(cmd.String())
			fmt.Println(outputStr)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
	}

	return outputStr
}

// SimulateMultisigMainnetDeployNonSOV simulates multisig non-sovereign subnet deployment on mainnet
/* #nosec G204 */
func SimulateMultisigMainnetDeployNonSOV(
	subnetName string,
	controlKeys []string,
	subnetAuthKeys []string,
	txPath string,
	expectError bool,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		SubnetCmd,
		"deploy",
		subnetName,
		"--mainnet",
		"--multisig",
		"--tx-path", txPath,
		"--" + constants.SkipUpdateFlag,
	}

	// Add control keys
	for _, key := range controlKeys {
		cmdArgs = append(cmdArgs, "--control-keys", key)
	}

	// Add subnet auth keys
	for _, key := range subnetAuthKeys {
		cmdArgs = append(cmdArgs, "--subnet-auth-keys", key)
	}

	// Execute command
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if expectError {
		gomega.Expect(err).ShouldNot(gomega.BeNil())
	} else {
		if err != nil {
			fmt.Println(cmd.String())
			fmt.Println(outputStr)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
	}

	return outputStr
}

// TransactionCommit commits a transaction from a file
/* #nosec G204 */
func TransactionCommit(subnetName string, txPath string, expectError bool) string {
	// Build command args
	cmdArgs := []string{
		"transaction",
		"commit",
		subnetName,
		"--tx-path", txPath,
		"--" + constants.SkipUpdateFlag,
	}

	// Execute command
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if expectError {
		gomega.Expect(err).ShouldNot(gomega.BeNil())
	} else {
		if err != nil {
			fmt.Println(cmd.String())
			fmt.Println(outputStr)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
	}

	return outputStr
}

// TransactionSignWithLedger signs a transaction with a ledger
/* #nosec G204 */
func TransactionSignWithLedger(subnetName string, txPath string, expectError bool) string {
	// Build command args
	cmdArgs := []string{
		"transaction",
		"sign",
		subnetName,
		"--ledger",
		"--tx-path", txPath,
		"--" + constants.SkipUpdateFlag,
	}

	// Execute command
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if expectError {
		gomega.Expect(err).ShouldNot(gomega.BeNil())
	} else {
		if err != nil {
			fmt.Println(cmd.String())
			fmt.Println(outputStr)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
	}

	return outputStr
}
