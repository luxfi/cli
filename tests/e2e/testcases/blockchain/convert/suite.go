// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package convert

import (
	"fmt"
	"runtime"

	"github.com/luxfi/cli/cmd"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	chainName = "testChain"
)

const ewoqEVMAddress = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"

func checkConvertOnlyOutput(output string, generateNodeID bool) {
	gomega.Expect(output).Should(gomega.ContainSubstring("Converted blockchain successfully generated"))
	gomega.Expect(output).Should(gomega.ContainSubstring("Have the Lux node(s) track the blockchain"))
	gomega.Expect(output).Should(gomega.ContainSubstring("Call `lux contract initValidatorManager testChain`"))
	gomega.Expect(output).Should(gomega.ContainSubstring("Ensure that the P2P port is exposed and 'public-ip' config value is set"))
	gomega.Expect(output).Should(gomega.ContainSubstring("Chain is successfully converted to sovereign L1"))
	if generateNodeID {
		gomega.Expect(output).Should(gomega.ContainSubstring("Create the corresponding Lux node(s) with the provided Node ID and BLS Info"))
	} else {
		gomega.Expect(output).ShouldNot(gomega.ContainSubstring("Create the corresponding Lux node(s) with the provided Node ID and BLS Info"))
	}
}

var _ = ginkgo.Describe("[Blockchain Convert]", ginkgo.Ordered, func() {
	blockchainCmdArgs := []string{chainName}
	_ = ginkgo.BeforeEach(func() {
		testFlags := utils.TestFlags{
			"latest":            true,
			"evm":               true,
			"evm-token":         "TOK",
			"sovereign":         false,
			"warp":              false,
			"skip-update-check": true,
			"genesis":           utils.EVMGenesisPoaPath,
		}
		_, err := utils.TestCommand(cmd.BlockchainCmd, "create", blockchainCmdArgs, nil, testFlags)
		gomega.Expect(err).Should(gomega.BeNil())

		globalFlags := utils.GlobalFlags{
			"local":             true,
			"skip-update-check": true,
		}
		_, err = utils.TestCommand(cmd.BlockchainCmd, "deploy", blockchainCmdArgs, globalFlags, nil)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		// Cleanup test chain config
		commands.DeleteChainConfig(chainName)
	})
	globalFlags := utils.GlobalFlags{
		"skip-update-check":         true,
		"local":                     true,
		"verify-input":              false,
		"validator-manager-owner":   "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC",
		"validator-manager-address": "0x0FEEDC0DE0000000000000000000000000000000",
		"proof-of-authority":        true,
		"key":                       "ewoq",
	}
	ginkgo.It("HAPPY PATH: local convert default", func() {
		testFlags := utils.TestFlags{}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(output).Should(gomega.ContainSubstring("Chain is successfully converted to sovereign L1"))
		gomega.Expect(err).Should(gomega.BeNil())
		// verify that we have a local machine created that is now a bootstrap validator
		localClusterUris, err := utils.GetLocalClusterUris()
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(localClusterUris)).Should(gomega.Equal(1))
	})

	ginkgo.It("HAPPY PATH: local convert with luxd path set", func() {
		luxdPath := "tests/e2e/assets/mac/luxd"
		if runtime.GOOS == "linux" {
			luxdPath = "tests/e2e/assets/linux/luxd"
		}
		testFlags := utils.TestFlags{
			"luxd-path": luxdPath,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(output).Should(gomega.ContainSubstring(fmt.Sprintf("Luxd path: %s", luxdPath)))
		gomega.Expect(output).Should(gomega.ContainSubstring("Chain is successfully converted to sovereign L1"))
		gomega.Expect(err).Should(gomega.BeNil())
		// verify that we have a local machine created that is now a bootstrap validator
		localClusterUris, err := utils.GetLocalClusterUris()
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(localClusterUris)).Should(gomega.Equal(1))
	})

	ginkgo.It("HAPPY PATH: convert only", func() {
		testFlags := utils.TestFlags{
			"convert-only": true,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		checkConvertOnlyOutput(output, false)
		gomega.Expect(err).Should(gomega.BeNil())
		// verify that we have a local machine created that is now a bootstrap validator
		localClusterUris, err := utils.GetLocalClusterUris()
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(localClusterUris)).Should(gomega.Equal(1))
	})

	ginkgo.It("HAPPY PATH: generate node id ends in convert only", func() {
		testFlags := utils.TestFlags{
			"generate-node-id":         true,
			"num-bootstrap-validators": 1,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		checkConvertOnlyOutput(output, true)
		gomega.Expect(err).Should(gomega.BeNil())
		sc, err := utils.GetSideCar(blockchainCmdArgs[0])
		gomega.Expect(err).Should(gomega.BeNil())
		networkData, exists := sc.Networks["Local Network"]
		gomega.Expect(exists).Should(gomega.BeTrue(), "Expected 'Local Network' to exist in Networks map")
		numValidators := len(networkData.BootstrapValidators)
		gomega.Expect(numValidators).Should(gomega.BeEquivalentTo(1))
		gomega.Expect(networkData.BootstrapValidators[0].NodeID).ShouldNot(gomega.BeEmpty())
		gomega.Expect(networkData.BootstrapValidators[0].BLSProofOfPossession).ShouldNot(gomega.BeEmpty())
		gomega.Expect(networkData.BootstrapValidators[0].BLSPublicKey).ShouldNot(gomega.BeEmpty())
	})

	ginkgo.It("HAPPY PATH: local convert with bootstrap validator balance", func() {
		testFlags := utils.TestFlags{
			"balance": 0.2,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(output).Should(gomega.ContainSubstring("Chain is successfully converted to sovereign L1"))
		gomega.Expect(err).Should(gomega.BeNil())

		sc, err := utils.GetSideCar(blockchainCmdArgs[0])
		gomega.Expect(err).Should(gomega.BeNil())

		networkData, exists := sc.Networks["Local Network"]
		gomega.Expect(exists).Should(gomega.BeTrue(), "Expected 'Local Network' to exist in Networks map")
		testFlags = utils.TestFlags{
			"local":         true,
			"validation-id": networkData.BootstrapValidators[0].ValidationID,
		}
		output, err = utils.TestCommand(cmd.ValidatorCmd, "getBalance", nil, nil, testFlags)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(output).To(gomega.ContainSubstring("Validator Balance: 0.20000 LUX"))
	})

	ginkgo.It("HAPPY PATH: local convert with bootstrap filepath", func() {
		testFlags := utils.TestFlags{
			"bootstrap-filepath": utils.BootstrapValidatorPath2,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		checkConvertOnlyOutput(output, false)
		gomega.Expect(err).Should(gomega.BeNil())

		sc, err := utils.GetSideCar(blockchainCmdArgs[0])
		gomega.Expect(err).Should(gomega.BeNil())

		networkData, exists := sc.Networks["Local Network"]
		gomega.Expect(exists).Should(gomega.BeTrue(), "Expected 'Local Network' to exist in Networks map")
		for i := 0; i < 2; i++ {
			testFlags := utils.TestFlags{
				"local":         true,
				"validation-id": networkData.BootstrapValidators[i].ValidationID,
			}
			output, err = utils.TestCommand(cmd.ValidatorCmd, "getBalance", nil, nil, testFlags)
			gomega.Expect(err).Should(gomega.BeNil())
			if i == 0 {
				gomega.Expect(networkData.BootstrapValidators[i].NodeID).Should(gomega.Equal("NodeID-144PM69m93kSFyfTHMwULTmoGZSWzQ4C1"))
				gomega.Expect(networkData.BootstrapValidators[i].Weight).Should(gomega.BeEquivalentTo(20))
				gomega.Expect(output).To(gomega.ContainSubstring("Validator Balance: 0.20000 LUX"))
			} else {
				gomega.Expect(networkData.BootstrapValidators[i].NodeID).Should(gomega.Equal("NodeID-FtB74cdqNRrrsEpcyMHMvdpsRVodBupi3"))
				gomega.Expect(networkData.BootstrapValidators[i].Weight).Should(gomega.BeEquivalentTo(30))
				gomega.Expect(output).To(gomega.ContainSubstring("Validator Balance: 0.30000 LUX"))
			}
		}
	})

	ginkgo.It("HAPPY PATH: local convert with change owner address", func() {
		testFlags := utils.TestFlags{
			"change-owner-address": "P-custom1y5ku603lh583xs9v50p8kk0awcqzgeq0mezkqr",
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(output).Should(gomega.ContainSubstring("Chain is successfully converted to sovereign L1"))
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("HAPPY PATH: local convert set num bootstrap validators", func() {
		testFlags := utils.TestFlags{
			"num-bootstrap-validators": 2,
		}
		_, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.BeNil())

		sc, err := utils.GetSideCar(blockchainCmdArgs[0])
		gomega.Expect(err).Should(gomega.BeNil())
		networkData, exists := sc.Networks["Local Network"]
		gomega.Expect(exists).Should(gomega.BeTrue(), "Expected 'Local Network' to exist in Networks map")
		numValidators := len(networkData.BootstrapValidators)
		gomega.Expect(numValidators).Should(gomega.BeEquivalentTo(2))

		localClusterUris, err := utils.GetLocalClusterUris()
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(localClusterUris)).Should(gomega.Equal(2))
	})

	ginkgo.It("ERROR PATH: invalid_version", func() {
		testFlags := utils.TestFlags{
			"luxd-version": "invalid_version",
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("invalid version string"))
	})

	ginkgo.It("ERROR PATH: invalid_luxd_path", func() {
		luxdPath := "invalid_luxd_path"
		testFlags := utils.TestFlags{
			"luxd-path": luxdPath,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring(fmt.Sprintf("luxd binary %s does not exist", luxdPath)))
	})

	ginkgo.It("ERROR PATH: zero balance value", func() {
		testFlags := utils.TestFlags{
			"balance": 0,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("bootstrap validator balance must be greater than 0 LUX"))
	})

	ginkgo.It("ERROR PATH: negative balance value", func() {
		testFlags := utils.TestFlags{
			"balance": -1.0,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("bootstrap validator balance must be greater than 0 LUX"))
	})

	ginkgo.It("ERROR PATH: invalid bootstrap filepath", func() {
		fileName := "nonexistent.json"
		testFlags := utils.TestFlags{
			"bootstrap-filepath": fileName,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("file path \"%s\" doesn't exist", fileName))
	})

	ginkgo.It("ERROR PATH: invalid change owner address format", func() {
		testFlags := utils.TestFlags{
			"change-owner-address": ewoqEVMAddress,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("failure parsing change owner address: no separator found in address"))
	})

	ginkgo.It("ERROR PATH: generate node id is not applicable if convert only is false", func() {
		testFlags := utils.TestFlags{
			"generate-node-id": true,
			"convert-only":     false,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot set --convert-only=false if --generate-node-id=true"))
	})
	ginkgo.It("ERROR PATH: generate node id is not applicable if use local machine is true", func() {
		testFlags := utils.TestFlags{
			"generate-node-id":  true,
			"use-local-machine": true,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use local machine as bootstrap validator if --generate-node-id=true"))
	})

	ginkgo.It("ERROR PATH: bootstrap filepath is not applicable if convert only is false", func() {
		testFlags := utils.TestFlags{
			"bootstrap-filepath": utils.BootstrapValidatorPath,
			"convert-only":       false,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot set --convert-only=false if --bootstrap-filepath is not empty"))
	})
	ginkgo.It("ERROR PATH: bootstrap filepath is not applicable if use local machine is true", func() {
		testFlags := utils.TestFlags{
			"bootstrap-filepath": utils.BootstrapValidatorPath,
			"use-local-machine":  true,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use local machine as bootstrap validator if --bootstrap-filepath is not empty"))
	})
	ginkgo.It("ERROR PATH: bootstrap endpoints is not applicable if convert only is false", func() {
		testFlags := utils.TestFlags{
			"bootstrap-endpoints": "127.0.0.1:9630",
			"convert-only":        false,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot set --convert-only=false if --bootstrap-endpoints is not empty"))
	})
	ginkgo.It("ERROR PATH: bootstrap endpoints is not applicable if use local machine is true", func() {
		testFlags := utils.TestFlags{
			"bootstrap-endpoints": "127.0.0.1:9630",
			"use-local-machine":   true,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use local machine as bootstrap validator if --bootstrap-endpoints is not empty"))
	})
	ginkgo.It("ERROR PATH: bootstrap filepath cannot be set if generate node id is true", func() {
		testFlags := utils.TestFlags{
			"bootstrap-filepath": utils.BootstrapValidatorPath2,
			"generate-node-id":   true,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use --generate-node-id=true and a non-empty --bootstrap-filepath at the same time"))
	})
	ginkgo.It("ERROR PATH: bootstrap filepath cannot be set if num bootstrap validators is set", func() {
		testFlags := utils.TestFlags{
			"bootstrap-filepath":       utils.BootstrapValidatorPath2,
			"num-bootstrap-validators": 2,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use a non-empty --num-bootstrap-validators and a non-empty --bootstrap-filepath at the same time"))
	})
	ginkgo.It("ERROR PATH: bootstrap filepath cannot be set if balance is set", func() {
		testFlags := utils.TestFlags{
			"bootstrap-filepath": utils.BootstrapValidatorPath2,
			"balance":            0.2,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use a non-empty --balance and a non-empty --bootstrap-filepath at the same time"))
	})
	ginkgo.It("ERROR PATH: bootstrap filepath cannot be set if bootstrap endpoints is set", func() {
		testFlags := utils.TestFlags{
			"bootstrap-filepath":  utils.BootstrapValidatorPath2,
			"bootstrap-endpoints": "127.0.0.1:9630",
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use a non-empty --bootstrap-endpoints and a non-empty --bootstrap-filepath at the same time"))
	})
	ginkgo.It("ERROR PATH: bootstrap endpoints is not applicable if generate node id is true", func() {
		testFlags := utils.TestFlags{
			"bootstrap-endpoints": utils.BootstrapValidatorPath2,
			"generate-node-id":    true,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use --generate-node-id=true and a non-empty --bootstrap-endpoints at the same time"))
	})
	ginkgo.It("ERROR PATH: bootstrap endpoints is not applicable if num bootstrap validators is set", func() {
		testFlags := utils.TestFlags{
			"bootstrap-endpoints":      utils.BootstrapValidatorPath2,
			"num-bootstrap-validators": 2,
		}
		output, err := utils.TestCommand(cmd.BlockchainCmd, "convert", blockchainCmdArgs, globalFlags, testFlags)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(output).Should(gomega.ContainSubstring("cannot use a non-empty --num-bootstrap-validators and a non-empty --bootstrap-endpoints at the same time"))
	})
})
