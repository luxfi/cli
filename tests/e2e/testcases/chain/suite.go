// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const chainName = "e2eChainTest"

var (
	mapping map[string]string
	err     error
)

var _ = ginkgo.Describe("[Chain]", ginkgo.Ordered, func() {
	_ = ginkgo.BeforeAll(func() {
		mapper := utils.NewVersionMapper()
		mapping, err = utils.GetVersionMapping(mapper)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("can create and delete a chain evm config", func() {
		commands.CreateEVMConfig(chainName, utils.EVMGenesisPath)
		commands.DeleteChainConfig(chainName)
	})

	ginkgo.It("can create and delete a custom vm chain config", func() {
		// let's use a EVM version which would be compatible with an existing Lux
		customVMPath, err := utils.DownloadCustomVMBin(mapping[utils.SoloEVMKey1])
		gomega.Expect(err).Should(gomega.BeNil())

		commands.CreateCustomVMConfig(chainName, utils.EVMGenesisPath, customVMPath)
		commands.DeleteChainConfig(chainName)
		exists, err := utils.ChainCustomVMExists(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(exists).Should(gomega.BeFalse())
	})
})
