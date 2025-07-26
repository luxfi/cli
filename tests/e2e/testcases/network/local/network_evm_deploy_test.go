// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/ids"
	"github.com/luxfi/netrunner/client"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	evmSubnetName = "evmLocalTest"
	chainID       = 1337
	nodeCount     = 5
)

var _ = ginkgo.Describe("[Local Network EVM Deploy]", ginkgo.Ordered, func() {
	ginkgo.AfterEach(func() {
		// Clean up after each test
		commands.CleanNetwork()
		err := utils.DeleteConfigs(evmSubnetName)
		if err != nil {
			fmt.Println("Clean network error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("can start 5-node local network with chain ID 1337", func() {
		// Start the network
		ctx := context.Background()
		startCmd := commands.StartNetwork("")
		gomega.Expect(startCmd).Should(gomega.BeNil())
		
		// Verify network is healthy
		cli, err := client.NewClient(constants.LocalNetworkGRPCEndpoint)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Get network status and verify node count
		status, err := cli.Status(ctx)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Check that we have exactly 5 nodes
		nodeNames, err := cli.GetNodeNames(ctx)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(nodeNames).Should(gomega.HaveLen(nodeCount))
		
		// Verify each node is healthy
		for _, nodeName := range nodeNames {
			nodeStatus, err := cli.GetNodeStatus(ctx, nodeName)
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(nodeStatus).Should(gomega.Equal("healthy"))
		}
		
		// Verify chain ID through network config
		networkConfig, err := cli.GetNetworkConfig(ctx)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(networkConfig.NetworkID).Should(gomega.Equal(uint32(chainID)))
	})

	ginkgo.It("can deploy EVM to local primary network", func() {
		// Start the network first
		startCmd := commands.StartNetwork("")
		gomega.Expect(startCmd).Should(gomega.BeNil())
		
		// Create an EVM subnet configuration
		genesisPath := filepath.Join(utils.GetTestDataPath(), "subnet-evm-genesis.json")
		commands.CreateSubnetEvmConfig(evmSubnetName, genesisPath)
		
		// Deploy the subnet to local network
		deployOutput := commands.DeploySubnetLocally(evmSubnetName)
		
		// Parse the RPC endpoints from output
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println("Deploy output:", deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		
		// Verify the EVM is accessible
		rpc := rpcs[0]
		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Run basic tests to ensure EVM is working
		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Get subnet info to verify deployment
		subnetID, err := utils.GetSubnetID(evmSubnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(subnetID).ShouldNot(gomega.Equal(ids.Empty))
		
		// Verify all 5 nodes are validators of the subnet
		validators, err := utils.GetValidators(evmSubnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(validators).Should(gomega.HaveLen(nodeCount))
	})

	ginkgo.It("verifies built-in certificates for all nodes", func() {
		// Start the network
		startCmd := commands.StartNetwork("")
		gomega.Expect(startCmd).Should(gomega.BeNil())
		
		ctx := context.Background()
		cli, err := client.NewClient(constants.LocalNetworkGRPCEndpoint)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Get all node names
		nodeNames, err := cli.GetNodeNames(ctx)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(nodeNames).Should(gomega.HaveLen(nodeCount))
		
		// Expected node IDs from genesis
		expectedNodeIDs := []string{
			"NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
			"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
			"NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
			"NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
			"NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
		}
		
		// Verify each node has the correct NodeID
		for i, nodeName := range nodeNames {
			nodeInfo, err := cli.GetNodeInfo(ctx, nodeName)
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(nodeInfo.ID).Should(gomega.Equal(expectedNodeIDs[i]))
			
			// Verify node has a valid staking certificate
			gomega.Expect(nodeInfo.StakingCert).ShouldNot(gomega.BeEmpty())
		}
	})

	ginkgo.It("can restart network and maintain state", func() {
		// Start the network
		startCmd := commands.StartNetwork("")
		gomega.Expect(startCmd).Should(gomega.BeNil())
		
		// Deploy EVM subnet
		genesisPath := filepath.Join(utils.GetTestDataPath(), "subnet-evm-genesis.json")
		commands.CreateSubnetEvmConfig(evmSubnetName, genesisPath)
		deployOutput := commands.DeploySubnetLocally(evmSubnetName)
		
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		gomega.Expect(err).Should(gomega.BeNil())
		subnetID, err := utils.GetSubnetID(evmSubnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		
		// Stop the network
		stopCmd := commands.StopNetwork()
		gomega.Expect(stopCmd).Should(gomega.BeNil())
		
		// Wait a bit
		time.Sleep(5 * time.Second)
		
		// Restart the network
		restartCmd := commands.StartNetwork("")
		gomega.Expect(restartCmd).Should(gomega.BeNil())
		
		// Verify subnet is still deployed
		restoredSubnetID, err := utils.GetSubnetID(evmSubnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(restoredSubnetID).Should(gomega.Equal(subnetID))
		
		// Verify EVM is still accessible
		err = utils.SetHardhatRPC(rpcs[0])
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())
	})
})