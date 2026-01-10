// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"fmt"

	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	chainName = "e2eChainTest"
)

var _ = ginkgo.Describe("[Network]", ginkgo.Ordered, func() {
	ginkgo.AfterEach(func() {
		commands.CleanNetwork()
		err := utils.DeleteConfigs(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("can stop and restart a deployed chain", func() {
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

		// Deploy greeter contract
		scriptOutput, scriptErr, err := utils.RunHardhatScript(utils.GreeterScript)
		if scriptErr != "" {
			fmt.Println(scriptOutput)
			fmt.Println(scriptErr)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.ParseGreeterAddress(scriptOutput)
		gomega.Expect(err).Should(gomega.BeNil())

		// Check greeter script before stopping
		scriptOutput, scriptErr, err = utils.RunHardhatScript(utils.GreeterCheck)
		if scriptErr != "" {
			fmt.Println(scriptOutput)
			fmt.Println(scriptErr)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		_ = commands.StopNetwork()
		restartOutput := commands.StartNetwork()
		rpcs, err = utils.ParseRPCsFromOutput(restartOutput)
		if err != nil {
			fmt.Println(restartOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))
		rpc = rpcs[0]

		err = utils.SetHardhatRPC(rpc)
		gomega.Expect(err).Should(gomega.BeNil())

		// Check greeter contract has right value
		scriptOutput, scriptErr, err = utils.RunHardhatScript(utils.GreeterCheck)
		if scriptErr != "" {
			fmt.Println(scriptOutput)
			fmt.Println(scriptErr)
		}
		gomega.Expect(err).Should(gomega.BeNil())

		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("clean hard deletes plugin binaries", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		deployOutput := commands.DeployChainLocally(chainName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(rpcs).Should(gomega.HaveLen(1))

		// check that plugin binaries exist
		plugins, err := utils.GetPluginBinaries()
		// should have only evm binary
		gomega.Expect(len(plugins)).Should(gomega.Equal(1))
		gomega.Expect(err).Should(gomega.BeNil())

		commands.CleanNetwork()

		// check that plugin binaries exist
		plugins, err = utils.GetPluginBinaries()
		// should be empty
		gomega.Expect(len(plugins)).Should(gomega.Equal(0))
		gomega.Expect(err).Should(gomega.BeNil())

		commands.DeleteChainConfig(chainName)
	})
})
