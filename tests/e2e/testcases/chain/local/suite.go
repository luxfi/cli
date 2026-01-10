// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	chainName       = "e2eChainTest"
	secondChainName = "e2eSecondChainTest"
	confPath        = "tests/e2e/assets/test_cli.json"
	stakeAmount     = "2000"
	stakeDuration   = "336h"
	localNetwork    = "Local Network"
)

var (
	mapping map[string]string
	err     error
)

var _ = ginkgo.Describe("[Local Chain]", ginkgo.Ordered, func() {
	_ = ginkgo.BeforeAll(func() {
		mapper := utils.NewVersionMapper()
		mapping, err = utils.GetVersionMapping(mapper)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		err := utils.DeleteConfigs(chainName)
		if err != nil {
			fmt.Println("Clean network error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.DeleteConfigs(secondChainName)
		if err != nil {
			fmt.Println("Delete config error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		// delete custom vm
		utils.DeleteCustomBinary(chainName)
		utils.DeleteCustomBinary(secondChainName)
	})

	ginkgo.It("can deploy a custom vm chain to local", func() {
		customVMPath, err := utils.DownloadCustomVMBin(mapping[utils.SoloEVMKey1])
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CreateCustomVMConfig(chainName, utils.EVMGenesisPath, customVMPath)
		deployOutput := commands.DeployChainLocallyWithVersion(chainName, mapping[utils.SoloLuxKey])
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

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can deploy a EVM chain to local", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		deployOutput := commands.DeployChainLocally(chainName)
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

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can transform a deployed EVM chain to elastic chain only once", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		deployOutput := commands.DeployChainLocally(chainName)
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

		// GetCurrentSupply will return error if queried for non-elastic chain
		err = utils.GetCurrentSupply(chainName)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		_, err = commands.TransformElasticChainLocally(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err := utils.ElasticChainConfigExists(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		// GetCurrentSupply will return result if queried for elastic chain
		err = utils.GetCurrentSupply(chainName)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.TransformElasticChainLocally(chainName)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		commands.DeleteChainConfig(chainName)
		commands.DeleteElasticChainConfig(chainName)
	})

	ginkgo.It("can transform chain to elastic chain and automatically transform validators to permissionless", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		deployOutput := commands.DeployChainLocally(chainName)
		_, err = utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.TransformElasticChainLocallyandTransformValidators(chainName, stakeAmount)
		gomega.Expect(err).Should(gomega.BeNil())

		// GetCurrentSupply will return result if queried for elastic chain
		err = utils.GetCurrentSupply(chainName)
		gomega.Expect(err).Should(gomega.BeNil())

		// wait for the last node to be current validator
		time.Sleep(constants.StakingMinimumLeadTime)

		isPendingValidator, err := utils.CheckAllNodesAreCurrentValidators(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(isPendingValidator).Should(gomega.BeTrue())

		exists, err := utils.AllPermissionlessValidatorExistsInSidecar(chainName, localNetwork)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		commands.DeleteChainConfig(chainName)
		commands.DeleteElasticChainConfig(chainName)
	})

	ginkgo.It("can add permissionless validator to elastic chain", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		deployOutput := commands.DeployChainLocally(chainName)
		_, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.TransformElasticChainLocally(chainName)
		gomega.Expect(err).Should(gomega.BeNil())

		nodeIDs, err := utils.GetValidators(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(nodeIDs)).Should(gomega.Equal(5))

		_, err = commands.RemoveValidator(chainName, nodeIDs[0])
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.AddPermissionlessValidator(chainName, nodeIDs[0], stakeAmount, stakeDuration)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err := utils.PermissionlessValidatorExistsInSidecar(chainName, nodeIDs[0], localNetwork)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		isPendingValidator, err := utils.IsNodeInPendingValidator(chainName, nodeIDs[0])
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(isPendingValidator).Should(gomega.BeTrue())

		_, err = commands.RemoveValidator(chainName, nodeIDs[1])
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.AddPermissionlessValidator(chainName, nodeIDs[1], stakeAmount, stakeDuration)
		gomega.Expect(err).Should(gomega.BeNil())
		exists, err = utils.PermissionlessValidatorExistsInSidecar(chainName, nodeIDs[1], localNetwork)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeTrue())

		isPendingValidator, err = utils.IsNodeInPendingValidator(chainName, nodeIDs[1])
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(isPendingValidator).Should(gomega.BeTrue())

		commands.DeleteChainConfig(chainName)
		commands.DeleteElasticChainConfig(chainName)
	})

	ginkgo.It("can load viper config and setup node properties for local deploy", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		deployOutput := commands.DeployChainLocallyWithViperConf(chainName, confPath)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]
		gomega.Expect(rpc).Should(gomega.HavePrefix("http://0.0.0.0:"))

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can't deploy the same chain twice to local", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)

		deployOutput := commands.DeployChainLocally(chainName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		deployOutput = commands.DeployChainLocally(chainName)
		rpcs, err = utils.ParseRPCsFromOutput(deployOutput)
		if err == nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(rpcs).Should(gomega.HaveLen(0))
		gomega.Expect(deployOutput).Should(gomega.ContainSubstring("has already been deployed"))
	})

	ginkgo.It("can deploy multiple chains to local", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		commands.CreateEVMConfig(secondChainName, utils.EVMGenesis2Path)

		deployOutput := commands.DeployChainLocally(chainName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		deployOutput = commands.DeployChainLocally(secondChainName)
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

		commands.DeleteChainConfig(chainName)
		commands.DeleteChainConfig(secondChainName)
	})

	ginkgo.It("can deploy custom chain config", func() {
		commands.CreateEVMConfig(chainName, utils.EVMAllowFeeRecpPath)

		addr := "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"

		chainConfig := "{\"feeRecipient\": \"" + addr + "\"}"

		// create a chain config in tmp
		file, err := os.CreateTemp("", constants.ChainConfigFile+"*")
		gomega.Expect(err).Should(gomega.BeNil())
		err = os.WriteFile(file.Name(), []byte(chainConfig), constants.DefaultPerms755)
		gomega.Expect(err).Should(gomega.BeNil())

		commands.ConfigureChainConfig(chainName, file.Name())

		deployOutput := commands.DeployChainLocally(chainName)
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

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can deploy with custom per chain config node", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)

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

		// configure the chain
		file, err := os.CreateTemp("", constants.PerNodeChainConfigFileName+"*")
		gomega.Expect(err).Should(gomega.BeNil())
		err = os.WriteFile(file.Name(), []byte(perNodeChainConfig), constants.DefaultPerms755)
		gomega.Expect(err).Should(gomega.BeNil())
		commands.ConfigurePerNodeChainConfig(chainName, file.Name())

		// deploy
		deployOutput := commands.DeployChainLocally(chainName)
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

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can list a chain's validators", func() {
		nodeIDs := []string{
			"NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
			"NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
			"NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
			"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
			"NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
		}

		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		deployOutput := commands.DeployChainLocally(chainName)
		_, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		output, err := commands.ListValidators(chainName, "local")
		gomega.Expect(err).Should(gomega.BeNil())

		for _, nodeID := range nodeIDs {
			gomega.Expect(output).Should(gomega.ContainSubstring(nodeID))
		}

		commands.DeleteChainConfig(chainName)
	})
})

var _ = ginkgo.Describe("[Chain Compatibility]", func() {
	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		if err := utils.DeleteConfigs(chainName); err != nil {
			fmt.Println("Clean network error:", err)
			gomega.Expect(err).Should(gomega.BeNil())
		}

		if err := utils.DeleteConfigs(secondChainName); err != nil {
			fmt.Println("Delete config error:", err)
			gomega.Expect(err).Should(gomega.BeNil())
		}
	})

	ginkgo.It("can deploy a evm with specific version", func() {
		evmVersion := "v0.7.9"

		commands.CreateEVMConfigWithVersion(chainName, utils.EVMGenesisPath, evmVersion)
		deployOutput := commands.DeployChainLocally(chainName)
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

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can't deploy conflicting vm versions", func() {
		// Using versions with different RPC protocols
		evmVersion1 := "v0.7.9" // RPC 42
		evmVersion2 := "v0.7.5" // RPC 41

		commands.CreateEVMConfigWithVersion(chainName, utils.EVMGenesisPath, evmVersion1)
		commands.CreateEVMConfigWithVersion(secondChainName, utils.EVMGenesis2Path, evmVersion2)

		deployOutput := commands.DeployChainLocally(chainName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		commands.DeployChainLocallyExpectError(secondChainName)

		commands.DeleteChainConfig(chainName)
		commands.DeleteChainConfig(secondChainName)
	})
})
