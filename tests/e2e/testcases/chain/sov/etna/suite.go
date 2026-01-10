// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chain contains chain E2E tests.
package chain

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Test constants.
const (
	CLIBinary         = "./bin/lux"
	keyName           = "ewoq"
	ewoqEVMAddress    = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	ewoqPChainAddress = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
)

func createEtnaEVMConfig(poa, pos bool) string {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	cmdArgs := []string{
		"blockchain",
		"create",
		utils.BlockchainName,
		"--evm",
		"--validator-manager-owner",
		ewoqEVMAddress,
		"--proxy-contract-owner",
		ewoqEVMAddress,
		"--production-defaults",
		"--evm-chain-id=99999",
		"--evm-token=TOK",
		"--warp=false",
		"--" + constants.SkipUpdateFlag,
	}
	if poa {
		cmdArgs = append(cmdArgs, "--proof-of-authority")
	} else if pos {
		cmdArgs = append(cmdArgs, "--proof-of-stake")
	}

	cmd := exec.Command(CLIBinary, cmdArgs...) //nolint:gosec // G204: Running our own CLI binary in tests
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err = utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// return binary versions for this conf
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())
	return mapping[utils.LatestLuxd2EVMKey]
}

func createEtnaEVMConfigWithoutProxyOwner(poa, pos bool) {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	cmdArgs := []string{
		"blockchain",
		"create",
		utils.BlockchainName,
		"--evm",
		"--validator-manager-owner",
		ewoqEVMAddress,
		"--production-defaults",
		"--evm-chain-id=99999",
		"--evm-token=TOK",
		"--warp=false",
		"--" + constants.SkipUpdateFlag,
	}
	if poa {
		cmdArgs = append(cmdArgs, "--proof-of-authority")
	} else if pos {
		cmdArgs = append(cmdArgs, "--proof-of-stake")
	}

	cmd := exec.Command(CLIBinary, cmdArgs...) //nolint:gosec // G204: Running our own CLI binary in tests
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err = utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

func createEtnaEVMConfigValidatorManagerFlagKeyname(poa, pos bool) {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	cmdArgs := []string{
		"blockchain",
		"create",
		utils.BlockchainName,
		"--evm",
		"--validator-manager-owner",
		ewoqEVMAddress,
		"--proxy-contract-owner",
		ewoqEVMAddress,
		"--production-defaults",
		"--evm-chain-id=99999",
		"--evm-token=TOK",
		"--warp=false",
		"--" + constants.SkipUpdateFlag,
	}
	if poa {
		cmdArgs = append(cmdArgs, "--proof-of-authority")
	} else if pos {
		cmdArgs = append(cmdArgs, "--proof-of-stake")
	}

	cmd := exec.Command(CLIBinary, cmdArgs...) //nolint:gosec // G204: Running our own CLI binary in tests
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err = utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

func createEtnaEVMConfigValidatorManagerFlagPChain(poa, pos bool) {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	cmdArgs := []string{
		"blockchain",
		"create",
		utils.BlockchainName,
		"--evm",
		"--validator-manager-owner",
		ewoqPChainAddress,
		"--proxy-contract-owner",
		ewoqPChainAddress,
		"--production-defaults",
		"--evm-chain-id=99999",
		"--evm-token=TOK",
		"--warp=false",
		"--" + constants.SkipUpdateFlag,
	}
	if poa {
		cmdArgs = append(cmdArgs, "--proof-of-authority")
	} else if pos {
		cmdArgs = append(cmdArgs, "--proof-of-stake")
	}

	cmd := exec.Command(CLIBinary, cmdArgs...) //nolint:gosec // G204: Running our own CLI binary in tests
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).ShouldNot(gomega.BeNil())
}

func destroyLocalNode() {
	_, err := os.Stat(utils.TestLocalNodeName)
	if os.IsNotExist(err) {
		return
	}
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"node",
		"local",
		"destroy",
		utils.TestLocalNodeName,
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

func deployEtnaChainEtnaFlag() {
	// Check config exists
	exists, err := utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy chain on etna devnet with local machine as bootstrap validator
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"blockchain",
		"deploy",
		utils.BlockchainName,
		"--local",
		"--num-bootstrap-validators=1",
		"--ewoq",
		"--change-owner-address",
		ewoqPChainAddress,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

func deployEtnaChainEtnaFlagConvertOnly() {
	// Check config exists
	exists, err := utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy chain on etna devnet with local machine as bootstrap validator
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"blockchain",
		"deploy",
		utils.BlockchainName,
		"--local",
		"--num-bootstrap-validators=1",
		"--convert-only",
		"--ewoq",
		"--change-owner-address",
		ewoqPChainAddress,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

func deployEtnaChainClusterFlagConvertOnly(clusterName string) {
	// Check config exists
	exists, err := utils.ChainConfigExists(utils.BlockchainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy chain on etna devnet with local machine as bootstrap validator
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"blockchain",
		"deploy",
		utils.BlockchainName,
		fmt.Sprintf("--cluster=%s", clusterName),
		"--convert-only",
		"--ewoq",
		"--change-owner-address",
		ewoqPChainAddress,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

func initValidatorManagerClusterFlag(
	chainName string,
	clusterName string,
) error {
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"contract",
		"initValidatorManager",
		chainName,
		"--cluster",
		clusterName,
		"--genesis-key",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
	return err
}

func initValidatorManagerEtnaFlag(
	chainName string,
) (string, error) {
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"contract",
		"initValidatorManager",
		chainName,
		"--local",
		"--genesis-key",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())
	return string(output), err
}

var luxdVersion string

var _ = ginkgo.Describe("[Etna Chain SOV]", func() {
	ginkgo.BeforeEach(func() {
		// key
		_ = utils.DeleteKey(keyName)
		output, err := commands.CreateKeyFromPath(keyName, utils.LocalKeyPath)
		if err != nil {
			fmt.Println(output)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		// chain config
		_ = utils.DeleteConfigs(utils.BlockchainName)
		destroyLocalNode()
	})

	ginkgo.AfterEach(func() {
		destroyLocalNode()
		commands.DeleteChainConfig(utils.BlockchainName)
		_ = utils.DeleteKey(keyName)
		commands.CleanNetwork()
	})

	ginkgo.It("Test Create Etna POA Chain Config With Key Name for Validator Manager Flag", func() {
		createEtnaEVMConfigValidatorManagerFlagKeyname(true, false)
	})

	ginkgo.It("Test Create Etna POA Chain Config Without Proxy Owner Flag", func() {
		createEtnaEVMConfigWithoutProxyOwner(true, false)
	})

	ginkgo.It("Create Etna POA Chain Config & Deploy the Chain To Etna Local Network On Local Machine", func() {
		createEtnaEVMConfig(true, false)
		deployEtnaChainEtnaFlag()
	})

	ginkgo.It("Create Etna POS Chain Config & Deploy the Chain To Etna Local Network On Local Machine", func() {
		createEtnaEVMConfig(false, true)
		deployEtnaChainEtnaFlag()
	})

	ginkgo.It("Start Local Node on Etna & Deploy the Chain To Etna Local Network using cluster flag", func() {
		luxdVersion = createEtnaEVMConfig(true, false)
		_ = commands.StartNetworkWithVersion(luxdVersion)
		_, err := commands.CreateLocalEtnaNode(luxdVersion, utils.TestLocalNodeName, 1)
		gomega.Expect(err).Should(gomega.BeNil())
		deployEtnaChainClusterFlagConvertOnly(utils.TestLocalNodeName)
		_, err = commands.TrackLocalEtnaChain(utils.TestLocalNodeName, utils.BlockchainName)
		gomega.Expect(err).Should(gomega.BeNil())
		err = initValidatorManagerClusterFlag(utils.BlockchainName, utils.TestLocalNodeName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("Mix and match network and cluster flags test 1", func() {
		luxdVersion = createEtnaEVMConfig(true, false)
		_ = commands.StartNetworkWithVersion(luxdVersion)
		_, err := commands.CreateLocalEtnaNode(luxdVersion, utils.TestLocalNodeName, 1)
		gomega.Expect(err).Should(gomega.BeNil())
		deployEtnaChainClusterFlagConvertOnly(utils.TestLocalNodeName)
		_, err = commands.TrackLocalEtnaChain(utils.TestLocalNodeName, utils.BlockchainName)
		gomega.Expect(err).Should(gomega.BeNil())
		_, err = initValidatorManagerEtnaFlag(utils.BlockchainName)
		gomega.Expect(err).Should(gomega.BeNil())
	})
	ginkgo.It("Mix and match network and cluster flags test 2", func() {
		createEtnaEVMConfig(true, false)
		deployEtnaChainEtnaFlagConvertOnly()
		_, err := commands.TrackLocalEtnaChain(utils.TestLocalNodeName, utils.BlockchainName)
		gomega.Expect(err).Should(gomega.BeNil())
		err = initValidatorManagerClusterFlag(utils.BlockchainName, utils.TestLocalNodeName)
		gomega.Expect(err).Should(gomega.BeNil())
	})
})

var _ = ginkgo.Describe("[Etna Chain SOV With Errors]", func() {
	ginkgo.BeforeEach(func() {
		// key
		_ = utils.DeleteKey(keyName)
		output, err := commands.CreateKeyFromPath(keyName, utils.LocalKeyPath)
		if err != nil {
			fmt.Println(output)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		// chain config
		_ = utils.DeleteConfigs(utils.BlockchainName)
		destroyLocalNode()
	})

	ginkgo.AfterEach(func() {
		err := utils.DeleteKey(keyName)
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CleanNetwork()
	})

	ginkgo.It("Test Create Etna POA Chain Config With P Chain Address for Validator Manager Flag", func() {
		createEtnaEVMConfigValidatorManagerFlagPChain(true, false)
	})
})
