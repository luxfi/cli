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

var _ = ginkgo.Describe("[Local Chain non SOV]", ginkgo.Ordered, func() {
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

	ginkgo.It("can deploy a custom vm chain to local non SOV", func() {
		customVMPath, err := utils.DownloadCustomVMBin(mapping[utils.SoloEVMKey1])
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CreateCustomVMConfigNonSOV(chainName, utils.EVMGenesisPath, customVMPath)
		deployOutput := commands.DeployChainLocallyWithVersionNonSOV(chainName, mapping[utils.SoloLuxdKey])
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

	ginkgo.It("can deploy a EVM chain to local non SOV", func() {
		commands.CreateEVMConfigNonSOV(chainName, utils.EVMGenesisPath, false)
		deployOutput := commands.DeployChainLocallyNonSOV(chainName)
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

	ginkgo.It("can load viper config and setup node properties for local deploy non SOV", func() {
		commands.CreateEVMConfigNonSOV(chainName, utils.EVMGenesisPath, false)
		deployOutput := commands.DeployChainLocallyWithViperConfNonSOV(chainName, confPath)
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

	ginkgo.It("can't deploy the same chain twice to local non SOV", func() {
		commands.CreateEVMConfigNonSOV(chainName, utils.EVMGenesisPath, false)

		deployOutput := commands.DeployChainLocallyNonSOV(chainName)
		fmt.Println(deployOutput)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		out, err := commands.DeployChainLocallyWithArgsAndOutputNonSOV(chainName, "", "")
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

	ginkgo.It("can deploy multiple chains to local non SOV", func() {
		commands.CreateEVMConfigNonSOV(chainName, utils.EVMGenesisPath, false)
		commands.CreateEVMConfigNonSOV(secondChainName, utils.EVMGenesis2Path, false)

		deployOutput := commands.DeployChainLocallyNonSOV(chainName)
		rpcs1, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs1).Should(gomega.HaveLen(1))

		deployOutput = commands.DeployChainLocallyNonSOV(secondChainName)
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

	ginkgo.It("can deploy a evm with old version non SOV", func() {
		evmVersion := "v0.7.1"

		commands.CreateEVMConfigWithVersionNonSOV(chainName, utils.EVMGenesisPath, evmVersion, false)
		deployOutput := commands.DeployChainLocallyNonSOV(chainName)
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

	ginkgo.It("can't deploy conflicting vm versions non SOV", func() {
		// Use version constants for better maintainability
		evmVersion1 := utils.GetLatestEVMVersion()
		evmVersion2 := utils.GetPreviousEVMVersion()

		commands.CreateEVMConfigWithVersionNonSOV(chainName, utils.EVMGenesisPath, evmVersion1, false)
		commands.CreateEVMConfigWithVersionNonSOV(secondChainName, utils.EVMGenesis2Path, evmVersion2, false)

		deployOutput := commands.DeployChainLocallyNonSOV(chainName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		commands.DeployChainLocallyExpectErrorNonSOV(secondChainName)

		commands.DeleteChainConfig(chainName)
		commands.DeleteChainConfig(secondChainName)
	})
})
