// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
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
	subnet1 = "wagmi"
	subnet2 = "spaces"
	vmid1   = "srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
	vmid2   = "sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm"

	testRepo = "https://github.com/luxfi/test-subnet-configs"
)

var _ = ginkgo.Describe("[LPM]", func() {
	ginkgo.BeforeEach(func() {
		// Clean up any existing LPM installations and repositories before each test
		// This ensures a clean state for each test run
		// See issue: https://github.com/luxfi/cli/issues/244
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
		// Clean up LPM repository after test completion
		utils.RemoveLPMRepo()
	})

	ginkgo.It("can import from lux-core", func() {
		ginkgo.Skip("Pending implementation of lux-core import functionality")
		repo := "luxfi/plugins-core"
		commands.ImportSubnetConfig(repo, subnet1)
	})

	ginkgo.It("can import from url", func() {
		ginkgo.Skip("Pending implementation of URL import functionality")
		branch := "master"
		commands.ImportSubnetConfigFromURL(testRepo, branch, subnet2)
	})
})
