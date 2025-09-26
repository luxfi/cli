// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package packageman

import (
	"fmt"

	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	subnetName       = "e2eSubnetTest"
	secondSubnetName = "e2eSecondSubnetTest"
)

var (
	binaryToVersion map[string]string
	err             error
)

var _ = ginkgo.Describe("[Package Management]", ginkgo.Ordered, func() {
	_ = ginkgo.BeforeAll(func() {
		mapper := utils.NewVersionMapper()
		binaryToVersion, err = utils.GetVersionMapping(mapper)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.BeforeEach(func() {
		commands.CleanNetworkHard()
	})

	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		err := utils.DeleteConfigs(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.DeleteConfigs(secondSubnetName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("can deploy a subnet with evm version", func() {
		// check evm install precondition
		gomega.Expect(utils.CheckEVMExists(binaryToVersion[utils.SoloEVMKey1])).Should(gomega.BeFalse())
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.SoloLuxKey])).Should(gomega.BeFalse())

		commands.CreateSubnetEvmConfigWithVersion(subnetName, utils.SubnetEvmGenesisPath, binaryToVersion[utils.SoloEVMKey1])
		deployOutput := commands.DeploySubnetLocallyWithVersion(subnetName, binaryToVersion[utils.SoloLuxKey])
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

		// check evm install
		gomega.Expect(utils.CheckEVMExists(binaryToVersion[utils.SoloEVMKey1])).Should(gomega.BeTrue())
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.SoloLuxKey])).Should(gomega.BeTrue())

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can deploy multiple evm versions", func() {
		// check evm install precondition
		gomega.Expect(utils.CheckEVMExists(binaryToVersion[utils.SoloEVMKey1])).Should(gomega.BeFalse())
		gomega.Expect(utils.CheckEVMExists(binaryToVersion[utils.SoloEVMKey2])).Should(gomega.BeFalse())

		commands.CreateSubnetEvmConfigWithVersion(subnetName, utils.SubnetEvmGenesisPath, binaryToVersion[utils.SoloEVMKey1])
		commands.CreateSubnetEvmConfigWithVersion(secondSubnetName, utils.SubnetEvmGenesis2Path, binaryToVersion[utils.SoloEVMKey2])

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

		// check evm install
		gomega.Expect(utils.CheckEVMExists(binaryToVersion[utils.SoloEVMKey1])).Should(gomega.BeTrue())
		gomega.Expect(utils.CheckEVMExists(binaryToVersion[utils.SoloEVMKey2])).Should(gomega.BeTrue())

		commands.DeleteSubnetConfig(subnetName)
		commands.DeleteSubnetConfig(secondSubnetName)
	})

	ginkgo.It("can deploy with multiple node versions", func() {
		// check lux install precondition
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.MultiLux1Key])).Should(gomega.BeFalse())
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.MultiLux2Key])).Should(gomega.BeFalse())

		commands.CreateSubnetEvmConfigWithVersion(subnetName, utils.SubnetEvmGenesisPath, binaryToVersion[utils.MultiLuxEVMKey])
		deployOutput := commands.DeploySubnetLocallyWithVersion(subnetName, binaryToVersion[utils.MultiLux1Key])
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]

		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		// Deploy greeter contract
		scriptOutput, scriptErr, err := utils.RunHardhatScript(utils.GreeterScript)
		if scriptErr != "" {
			fmt.Println(scriptOutput)
			fmt.Println(scriptErr)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.ParseGreeterAddress(scriptOutput)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		// check lux install
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.MultiLux1Key])).Should(gomega.BeTrue())
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.MultiLux2Key])).Should(gomega.BeFalse())

		commands.CleanNetwork()

		deployOutput = commands.DeploySubnetLocallyWithVersion(subnetName, binaryToVersion[utils.MultiLux2Key])
		rpcs, err = utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc = rpcs[0]

		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		// check lux install
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.MultiLux1Key])).Should(gomega.BeTrue())
		gomega.Expect(utils.CheckLuxExists(binaryToVersion[utils.MultiLux2Key])).Should(gomega.BeTrue())

		commands.DeleteSubnetConfig(subnetName)
	})
})
