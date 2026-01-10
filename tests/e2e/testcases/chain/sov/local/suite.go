// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"

	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	chainName       = "e2eChainTest"
	secondChainName = "e2eSecondChainTest"
	confPath        = "tests/e2e/assets/test_lux-cli.json"
)

var (
	mapping map[string]string
	err     error
)

var _ = ginkgo.Describe("[Local Chain SOV]", ginkgo.Ordered, func() {
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

	ginkgo.It("can deploy a custom vm chain to local SOV", func() {
		customVMPath, err := utils.DownloadCustomVMBin(mapping[utils.SoloEVMKey1])
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CreateCustomVMConfigSOV(chainName, utils.EVMGenesisPoaPath, customVMPath)
		deployOutput := commands.DeployChainLocallyWithVersionSOV(chainName, mapping[utils.SoloLuxdKey])
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

	ginkgo.It("can deploy a EVM chain to local SOV", func() {
		commands.CreateEVMConfigSOV(chainName, utils.EVMGenesisPoaPath)
		deployOutput := commands.DeployChainLocallySOV(chainName)
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

	ginkgo.It("can load viper config and setup node properties for local deploy SOV", func() {
		commands.CreateEVMConfigSOV(chainName, utils.EVMGenesisPoaPath)
		deployOutput := commands.DeployChainLocallyWithViperConfSOV(chainName, confPath)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc := rpcs[0]
		gomega.Expect(rpc).Should(gomega.HavePrefix("http://127.0.0.1:"))

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can't deploy the same chain twice to local SOV", func() {
		commands.CreateEVMConfigSOV(chainName, utils.EVMGenesisPoaPath)

		deployOutput := commands.DeployChainLocallySOV(chainName)
		fmt.Println(deployOutput)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		out, err := commands.DeployChainLocallyWithArgsAndOutputSOV(chainName, "", "")
		gomega.Expect(err).Should(gomega.HaveOccurred())
		deployOutput = string(out)
		rpcs, err = utils.ParseRPCsFromOutput(deployOutput)
		if err == nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(rpcs).Should(gomega.HaveLen(0))
		gomega.Expect(deployOutput).Should(gomega.ContainSubstring("has already been deployed"))
	})

	ginkgo.It("can deploy multiple chains to local SOV", func() {
		commands.CreateEVMConfigSOV(chainName, utils.EVMGenesisPoaPath)
		commands.CreateEVMConfigSOV(secondChainName, utils.EVMGenesis2Path)

		deployOutput := commands.DeployChainLocallySOV(chainName)
		rpcs1, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs1).Should(gomega.HaveLen(1))

		deployOutput = commands.DeployChainLocallySOV(secondChainName)
		rpcs2, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs2).Should(gomega.HaveLen(1))

		err = utils.SetHardhatRPC(rpcs1[0])
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.SetHardhatRPC(rpcs2[0])
		gomega.Expect(err).Should(gomega.BeNil())

		err = utils.RunHardhatTests(utils.BaseTest)
		gomega.Expect(err).Should(gomega.BeNil())

		commands.DeleteChainConfig(chainName)
		commands.DeleteChainConfig(secondChainName)
	})

	ginkgo.It("can list a chain's validators SOV", func() {
		nodeIDs := []string{
			"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
			"NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
		}

		commands.CreateEVMConfigSOV(chainName, utils.EVMGenesisPoaPath)
		deployOutput := commands.DeployChainLocallySOV(chainName)
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

	ginkgo.It("can deploy a evm with old version SOV", func() {
		evmVersion := mapping[utils.SoloEVMKey1]
		commands.CreateEVMConfigWithVersionSOV(chainName, utils.EVMGenesisPoaPath, evmVersion)
		deployOutput := commands.DeployChainLocallySOV(chainName)
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

	ginkgo.It("can't deploy conflicting vm versions SOV", func() {
		evmVersion1 := mapping[utils.SoloEVMKey1]
		evmVersion2 := "v0.6.12"

		commands.CreateEVMConfigWithVersionSOV(chainName, utils.EVMGenesisPoaPath, evmVersion1)
		commands.CreateEVMConfigWithVersionSOV(secondChainName, utils.EVMGenesis2Path, evmVersion2)

		deployOutput := commands.DeployChainLocallySOV(chainName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		commands.DeployChainLocallyExpectErrorSOV(secondChainName)

		commands.DeleteChainConfig(chainName)
		commands.DeleteChainConfig(secondChainName)
	})
})
