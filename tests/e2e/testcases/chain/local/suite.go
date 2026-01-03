// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package subnet

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	subnetName       = "e2eSubnetTest"
	secondSubnetName = "e2eSecondSubnetTest"
	confPath         = "tests/e2e/assets/test_cli.json"
	stakeAmount      = "2000"
	stakeDuration    = "336h"
	localNetwork     = "Local Network"
)

var (
	mapping map[string]string
	err     error
)

var _ = ginkgo.Describe("[Local Subnet]", ginkgo.Ordered, func() {
	_ = ginkgo.BeforeAll(func() {
		mapper := utils.NewVersionMapper()
		mapping, err = utils.GetVersionMapping(mapper)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		err := utils.DeleteConfigs(subnetName)
		if err != nil {
			fmt.Println("Clean network error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.DeleteConfigs(secondSubnetName)
		if err != nil {
			fmt.Println("Delete config error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		// delete custom vm
		utils.DeleteCustomBinary(subnetName)
		utils.DeleteCustomBinary(secondSubnetName)
	})

	ginkgo.It("can deploy a custom vm subnet to local", func() {
		customVMPath, err := utils.DownloadCustomVMBin(mapping[utils.SoloEVMKey1])
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CreateCustomVMConfig(subnetName, utils.EVMGenesisPath, customVMPath)
		deployOutput := commands.DeploySubnetLocallyWithVersion(subnetName, mapping[utils.SoloLuxKey])
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]

		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can deploy a EVM subnet to local", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]

		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can transform a deployed EVM subnet to elastic subnet only once", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]

		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		// GetCurrentSupply will return error if queried for non-elastic subnet
		err = utils.GetCurrentSupply(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		_, err = commands.TransformElasticChainLocally(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err := utils.ElasticChainConfigExists(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		// GetCurrentSupply will return result if queried for elastic subnet
		err = utils.GetCurrentSupply(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.TransformElasticChainLocally(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		commands.DeleteSubnetConfig(subnetName)
		commands.DeleteElasticChainConfig(subnetName)
	})

	ginkgo.It("can transform subnet to elastic subnet and automatically transform validators to permissionless", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		_, err = utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.TransformElasticChainLocallyandTransformValidators(subnetName, stakeAmount)
		gomega.Expect(err).Should(gomega.BeNil())

		// GetCurrentSupply will return result if queried for elastic subnet
		err = utils.GetCurrentSupply(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		// wait for the last node to be current validator
		time.Sleep(constants.StakingMinimumLeadTime)

		isPendingValidator, err := utils.CheckAllNodesAreCurrentValidators(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(isPendingValidator).Should(gomega.BeTrue())

		exists, err := utils.AllPermissionlessValidatorExistsInSidecar(subnetName, localNetwork)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		commands.DeleteSubnetConfig(subnetName)
		commands.DeleteElasticChainConfig(subnetName)
	})

	ginkgo.It("can add permissionless validator to elastic subnet", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		_, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.TransformElasticChainLocally(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		nodeIDs, err := utils.GetValidators(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(nodeIDs)).Should(gomega.Equal(5))

		_, err = commands.RemoveValidator(subnetName, nodeIDs[0])
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.AddPermissionlessValidator(subnetName, nodeIDs[0], stakeAmount, stakeDuration)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err := utils.PermissionlessValidatorExistsInSidecar(subnetName, nodeIDs[0], localNetwork)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		isPendingValidator, err := utils.IsNodeInPendingValidator(subnetName, nodeIDs[0])
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(isPendingValidator).Should(gomega.BeTrue())

		_, err = commands.RemoveValidator(subnetName, nodeIDs[1])
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.AddPermissionlessValidator(subnetName, nodeIDs[1], stakeAmount, stakeDuration)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err = utils.PermissionlessValidatorExistsInSidecar(subnetName, nodeIDs[1], localNetwork)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		isPendingValidator, err = utils.IsNodeInPendingValidator(subnetName, nodeIDs[1])
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(isPendingValidator).Should(gomega.BeTrue())

		commands.DeleteSubnetConfig(subnetName)
		commands.DeleteElasticChainConfig(subnetName)
	})

	ginkgo.It("can load viper config and setup node properties for local deploy", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)
		deployOutput := commands.DeploySubnetLocallyWithViperConf(subnetName, confPath)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]
		gomega.Expect(rpc).Should(gomega.HavePrefix("http://0.0.0.0:"))

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can't deploy the same subnet twice to local", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		deployOutput = commands.DeploySubnetLocally(subnetName)
		rpcs, err = utils.ParseRPCsFromOutput(deployOutput)
		if err == nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(rpcs).Should(gomega.HaveLen(0))
		gomega.Expect(deployOutput).Should(gomega.ContainSubstring("has already been deployed"))
	})

	ginkgo.It("can deploy multiple subnets to local", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)
		commands.CreateEVMConfig(secondSubnetName, utils.EVMGenesis2Path)

		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		deployOutput = commands.DeploySubnetLocally(secondSubnetName)
		rpcs, err = utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(2))

		err = utils.SetHardhatRPC(rpcs[0])
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.SetHardhatRPC(rpcs[1])
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		commands.DeleteSubnetConfig(subnetName)
		commands.DeleteSubnetConfig(secondSubnetName)
	})

	ginkgo.It("can deploy custom chain config", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMAllowFeeRecpPath)

		addr := "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"

		chainConfig := "{\"feeRecipient\": \"" + addr + "\"}"

		// create a chain config in tmp
		file, err := os.CreateTemp("", constants.ChainConfigFile+"*")
		gomega.Expect(err).Should(gomega.BeNil())
		err = os.WriteFile(file.Name(), []byte(chainConfig), constants.DefaultPerms755)
		gomega.Expect(err).Should(gomega.BeNil())

		commands.ConfigureChainConfig(subnetName, file.Name())

		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		rpc := rpcs[0]
		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		cClient, err := ethclient.Dial(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		ethAddr := common.HexToAddress(addr)
		balance, err := cClient.BalanceAt(context.Background(), ethAddr, nil)
		gomega.Expect(err).Should(gomega.BeNil())

		gomega.Expect(balance.Int64()).Should(gomega.Not(gomega.BeZero()))

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can deploy with custom per chain config node", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		// create per node chain config
		nodesRPCTxFeeCap := map[string]string{
			"node1": "101",
			"node2": "102",
			"node3": "103",
			"node4": "104",
			"node5": "105",
		}
		perNodeChainConfig := "{\n"
		i := 0
		for nodeName, rpcTxFeeCap := range nodesRPCTxFeeCap {
			commaStr := ","
			if i == len(nodesRPCTxFeeCap)-1 {
				commaStr = ""
			}
			perNodeChainConfig += fmt.Sprintf("  \"%s\": {\"rpc-tx-fee-cap\": %s}%s\n", nodeName, rpcTxFeeCap, commaStr)
			i++
		}
		perNodeChainConfig += "}\n"

		// configure the subnet
		file, err := os.CreateTemp("", constants.PerNodeChainConfigFileName+"*")
		gomega.Expect(err).Should(gomega.BeNil())
		err = os.WriteFile(file.Name(), []byte(perNodeChainConfig), constants.DefaultPerms755)
		gomega.Expect(err).Should(gomega.BeNil())
		commands.ConfigurePerNodeChainConfig(subnetName, file.Name())

		// deploy
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		// get blockchain ID
		rpcParts := strings.Split(rpcs[0], "/")
		gomega.Expect(rpcParts).Should(gomega.HaveLen(7))
		blockchainID := rpcParts[5]

		// verify that plugin logs reflect per node configuration
		nodesInfo, err := utils.GetNodesInfo()
		gomega.Expect(err).Should(gomega.BeNil())
		for nodeName, nodeInfo := range nodesInfo {
			logFile := path.Join(nodeInfo.LogDir, blockchainID+".log")
			fileBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: Test code reading from test directories
			gomega.Expect(err).Should(gomega.BeNil())
			rpcTxFeeCap, ok := nodesRPCTxFeeCap[nodeName]
			gomega.Expect(ok).Should(gomega.BeTrue())
			gomega.Expect(fileBytes).Should(gomega.ContainSubstring("RPCTxFeeCap:%s", rpcTxFeeCap))
		}

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can list a subnet's validators", func() {
		nodeIDs := []string{
			"NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
			"NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
			"NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
			"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
			"NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
		}

		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		_, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		output, err := commands.ListValidators(subnetName, "local")
		gomega.Expect(err).Should(gomega.BeNil())

		for _, nodeID := range nodeIDs {
			gomega.Expect(output).Should(gomega.ContainSubstring(nodeID))
		}

		commands.DeleteSubnetConfig(subnetName)
	})
})

var _ = ginkgo.Describe("[Subnet Compatibility]", func() {
	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		if err := utils.DeleteConfigs(subnetName); err != nil {
			fmt.Println("Clean network error:", err)
			gomega.Expect(err).Should(gomega.BeNil())
		}

		if err := utils.DeleteConfigs(secondSubnetName); err != nil {
			fmt.Println("Delete config error:", err)
			gomega.Expect(err).Should(gomega.BeNil())
		}
	})

	ginkgo.It("can deploy a evm with specific version", func() {
		evmVersion := "v0.7.9"

		commands.CreateEVMConfigWithVersion(subnetName, utils.EVMGenesisPath, evmVersion)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]

		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can't deploy conflicting vm versions", func() {
		// Using versions with different RPC protocols
		evmVersion1 := "v0.7.9" // RPC 42
		evmVersion2 := "v0.7.5" // RPC 41

		commands.CreateEVMConfigWithVersion(subnetName, utils.EVMGenesisPath, evmVersion1)
		commands.CreateEVMConfigWithVersion(secondSubnetName, utils.EVMGenesis2Path, evmVersion2)

		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		commands.DeploySubnetLocallyExpectError(secondSubnetName)

		commands.DeleteSubnetConfig(subnetName)
		commands.DeleteSubnetConfig(secondSubnetName)
	})
})
