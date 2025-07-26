// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/netrunner/client"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	subnetName      = "testSubnet"
	defaultChainID  = 1337
	nodeCount       = 5
	ewoqEVMAddress  = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
)

var _ = ginkgo.Describe("[Blockchain Deploy]", ginkgo.Ordered, func() {
	ginkgo.BeforeEach(func() {
		// Create test subnet config with chain ID 1337
		commands.CreateSubnetEvmConfigWithChainID(subnetName, utils.SubnetEvmGenesisPath, defaultChainID)
	})

	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		// Cleanup test subnet config
		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("HAPPY PATH: local deploy with 5 nodes and chain ID 1337", func() {
		// Start the network
		ctx := context.Background()
		startOutput := commands.StartNetwork("")
		gomega.Expect(startOutput).Should(gomega.BeNil())
		
		// Verify network has 5 nodes
		cli, err := client.NewClient(constants.LocalNetworkGRPCEndpoint)
		gomega.Expect(err).Should(gomega.BeNil())
		
		nodeNames, err := cli.GetNodeNames(ctx)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(nodeNames).Should(gomega.HaveLen(nodeCount))
		
		// Deploy the subnet
		deployOutput := commands.DeploySubnetLocally(subnetName)
		gomega.Expect(deployOutput).Should(gomega.ContainSubstring("Subnet successfully deployed"))
		
		// Parse RPC endpoints
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		
		// Verify chain ID
		rpc := rpcs[0]
		chainID, err := utils.GetChainID(rpc)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(chainID).Should(gomega.Equal(uint64(defaultChainID)))
		
		// Verify all nodes are validators
		validators, err := utils.GetValidators(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(validators).Should(gomega.HaveLen(nodeCount))
		
		// Run basic EVM tests
		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("HAPPY PATH: verify pre-configured node certificates", func() {
		// Expected node IDs from default genesis
		expectedNodeIDs := []string{
			"NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
			"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
			"NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
			"NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
			"NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
		}
		
		// Start the network
		ctx := context.Background()
		startOutput := commands.StartNetwork("")
		gomega.Expect(startOutput).Should(gomega.BeNil())
		
		cli, err := client.NewClient(constants.LocalNetworkGRPCEndpoint)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Get all node info
		nodeNames, err := cli.GetNodeNames(ctx)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Verify each node has expected node ID
		for i, nodeName := range nodeNames {
			nodeInfo, err := cli.GetNodeInfo(ctx, nodeName)
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(nodeInfo.ID).Should(gomega.Equal(expectedNodeIDs[i]))
		}
	})

	ginkgo.It("HAPPY PATH: deploy with custom allocation", func() {
		// Create subnet with custom allocation
		customAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
		customBalance := "1000000000000000000000" // 1000 tokens
		
		commands.CreateSubnetEvmConfigWithAllocation(
			subnetName,
			utils.SubnetEvmGenesisPath,
			defaultChainID,
			map[common.Address]string{
				customAddress: customBalance,
			},
		)
		
		// Start network and deploy
		startOutput := commands.StartNetwork("")
		gomega.Expect(startOutput).Should(gomega.BeNil())
		
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Verify custom allocation
		balance, err := utils.GetBalance(rpcs[0], customAddress.Hex())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(balance.String()).Should(gomega.Equal(customBalance))
	})

	ginkgo.It("HAPPY PATH: subnet persists after network restart", func() {
		// Start network and deploy subnet
		startOutput := commands.StartNetwork("")
		gomega.Expect(startOutput).Should(gomega.BeNil())
		
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		gomega.Expect(err).Should(gomega.BeNil())
		
		subnetID, err := utils.GetSubnetID(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Deploy a simple contract
		contractAddress, err := utils.DeployTestContract(rpcs[0])
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Stop the network
		stopOutput := commands.StopNetwork()
		gomega.Expect(stopOutput).Should(gomega.BeNil())
		
		// Wait a bit
		time.Sleep(5 * time.Second)
		
		// Restart the network
		restartOutput := commands.StartNetwork("")
		gomega.Expect(restartOutput).Should(gomega.BeNil())
		
		// Verify subnet still exists
		restoredSubnetID, err := utils.GetSubnetID(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(restoredSubnetID).Should(gomega.Equal(subnetID))
		
		// Verify contract still exists
		code, err := utils.GetContractCode(rpcs[0], contractAddress)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(code)).Should(gomega.BeNumerically(">", 2)) // Not just "0x"
	})

	ginkgo.It("HAPPY PATH: verify database is using badgerdb", func() {
		// Start network and deploy subnet
		startOutput := commands.StartNetwork("")
		gomega.Expect(startOutput).Should(gomega.BeNil())
		
		deployOutput := commands.DeploySubnetLocally(subnetName)
		gomega.Expect(deployOutput).Should(gomega.ContainSubstring("Subnet successfully deployed"))
		
		// Get subnet config to verify database type
		config, err := utils.GetSubnetConfig(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Check that database type is badgerdb (default)
		// Since we set badgerdb as default, it should be used
		gomega.Expect(config.DatabaseType).Should(gomega.Or(
			gomega.Equal("badgerdb"),
			gomega.BeEmpty(), // Empty means using default which is badgerdb
		))
	})
})