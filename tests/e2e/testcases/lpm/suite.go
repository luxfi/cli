// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package lpm

import (
	"fmt"

	"github.com/luxdefi/cli/tests/e2e/commands"
	"github.com/luxdefi/cli/tests/e2e/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	subnet1 = "wagmi"
	subnet2 = "spaces"
	vmid1   = "srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
	vmid2   = "sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm"

	testRepo = "https://github.com/luxdefi/test-subnet-configs"
)

var _ = ginkgo.Describe("[LPM]", func() {
	ginkgo.BeforeEach(func() {
		// TODO this is a bit coarse, but I'm not sure a better solution is possible
		// without modifications to the LPM.
		// More details: https://github.com/luxdefi/cli/issues/244
		utils.RemoveLPMRepo()
	})

	ginkgo.AfterEach(func() {
		err := utils.DeleteConfigs(subnet1)
		if err != nil {
			fmt.Println("Clean network error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.DeleteConfigs(subnet2)
		if err != nil {
			fmt.Println("Delete config error:", err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		utils.DeleteLPMBin(vmid1)
		utils.DeleteLPMBin(vmid2)
		// TODO same as above
		utils.RemoveLPMRepo()
	})

	ginkgo.It("can import from lux-core", func() {
		ginkgo.Skip("TODO")
		repo := "luxdefi/plugins-core"
		commands.ImportSubnetConfig(repo, subnet1)
	})

	ginkgo.It("can import from url", func() {
		ginkgo.Skip("TODO")
		branch := "master"
		commands.ImportSubnetConfigFromURL(testRepo, branch, subnet2)
	})
})
