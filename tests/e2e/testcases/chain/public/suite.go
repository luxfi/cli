// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chain contains chain E2E tests.
package chain

import (
	"fmt"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	chainName   = "e2eChainTest"
	controlKeys = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
	keyName     = "ewoq"
)

func deployChainToTestnet() (string, map[string]utils.NodeInfo) {
	// deploy
	s := commands.SimulateTestnetDeploy(chainName, keyName, controlKeys)
	chainID, err := utils.ParsePublicDeployOutput(s, utils.ChainIDParseType)
	gomega.Expect(err).Should(gomega.BeNil())
	// add validators to chain
	nodeInfos, err := utils.GetNodesInfo()
	gomega.Expect(err).Should(gomega.BeNil())
	for _, nodeInfo := range nodeInfos {
		start := time.Now().Add(time.Second * 30).UTC().Format("2006-01-02 15:04:05")
		_ = commands.SimulateTestnetAddValidator(chainName, keyName, nodeInfo.ID, start, "24h", "20")
	}
	// join to copy vm binary and update config file
	for _, nodeInfo := range nodeInfos {
		_ = commands.SimulateTestnetJoin(chainName, nodeInfo.ConfigFile, nodeInfo.PluginDir, nodeInfo.ID)
	}
	// get and check whitelisted chains from config file
	var whitelistedChains string
	for _, nodeInfo := range nodeInfos {
		whitelistedChains, err = utils.GetWhitelistedChainsFromConfigFile(nodeInfo.ConfigFile)
		gomega.Expect(err).Should(gomega.BeNil())
		whitelistedChainsSlice := strings.Split(whitelistedChains, ",")
		gomega.Expect(whitelistedChainsSlice).Should(gomega.ContainElement(chainID))
	}
	// update nodes whitelisted chains
	err = utils.RestartNodesWithWhitelistedChains(whitelistedChains)
	gomega.Expect(err).Should(gomega.BeNil())
	// wait for chain walidators to be up
	err = utils.WaitChainValidators(chainID, nodeInfos)
	gomega.Expect(err).Should(gomega.BeNil())
	return chainID, nodeInfos
}

var _ = ginkgo.Describe("[Public Chain]", func() {
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
		_ = utils.DeleteConfigs(chainName)
		_, luxVersion := commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)

		// local network
		commands.StartNetworkWithVersion(luxVersion)
	})

	ginkgo.AfterEach(func() {
		commands.DeleteChainConfig(chainName)
		err := utils.DeleteKey(keyName)
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CleanNetwork()
	})

	ginkgo.It("deploy chain to testnet", func() {
		deployChainToTestnet()
	})

	ginkgo.It("deploy chain to mainnet", ginkgo.Label("local_machine"), func() {
		// fund ledger address
		err := utils.FundLedgerAddress(1000000000) // 1 LUX in nanoLUX
		gomega.Expect(err).Should(gomega.BeNil())
		fmt.Println()
		fmt.Println(luxlog.LightRed.Wrap("DEPLOYING CHAIN. VERIFY LEDGER ADDRESS HAS CUSTOM HRP BEFORE SIGNING"))
		s := commands.SimulateMainnetDeploy(chainName)
		// deploy
		chainID, err := utils.ParsePublicDeployOutput(s, utils.ChainIDParseType)
		gomega.Expect(err).Should(gomega.BeNil())
		// add validators to chain
		nodeInfos, err := utils.GetNodesInfo()
		gomega.Expect(err).Should(gomega.BeNil())
		nodeIdx := 1
		for _, nodeInfo := range nodeInfos {
			fmt.Println(luxlog.LightRed.Wrap(
				fmt.Sprintf("ADDING VALIDATOR %d of %d. VERIFY LEDGER ADDRESS HAS CUSTOM HRP BEFORE SIGNING", nodeIdx, len(nodeInfos))))
			start := time.Now().Add(time.Second * 30).UTC().Format("2006-01-02 15:04:05")
			_ = commands.SimulateMainnetAddValidator(chainName, nodeInfo.ID, start, "24h", "20")
			nodeIdx++
		}
		fmt.Println(luxlog.LightBlue.Wrap("EXECUTING NON INTERACTIVE PART OF THE TEST: JOIN/WHITELIST/WAIT/HARDHAT"))
		// join to copy vm binary and update config file
		for _, nodeInfo := range nodeInfos {
			_ = commands.SimulateMainnetJoin(chainName, nodeInfo.ConfigFile, nodeInfo.PluginDir, nodeInfo.ID)
		}
		// get and check whitelisted chains from config file
		var whitelistedChains string
		for _, nodeInfo := range nodeInfos {
			whitelistedChains, err = utils.GetWhitelistedChainsFromConfigFile(nodeInfo.ConfigFile)
			gomega.Expect(err).Should(gomega.BeNil())
			whitelistedChainsSlice := strings.Split(whitelistedChains, ",")
			gomega.Expect(whitelistedChainsSlice).Should(gomega.ContainElement(chainID))
		}
		// update nodes whitelisted chains
		err = utils.RestartNodesWithWhitelistedChains(whitelistedChains)
		gomega.Expect(err).Should(gomega.BeNil())
		// wait for chain walidators to be up
		err = utils.WaitChainValidators(chainID, nodeInfos)
		gomega.Expect(err).Should(gomega.BeNil())

		// this is a simulation, so app is probably saving the info in the
		// `local network` section of the sidecar instead of the `testnet` section...
		// ...need to manipulate the `testnet` section of the sidecar to contain the chainID info
		// so that the `stats` command for `testnet` can find it
		output := commands.SimulateGetChainStatsTestnet(chainName, chainID)
		gomega.Expect(output).Should(gomega.Not(gomega.BeNil()))
		gomega.Expect(output).Should(gomega.ContainSubstring("Current validators"))
		gomega.Expect(output).Should(gomega.ContainSubstring("NodeID-"))
		gomega.Expect(output).Should(gomega.ContainSubstring("No pending validators found"))
	})

	ginkgo.It("can transform a deployed EVM chain to elastic chain only on testnet", func() {
		chainIDStr, _ := deployChainToTestnet()
		chainID, err := ids.FromString(chainIDStr)
		gomega.Expect(err).Should(gomega.BeNil())

		// GetCurrentSupply will return error if queried for non-elastic chain
		err = chain.GetCurrentSupply(chainID)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		_, err = commands.SimulateTestnetTransformChain(chainName, keyName)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err := utils.ElasticChainConfigExists(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		// GetCurrentSupply will return result if queried for elastic chain
		err = chain.GetCurrentSupply(chainID)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.SimulateTestnetTransformChain(chainName, keyName)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		commands.DeleteElasticChainConfig(chainName)
	})

	ginkgo.It("remove validator testnet", func() {
		chainIDStr, nodeInfos := deployChainToTestnet()

		// pick a validator to remove
		var validatorToRemove string
		for _, nodeInfo := range nodeInfos {
			validatorToRemove = nodeInfo.ID
			break
		}

		// confirm current validator set
		chainID, err := ids.FromString(chainIDStr)
		gomega.Expect(err).Should(gomega.BeNil())
		validators, err := chain.GetChainValidators(chainID)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(validators)).Should(gomega.Equal(5))

		// Check that the validatorToRemove is in the chain validator set
		var found bool
		for _, validator := range validators {
			if validator.NodeID.String() == validatorToRemove {
				found = true
				break
			}
		}
		gomega.Expect(found).Should(gomega.BeTrue())

		// remove validator
		_ = commands.SimulateTestnetRemoveValidator(chainName, keyName, validatorToRemove)

		// confirm current validator set
		validators, err = chain.GetChainValidators(chainID)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(validators)).Should(gomega.Equal(4))

		// Check that the validatorToRemove is NOT in the chain validator set
		found = false
		for _, validator := range validators {
			if validator.NodeID.String() == validatorToRemove {
				found = true
				break
			}
		}
		gomega.Expect(found).Should(gomega.BeFalse())
	})
})
