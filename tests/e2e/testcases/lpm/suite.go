// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpm

import (
	"fmt"

	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	chain1 = "wagmi"
	chain2 = "spaces"
	vmid1  = "srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
	vmid2  = "sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm"

	testRepo = "https://github.com/luxfi/test-chain-configs"
)

var _ = ginkgo.Describe("[LPM]", func() {
	ginkgo.BeforeEach(func() {
		// Clean LPM repository state before each test
		// This ensures tests start with a clean state - see issue #244
		utils.RemoveLPMRepo()
	})

	ginkgo.AfterEach(func() {
		err := utils.DeleteConfigs(chain1)
		if err != nil {
			fmt.Println("Clean network error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.DeleteConfigs(chain2)
		if err != nil {
			fmt.Println("Delete config error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		utils.DeleteLPMBin(vmid1)
		utils.DeleteLPMBin(vmid2)
		// Clean LPM repository state after test completion
		utils.RemoveLPMRepo()
	})

	ginkgo.It("can import from core", func() {
		repo := "luxfi/plugins-core"
		commands.ImportChainConfig(repo, chain1)
	})

	ginkgo.It("can import from url", func() {
		branch := "master"
		commands.ImportChainConfigFromURL(testRepo, branch, chain2)
	})
})
