// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package subnet

import (
	"fmt"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	subnetName  = "e2eSubnetTest"
	controlKeys = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
	keyName     = "ewoq"
)

func deploySubnetToTestnet() (string, map[string]utils.NodeInfo) {
	// deploy
	s := commands.SimulateTestnetDeploy(subnetName, keyName, controlKeys)
	subnetID, err := utils.ParsePublicDeployOutput(s)
	gomega.Expect(err).Should(gomega.BeNil())
	// add validators to subnet
	nodeInfos, err := utils.GetNodesInfo()
	gomega.Expect(err).Should(gomega.BeNil())
	for _, nodeInfo := range nodeInfos {
		start := time.Now().Add(time.Second * 30).UTC().Format("2006-01-02 15:04:05")
		_ = commands.SimulateTestnetAddValidator(subnetName, keyName, nodeInfo.ID, start, "24h", "20")
	}
	// join to copy vm binary and update config file
	for _, nodeInfo := range nodeInfos {
		_ = commands.SimulateTestnetJoin(subnetName, nodeInfo.ConfigFile, nodeInfo.PluginDir, nodeInfo.ID)
	}
	// get and check whitelisted subnets from config file
	var whitelistedSubnets string
	for _, nodeInfo := range nodeInfos {
		whitelistedSubnets, err = utils.GetWhitelistedSubnetsFromConfigFile(nodeInfo.ConfigFile)
		gomega.Expect(err).Should(gomega.BeNil())
		whitelistedSubnetsSlice := strings.Split(whitelistedSubnets, ",")
		gomega.Expect(whitelistedSubnetsSlice).Should(gomega.ContainElement(subnetID))
	}
	// update nodes whitelisted subnets
	err = utils.RestartNodesWithWhitelistedSubnets(whitelistedSubnets)
	gomega.Expect(err).Should(gomega.BeNil())
	// wait for subnet walidators to be up
	err = utils.WaitSubnetValidators(subnetID, nodeInfos)
	gomega.Expect(err).Should(gomega.BeNil())
	return subnetID, nodeInfos
}

var _ = ginkgo.Describe("[Public Subnet]", func() {
	ginkgo.BeforeEach(func() {
		// key
		_ = utils.DeleteKey(keyName)
		output, err := commands.CreateKeyFromPath(keyName, utils.EwoqKeyPath)
		if err != nil {
			fmt.Println(output)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		// subnet config
		_ = utils.DeleteConfigs(subnetName)
		_, luxVersion := commands.CreateSubnetEvmConfig(subnetName, utils.SubnetEvmGenesisPath)

		// local network
		commands.StartNetworkWithVersion(luxVersion)
	})

	ginkgo.AfterEach(func() {
		commands.DeleteSubnetConfig(subnetName)
		err := utils.DeleteKey(keyName)
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CleanNetwork()
	})

	ginkgo.It("deploy subnet to testnet", func() {
		deploySubnetToTestnet()
	})

	ginkgo.It("deploy subnet to mainnet", ginkgo.Label("local_machine"), func() {
		// fund ledger address
		err := utils.FundLedgerAddress()
		gomega.Expect(err).Should(gomega.BeNil())
		fmt.Println()
		fmt.Println(luxlog.LightRed.Wrap("DEPLOYING SUBNET. VERIFY LEDGER ADDRESS HAS CUSTOM HRP BEFORE SIGNING"))
		s := commands.SimulateMainnetDeploy(subnetName)
		// deploy
		subnetID, err := utils.ParsePublicDeployOutput(s)
		gomega.Expect(err).Should(gomega.BeNil())
		// add validators to subnet
		nodeInfos, err := utils.GetNodesInfo()
		gomega.Expect(err).Should(gomega.BeNil())
		nodeIdx := 1
		for _, nodeInfo := range nodeInfos {
			fmt.Println(luxlog.LightRed.Wrap(
				fmt.Sprintf("ADDING VALIDATOR %d of %d. VERIFY LEDGER ADDRESS HAS CUSTOM HRP BEFORE SIGNING", nodeIdx, len(nodeInfos))))
			start := time.Now().Add(time.Second * 30).UTC().Format("2006-01-02 15:04:05")
			_ = commands.SimulateMainnetAddValidator(subnetName, nodeInfo.ID, start, "24h", "20")
			nodeIdx++
		}
		fmt.Println(luxlog.LightBlue.Wrap("EXECUTING NON INTERACTIVE PART OF THE TEST: JOIN/WHITELIST/WAIT/HARDHAT"))
		// join to copy vm binary and update config file
		for _, nodeInfo := range nodeInfos {
			_ = commands.SimulateMainnetJoin(subnetName, nodeInfo.ConfigFile, nodeInfo.PluginDir, nodeInfo.ID)
		}
		// get and check whitelisted subnets from config file
		var whitelistedSubnets string
		for _, nodeInfo := range nodeInfos {
			whitelistedSubnets, err = utils.GetWhitelistedSubnetsFromConfigFile(nodeInfo.ConfigFile)
			gomega.Expect(err).Should(gomega.BeNil())
			whitelistedSubnetsSlice := strings.Split(whitelistedSubnets, ",")
			gomega.Expect(whitelistedSubnetsSlice).Should(gomega.ContainElement(subnetID))
		}
		// update nodes whitelisted subnets
		err = utils.RestartNodesWithWhitelistedSubnets(whitelistedSubnets)
		gomega.Expect(err).Should(gomega.BeNil())
		// wait for subnet walidators to be up
		err = utils.WaitSubnetValidators(subnetID, nodeInfos)
		gomega.Expect(err).Should(gomega.BeNil())

		// this is a simulation, so app is probably saving the info in the
		// `local network` section of the sidecar instead of the `testnet` section...
		// ...need to manipulate the `testnet` section of the sidecar to contain the subnetID info
		// so that the `stats` command for `testnet` can find it
		output := commands.SimulateGetSubnetStatsTestnet(subnetName, subnetID)
		gomega.Expect(output).Should(gomega.Not(gomega.BeNil()))
		gomega.Expect(output).Should(gomega.ContainSubstring("Current validators"))
		gomega.Expect(output).Should(gomega.ContainSubstring("NodeID-"))
		gomega.Expect(output).Should(gomega.ContainSubstring("No pending validators found"))
	})

	ginkgo.It("can transform a deployed SubnetEvm subnet to elastic subnet only on testnet", func() {
		subnetIDStr, _ := deploySubnetToTestnet()
		subnetID, err := ids.FromString(subnetIDStr)
		gomega.Expect(err).Should(gomega.BeNil())

		// GetCurrentSupply will return error if queried for non-elastic subnet
		err = subnet.GetCurrentSupply(subnetID)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		_, err = commands.SimulateTestnetTransformSubnet(subnetName, keyName)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err := utils.ElasticSubnetConfigExists(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		// GetCurrentSupply will return result if queried for elastic subnet
		err = subnet.GetCurrentSupply(subnetID)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.SimulateTestnetTransformSubnet(subnetName, keyName)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		commands.DeleteElasticSubnetConfig(subnetName)
	})

	ginkgo.It("remove validator testnet", func() {
		subnetIDStr, nodeInfos := deploySubnetToTestnet()

		// pick a validator to remove
		var validatorToRemove string
		for _, nodeInfo := range nodeInfos {
			validatorToRemove = nodeInfo.ID
			break
		}

		// confirm current validator set
		subnetID, err := ids.FromString(subnetIDStr)
		gomega.Expect(err).Should(gomega.BeNil())
		validators, err := subnet.GetSubnetValidators(subnetID)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(validators)).Should(gomega.Equal(5))

		// Check that the validatorToRemove is in the subnet validator set
		var found bool
		for _, validator := range validators {
			if validator.NodeID.String() == validatorToRemove {
				found = true
				break
			}
		}
		gomega.Expect(found).Should(gomega.BeTrue())

		// remove validator
		_ = commands.SimulateTestnetRemoveValidator(subnetName, keyName, validatorToRemove)

		// confirm current validator set
		validators, err = subnet.GetSubnetValidators(subnetID)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(validators)).Should(gomega.Equal(4))

		// Check that the validatorToRemove is NOT in the subnet validator set
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
