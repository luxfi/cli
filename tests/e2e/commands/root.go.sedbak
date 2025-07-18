// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"os/exec"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/onsi/gomega"
)

func GetVersion() string {
	/* #nosec G204 */
	cmd := exec.Command(
		CLIBinary,
		"--version",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.Output()
	gomega.Expect(err).Should(gomega.BeNil())
	return string(output)
}
