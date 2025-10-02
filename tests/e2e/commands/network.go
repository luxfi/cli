// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"fmt"
	"os/exec"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/onsi/gomega"
)

/* #nosec G204 */
func CleanNetwork() {
	cmd := exec.Command(
		CLIBinary,
		NetworkCmd,
		"clean",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

/* #nosec G204 */
func CleanNetworkHard() {
	cmd := exec.Command(
		CLIBinary,
		NetworkCmd,
		"clean",
		"--hard",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

/* #nosec G204 */
func StartNetwork() string {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())

	return StartNetworkWithVersion(mapping[utils.OnlyLuxKey])
}

/* #nosec G204 */
func StartNetworkWithVersion(version string) string {
	cmdArgs := []string{NetworkCmd, "start"}
	if version != "" {
		cmdArgs = append(
			cmdArgs,
			"--node-version",
			version,
			"--"+constants.SkipUpdateFlag,
		)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...)
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
func StopNetwork(args ...string) error {
	cmdArgs := []string{NetworkCmd, "stop"}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "--"+constants.SkipUpdateFlag)

	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return err
}

/* #nosec G204 */
func GetNetworkStatus() (string, error) {
	cmd := exec.Command(
		CLIBinary,
		NetworkCmd,
		"status",
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
