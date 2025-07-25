// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	_ "github.com/luxfi/cli/tests/e2e/testcases/lpm"
	_ "github.com/luxfi/cli/tests/e2e/testcases/errhandling"
	_ "github.com/luxfi/cli/tests/e2e/testcases/key"
	_ "github.com/luxfi/cli/tests/e2e/testcases/network"
	_ "github.com/luxfi/cli/tests/e2e/testcases/packageman"
	_ "github.com/luxfi/cli/tests/e2e/testcases/root"
	_ "github.com/luxfi/cli/tests/e2e/testcases/subnet"
	_ "github.com/luxfi/cli/tests/e2e/testcases/subnet/local"
	_ "github.com/luxfi/cli/tests/e2e/testcases/subnet/public"
	_ "github.com/luxfi/cli/tests/e2e/testcases/upgrade"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

func TestE2e(t *testing.T) {
	if os.Getenv("RUN_E2E") == "" {
		t.Skip("Environment variable RUN_E2E not set; skipping E2E tests")
	}
	gomega.RegisterFailHandler(ginkgo.Fail)
	format.UseStringerRepresentation = true
	ginkgo.RunSpecs(t, "cli e2e test suites")
}

var _ = ginkgo.BeforeSuite(func() {
	cmd := exec.Command("./scripts/build.sh")
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	gomega.Expect(err).Should(gomega.BeNil())
})
