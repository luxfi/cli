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
	CLIBinary             = "./bin/lux"
	keyName               = "treasury"
	treasuryEVMAddress    = "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
	treasuryPChainAddress = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
)

var luxdVersion string

var _ = ginkgo.Describe("[Etna Add Validator SOV Local]", func() {
	ginkgo.It("Create Etna Chain Config", func() {
		_, luxdVersion = commands.CreateEtnaEVMConfig(
			utils.BlockchainName,
			treasuryEVMAddress,
			commands.PoS,
		)
	})
	ginkgo.It("Can deploy blockchain to localhost and upsize it", func() {
		output := commands.StartNetworkWithVersion(luxdVersion)
		fmt.Println(output)
		output, err := commands.DeployEtnaBlockchain(
			utils.BlockchainName,
			"",
			nil,
			treasuryPChainAddress,
			false, // convertOnly
		)
		gomega.Expect(err).Should(gomega.BeNil())
		fmt.Println(output)
		output, err = commands.AddEtnaChainValidatorToCluster(
			"",
			utils.BlockchainName,
			"",
			treasuryPChainAddress,
			1,
			true,
		)
		gomega.Expect(err).Should(gomega.BeNil())
		fmt.Println(output)
	})

	ginkgo.It("Can destroy local node", func() {
		output, err := commands.DestroyLocalNode(utils.TestLocalNodeName)
		gomega.Expect(err).Should(gomega.BeNil())
		fmt.Println(output)
	})

	ginkgo.It("Can destroy Etna Local Network", func() {
		commands.CleanNetwork()
	})

	ginkgo.It("Can remove Etna Chain Config", func() {
		commands.DeleteChainConfig(utils.BlockchainName)
	})
})
