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

// SimulateMainnetDeploySOV simulates sovereign chain deployment on mainnet
/* #nosec G204 */
func SimulateMainnetDeploySOV(chainName string, chainID int, skipPrompt bool) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		ChainCmd,
		"deploy",
		chainName,
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

// SimulateMultisigMainnetDeploySOV simulates multisig sovereign chain deployment on mainnet
/* #nosec G204 */
func SimulateMultisigMainnetDeploySOV(
	chainName string,
	controlKeys []string,
	chainAuthKeys []string,
	txPath string,
	expectError bool,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		ChainCmd,
		"deploy",
		chainName,
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

	// Add chain auth keys
	for _, key := range chainAuthKeys {
		cmdArgs = append(cmdArgs, "--chain-auth-keys", key)
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

// SimulateMultisigMainnetDeployNonSOV simulates multisig non-sovereign chain deployment on mainnet
/* #nosec G204 */
func SimulateMultisigMainnetDeployNonSOV(
	chainName string,
	controlKeys []string,
	chainAuthKeys []string,
	txPath string,
	expectError bool,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		ChainCmd,
		"deploy",
		chainName,
		"--mainnet",
		"--multisig",
		"--tx-path", txPath,
		"--" + constants.SkipUpdateFlag,
	}

	// Add control keys
	for _, key := range controlKeys {
		cmdArgs = append(cmdArgs, "--control-keys", key)
	}

	// Add chain auth keys
	for _, key := range chainAuthKeys {
		cmdArgs = append(cmdArgs, "--chain-auth-keys", key)
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
func TransactionCommit(chainName string, txPath string, expectError bool) string {
	// Build command args
	cmdArgs := []string{
		"transaction",
		"commit",
		chainName,
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
func TransactionSignWithLedger(chainName string, txPath string, expectError bool) string {
	// Build command args
	cmdArgs := []string{
		"transaction",
		"sign",
		chainName,
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
