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

/* #nosec G204 */
func ImportUpgradeBytes(chainName, filepath string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"import",
		chainName,
		"--upgrade-filepath",
		filepath,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

/* #nosec G204 */
func UpgradeVMConfig(chainName string, targetVersion string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"vm",
		chainName,
		"--config",
		"--version",
		targetVersion,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

/* #nosec G204 */
func UpgradeCustomVM(chainName string, binaryPath string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"vm",
		chainName,
		"--config",
		"--binary",
		binaryPath,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

/* #nosec G204 */
func UpgradeVMPublic(chainName string, targetVersion string, pluginDir string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"vm",
		chainName,
		"--testnet",
		"--version",
		targetVersion,
		"--plugin-dir",
		pluginDir,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

/* #nosec G204 */
func UpgradeVMLocal(chainName string, targetVersion string) string {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"vm",
		chainName,
		"--local",
		"--version",
		targetVersion,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}

	gomega.Expect(err).Should(gomega.BeNil())
	return string(output)
}

/* #nosec G204 */
func UpgradeCustomVMLocal(chainName string, binaryPath string) string {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"vm",
		chainName,
		"--local",
		"--binary",
		binaryPath,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
	return string(output)
}

/* #nosec G204 */
func ApplyUpgradeLocal(chainName string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"apply",
		chainName,
		"--local",
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

/* #nosec G204 */
func ApplyUpgradeToPublicNode(chainName, luxChainConfDir string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		UpgradeCmd,
		"apply",
		chainName,
		"--testnet",
		"--node-chain-config-dir",
		luxChainConfDir,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}
