// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"os/exec"

	"github.com/luxfi/cli/pkg/constants"
)

/* #nosec G204 */
func CreateKey(keyName string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		KeyCmd,
		"create",
		keyName,
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.Output()
	return string(out), err
}

/* #nosec G204 */
func CreateKeyFromPath(keyName string, keyPath string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		KeyCmd,
		"create",
		"--file",
		keyPath,
		keyName,
		"--"+constants.SkipUpdateFlag,
	)
	out, err := cmd.Output()
	return string(out), err
}

/* #nosec G204 */
func CreateKeyForce(keyName string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		KeyCmd,
		"create",
		keyName,
		"--force",
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.Output()
	return string(out), err
}

/* #nosec G204 */
func ListKeys(network string, allBalances bool, chains string, tokens string) (string, error) {
	// Create config
	args := []string{
		CLIBinary,
		KeyCmd,
		"list",
	}

	// Add network flag (local, mainnet, testnet)
	switch network {
	case "local":
		args = append(args, "--local")
	case "testnet":
		args = append(args, "--testnet")
	default:
		args = append(args, "--mainnet")
	}

	// Add all-balances flag
	if allBalances {
		args = append(args, "--all-balances")
	}

	// Add chains flag if provided
	if chains != "" {
		args = append(args, "--chains", chains)
	}

	// Add tokens flag if provided
	if tokens != "" {
		args = append(args, "--tokens", tokens)
	}

	args = append(args, "--"+constants.SkipUpdateFlag)

	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.Output()
	return string(out), err
}

/* #nosec G204 */
func DeleteKey(keyName string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		KeyCmd,
		"delete",
		keyName,
		"--force",
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.Output()
	return string(out), err
}

/* #nosec G204 */
func ExportKey(keyName string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		KeyCmd,
		"export",
		keyName,
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.Output()
	return string(out), err
}

/* #nosec G204 */
func ExportKeyToFile(keyName string, outputPath string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		KeyCmd,
		"export",
		keyName,
		"-o",
		outputPath,
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.Output()
	return string(out), err
}

/* #nosec G204 */
func KeyTransferSend(args []string) (string, error) {
	// Build command args
	cmdArgs := []string{
		CLIBinary,
		KeyCmd,
		"transfer",
	}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "--"+constants.SkipUpdateFlag)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	out, err := cmd.Output()
	return string(out), err
}
