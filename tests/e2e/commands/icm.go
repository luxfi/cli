// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package commands

import (
	"github.com/luxfi/cli/cmd"
	"github.com/luxfi/cli/tests/e2e/utils"
)

const (
	ICMCmd = "icm"
)

/* #nosec G204 */
func SendICMMessage(args []string, testFlags utils.TestFlags) (string, error) {
	return utils.TestCommand(ICMCmd, "sendMsg", args, utils.GlobalFlags{
		"local":             true,
		"skip-update-check": true,
	}, testFlags)
}

/* #nosec G204 */
func DeployICMContracts(args []string, testFlags utils.TestFlags) (string, error) {
	return utils.TestCommand(cmd.ICMCmd, "deploy", args, utils.GlobalFlags{
		"local":             true,
		"skip-update-check": true,
	}, testFlags)
}
