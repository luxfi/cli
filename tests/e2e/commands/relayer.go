// / Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"github.com/luxfi/cli/tests/e2e/utils"
)

/* #nosec G204 */
func StopRelayer() (string, error) {
	return utils.TestCommand(InterchainCMD, "relayer", []string{"stop"}, utils.GlobalFlags{
		"network": "local",
	}, utils.TestFlags{})
}

/* #nosec G204 */
func DeployRelayer(args []string, testFlags utils.TestFlags) (string, error) {
	return utils.TestCommand(InterchainCMD, "relayer", args, utils.GlobalFlags{
		"network": "local",
	}, testFlags)
}
