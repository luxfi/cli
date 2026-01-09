// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"os/exec"

	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	"github.com/onsi/gomega"
)

const (
	ContractCMD = "contract"
)

/* #nosec G204 */
func DeployERC20Contract(network, key, symbol, supply, receiver, blockchain string) string {
	// Create config
	erc20Args := []string{
		ContractCMD,
		"deploy",
		"erc20",
		network,
		"--key",
		key,
		"--symbol",
		symbol,
		"--supply",
		supply,
		"--funded",
		receiver,
		"--" + constants.SkipUpdateFlag,
	}

	if blockchain != "--c-chain" {
		erc20Args = append(erc20Args, "--blockchain", blockchain)
	} else {
		erc20Args = append(erc20Args, "--c-chain")
	}

	cmd := exec.Command(CLIBinary, erc20Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}
