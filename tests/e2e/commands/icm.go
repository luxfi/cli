// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package commands

import (
	"github.com/luxfi/cli/tests/e2e/utils"
)

const (
	WarpCmd = "warp"
)

/* #nosec G204 */
func SendWarpMessage(args []string, testFlags utils.TestFlags) (string, error) {
	return utils.TestCommand(WarpCmd, "sendMsg", args, utils.GlobalFlags{
		Network: "local",
	}, testFlags)
}

/* #nosec G204 */
func DeployWarpContracts(args []string, testFlags utils.TestFlags) (string, error) {
	return utils.TestCommand(WarpCmd, "deploy", args, utils.GlobalFlags{
		Network: "local",
	}, testFlags)
}
