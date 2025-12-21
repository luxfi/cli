// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/sdk/models"
	"github.com/onsi/gomega"
)

/* #nosec G204 */
func CreateEVMConfig(subnetName string, genesisPath string) (string, string) {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())
	// let's use a EVM version which has a guaranteed compatible lux
	CreateEVMConfigWithVersion(subnetName, genesisPath, mapping[utils.LatestEVM2LuxKey])
	return mapping[utils.LatestEVM2LuxKey], mapping[utils.LatestLux2EVMKey]
}

/* #nosec G204 */
func CreateEVMConfigWithVersion(subnetName string, genesisPath string, version string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmdArgs := []string{SubnetCmd, "create", "--genesis", genesisPath, "--evm", subnetName, "--" + constants.SkipUpdateFlag}
	if version == "" {
		cmdArgs = append(cmdArgs, "--latest")
	} else {
		cmdArgs = append(cmdArgs, "--vm-version", version)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err = utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func ConfigureChainConfig(subnetName string, genesisPath string) {
	// run configure
	cmdArgs := []string{SubnetCmd, "configure", subnetName, "--chain-config", genesisPath, "--" + constants.SkipUpdateFlag}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err := utils.ChainConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func ConfigurePerNodeChainConfig(subnetName string, perNodeChainConfigPath string) {
	// run configure
	cmdArgs := []string{SubnetCmd, "configure", subnetName, "--per-node-chain-config", perNodeChainConfigPath, "--" + constants.SkipUpdateFlag}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err := utils.PerNodeChainConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func CreateCustomVMConfig(subnetName string, genesisPath string, vmPath string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"create",
		"--genesis",
		genesisPath,
		"--vm",
		vmPath,
		"--custom",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}

	// Config should now exist
	exists, err = utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func DeleteSubnetConfig(subnetName string) {
	// Config should exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Now delete config
	cmd := exec.Command(CLIBinary, SubnetCmd, "delete", subnetName, "--"+constants.SkipUpdateFlag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should no longer exist
	exists, err = utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
}

func DeleteElasticSubnetConfig(subnetName string) {
	var err error
	elasticSubnetConfig := filepath.Join(utils.GetBaseDir(), constants.ChainsDir, subnetName, constants.ElasticSubnetConfigFileName)
	if _, err = os.Stat(elasticSubnetConfig); errors.Is(err, os.ErrNotExist) {
		// does *not* exist
		err = nil
	} else {
		err = os.Remove(elasticSubnetConfig)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocally(subnetName string) string {
	return DeploySubnetLocallyWithArgs(subnetName, "", "")
}

/* #nosec G204 */
func DeploySubnetLocallyExpectError(subnetName string) {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())

	DeploySubnetLocallyWithArgsExpectError(subnetName, mapping[utils.OnlyLuxKey], "")
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocallyWithViperConf(subnetName string, confPath string) string {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())

	return DeploySubnetLocallyWithArgs(subnetName, mapping[utils.OnlyLuxKey], confPath)
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocallyWithVersion(subnetName string, version string) string {
	return DeploySubnetLocallyWithArgs(subnetName, version, "")
}

// Returns the deploy output
/* #nosec G204 */
func DeploySubnetLocallyWithArgs(subnetName string, version string, confPath string) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy subnet locally
	cmdArgs := []string{SubnetCmd, "deploy", "--local", subnetName, "--" + constants.SkipUpdateFlag}
	if version != "" {
		cmdArgs = append(cmdArgs, "--node-version", version)
	}
	if confPath != "" {
		cmdArgs = append(cmdArgs, "--config", confPath)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

func DeploySubnetLocallyWithArgsAndOutput(subnetName string, version string, confPath string) ([]byte, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy subnet locally
	cmdArgs := []string{SubnetCmd, "deploy", "--local", subnetName, "--" + constants.SkipUpdateFlag}
	if version != "" {
		cmdArgs = append(cmdArgs, "--node-version", version)
	}
	if confPath != "" {
		cmdArgs = append(cmdArgs, "--config", confPath)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...)
	return cmd.CombinedOutput()
}

/* #nosec G204 */
func DeploySubnetLocallyWithArgsExpectError(subnetName string, version string, confPath string) {
	_, err := DeploySubnetLocallyWithArgsAndOutput(subnetName, version, confPath)
	gomega.Expect(err).Should(gomega.HaveOccurred())
}

// simulates testnet deploy execution path on a local network
/* #nosec G204 */
func SimulateTestnetDeploy(
	subnetName string,
	key string,
	controlKeys string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	// Deploy subnet locally
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"deploy",
		"--testnet",
		"--threshold",
		"1",
		"--key",
		key,
		"--control-keys",
		controlKeys,
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

// simulates mainnet deploy execution path on a local network
/* #nosec G204 */
func SimulateMainnetDeploy(
	subnetName string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	// Deploy subnet locally
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"deploy",
		"--mainnet",
		"--threshold",
		"1",
		"--same-control-key",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	stdoutPipe, err := cmd.StdoutPipe()
	gomega.Expect(err).Should(gomega.BeNil())
	stderrPipe, err := cmd.StderrPipe()
	gomega.Expect(err).Should(gomega.BeNil())
	err = cmd.Start()
	gomega.Expect(err).Should(gomega.BeNil())

	stdout := ""
	go func(p io.ReadCloser) {
		reader := bufio.NewReader(p)
		line, err := reader.ReadString('\n')
		for err == nil {
			stdout += line
			fmt.Print(line)
			line, err = reader.ReadString('\n')
		}
	}(stdoutPipe)

	stderr, err := io.ReadAll(stderrPipe)
	gomega.Expect(err).Should(gomega.BeNil())
	fmt.Println(string(stderr))

	err = cmd.Wait()
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return stdout + string(stderr)
}

// simulates testnet add validator execution path on a local network
/* #nosec G204 */
func SimulateTestnetAddValidator(
	subnetName string,
	key string,
	nodeID string,
	start string,
	period string,
	weight string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"addValidator",
		"--testnet",
		"--key",
		key,
		"--nodeID",
		nodeID,
		"--start-time",
		start,
		"--staking-period",
		period,
		"--weight",
		weight,
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

// simulates testnet add validator execution path on a local network
func SimulateTestnetRemoveValidator(
	subnetName string,
	key string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"removeValidator",
		"--testnet",
		"--key",
		key,
		"--nodeID",
		nodeID,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

func SimulateTestnetTransformSubnet(
	subnetName string,
	key string,
) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		ElasticTransformCmd,
		"--testnet",
		"--key",
		key,
		"--tokenName",
		"BLIZZARD",
		"--tokenSymbol",
		"BRRR",
		"--denomination",
		"0",
		"--default",
		"--force",
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		err2 := os.Unsetenv(constants.SimulatePublicNetwork)
		gomega.Expect(err2).Should(gomega.BeNil())
		return "", err
	}

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output), nil
}

// simulates mainnet add validator execution path on a local network
/* #nosec G204 */
func SimulateMainnetAddValidator(
	subnetName string,
	nodeID string,
	start string,
	period string,
	weight string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"addValidator",
		"--mainnet",
		"--nodeID",
		nodeID,
		"--start-time",
		start,
		"--staking-period",
		period,
		"--weight",
		weight,
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	stdoutPipe, err := cmd.StdoutPipe()
	gomega.Expect(err).Should(gomega.BeNil())
	stderrPipe, err := cmd.StderrPipe()
	gomega.Expect(err).Should(gomega.BeNil())
	err = cmd.Start()
	gomega.Expect(err).Should(gomega.BeNil())

	stdout := ""
	go func(p io.ReadCloser) {
		reader := bufio.NewReader(p)
		line, err := reader.ReadString('\n')
		for err == nil {
			stdout += line
			fmt.Print(line)
			line, err = reader.ReadString('\n')
		}
	}(stdoutPipe)

	stderr, err := io.ReadAll(stderrPipe)
	gomega.Expect(err).Should(gomega.BeNil())
	fmt.Println(string(stderr))

	err = cmd.Wait()
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return stdout + string(stderr)
}

// simulates testnet join execution path on a local network
/* #nosec G204 */
func SimulateTestnetJoin(
	subnetName string,
	nodeConfig string,
	pluginDir string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"join",
		"--testnet",
		"--node-config",
		nodeConfig,
		"--plugin-dir",
		pluginDir,
		"--force-whitelist-check",
		"--fail-if-not-validating",
		"--nodeID",
		nodeID,
		"--force-write",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

// simulates mainnet join execution path on a local network
/* #nosec G204 */
func SimulateMainnetJoin(
	subnetName string,
	nodeConfig string,
	pluginDir string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"join",
		"--mainnet",
		"--node-config",
		nodeConfig,
		"--plugin-dir",
		pluginDir,
		"--force-whitelist-check",
		"--fail-if-not-validating",
		"--nodeID",
		nodeID,
		"--force-write",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// disable simulation of public network execution paths on a local network
	err = os.Unsetenv(constants.SimulatePublicNetwork)
	gomega.Expect(err).Should(gomega.BeNil())

	return string(output)
}

/* #nosec G204 */
func ImportSubnetConfig(repoAlias string, subnetName string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"import",
		"file",
		"--repo",
		repoAlias,
		"--subnet",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		fmt.Println(err)
		fmt.Println(stderr)
	}

	// Config should now exist
	exists, err = utils.LPMConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.SubnetLPMVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func ImportSubnetConfigFromURL(repoURL string, branch string, subnetName string) {
	// Check config does not already exist
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.SubnetCustomVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"import",
		"file",
		"--repo",
		repoURL,
		"--branch",
		branch,
		"--subnet",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var (
			exitErr *exec.ExitError
			stderr  string
		)
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}

	// Config should now exist
	exists, err = utils.LPMConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.SubnetLPMVMExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func DescribeSubnet(subnetName string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"describe",
		subnetName,
		"--"+constants.SkipUpdateFlag,
	)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

/* #nosec G204 */
func SimulateGetSubnetStatsTestnet(subnetName, subnetID string) string {
	// Check config does already exist:
	// We want to run stats on an existing subnet
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// add the subnet ID to the `testnet` section so that the `stats` command
	// can find it (as this is a simulation with a `local` network,
	// it got written in to the `local` network section)
	err = utils.AddSubnetIDToSidecar(subnetName, models.Testnet, subnetID)
	gomega.Expect(err).Should(gomega.BeNil())
	// run stats
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"stats",
		subnetName,
		"--testnet",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if err != nil {
		stderr := ""
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		fmt.Println(string(output))
		fmt.Println(err)
		fmt.Println(stderr)
	}
	gomega.Expect(exitErr).Should(gomega.BeNil())
	return string(output)
}

func TransformElasticSubnetLocally(subnetName string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		ElasticTransformCmd,
		"--local",
		"--tokenName",
		"BLIZZARD",
		"--tokenSymbol",
		"BRRR",
		"--default",
		"--force",
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

func TransformElasticSubnetLocallyandTransformValidators(subnetName string, stakeAmount string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		ElasticTransformCmd,
		"--local",
		"--tokenName",
		"BLIZZARD",
		"--tokenSymbol",
		"BRRR",
		"--default",
		"--force",
		"--transform-validators",
		"--stake-amount",
		stakeAmount,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

func RemoveValidator(subnetName string, nodeID string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		RemoveValidatorCmd,
		"--local",
		"--nodeID",
		nodeID,
		subnetName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

func AddPermissionlessValidator(subnetName string, nodeID string, stakeAmount string, stakingPeriod string) (string, error) {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	startTimeStr := time.Now().Add(constants.StakingStartLeadTime).UTC().Format(constants.TimeParseLayout)
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		JoinCmd,
		"--local",
		"--elastic",
		"--nodeID",
		nodeID,
		"--stake-amount",
		stakeAmount,
		"--start-time",
		startTimeStr,
		"--staking-period",
		stakingPeriod,
		subnetName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		var stderr string
		fmt.Println(string(output))
		utils.PrintStdErr(err)
		fmt.Println(stderr)
	}
	return string(output), err
}

/* #nosec G204 */
func ListValidators(subnetName string, network string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		"validators",
		subnetName,
		"--"+network,
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.Output()
	return string(out), err
}

// CreateEVMConfigNonSOV creates a non-sovereign subnet EVM config
/* #nosec G204 */
func CreateEVMConfigNonSOV(subnetName string, genesisPath string, bootstrapped bool) (string, string) {
	// For now, just call the regular CreateEVMConfig
	// The bootstrapped parameter is ignored in the base implementation
	return CreateEVMConfig(subnetName, genesisPath)
}

// CreateEVMConfigSOV creates a sovereign subnet EVM config
/* #nosec G204 */
func CreateEVMConfigSOV(subnetName string, genesisPath string) (string, string) {
	// For now, just call the regular CreateEVMConfig
	// SOV-specific functionality would be added here
	return CreateEVMConfig(subnetName, genesisPath)
}

// CreateCustomVMConfigNonSOV creates a non-sovereign custom VM config
/* #nosec G204 */
func CreateCustomVMConfigNonSOV(subnetName string, genesisPath string, vmPath string) {
	// For now, just call the regular CreateCustomVMConfig
	CreateCustomVMConfig(subnetName, genesisPath, vmPath)
}

// CreateCustomVMConfigSOV creates a sovereign custom VM config
/* #nosec G204 */
func CreateCustomVMConfigSOV(subnetName string, genesisPath string, vmPath string) {
	// For now, just call the regular CreateCustomVMConfig
	// SOV-specific functionality would be added here
	CreateCustomVMConfig(subnetName, genesisPath, vmPath)
}

// DeploySubnetLocallyNonSOV deploys a non-sovereign subnet locally
/* #nosec G204 */
func DeploySubnetLocallyNonSOV(subnetName string) string {
	// For now, just call the regular DeploySubnetLocally
	return DeploySubnetLocally(subnetName)
}

// DeploySubnetLocallyWithVersionNonSOV deploys a non-sovereign subnet locally with specific version
/* #nosec G204 */
func DeploySubnetLocallyWithVersionNonSOV(subnetName string, version string) string {
	// Call the existing function with version
	return DeploySubnetLocallyWithVersion(subnetName, version)
}

// DeploySubnetLocallyWithViperConfNonSOV deploys a non-sovereign subnet locally with viper config
/* #nosec G204 */
func DeploySubnetLocallyWithViperConfNonSOV(subnetName string, confPath string) string {
	// Call the existing function with config path
	return DeploySubnetLocallyWithViperConf(subnetName, confPath)
}

// SimulateTestnetDeploySOV simulates sovereign subnet deployment on testnet
/* #nosec G204 */
func SimulateTestnetDeploySOV(subnetName string, key string, controlKeys string) string {
	// For now, just call the regular SimulateTestnetDeploy
	// SOV-specific functionality would be added here
	return SimulateTestnetDeploy(subnetName, key, controlKeys)
}

// DeploySubnetLocallySOV deploys a sovereign subnet locally
/* #nosec G204 */
func DeploySubnetLocallySOV(subnetName string) string {
	// For now, just call the regular DeploySubnetLocally
	// SOV-specific functionality would be added here
	return DeploySubnetLocally(subnetName)
}

// DeploySubnetLocallyWithViperConfSOV deploys a sovereign subnet locally with viper config
/* #nosec G204 */
func DeploySubnetLocallyWithViperConfSOV(subnetName string, confPath string) string {
	// Call the existing function with config path
	// SOV-specific functionality would be added here
	return DeploySubnetLocallyWithViperConf(subnetName, confPath)
}

// DeploySubnetLocallyWithVersionSOV deploys a sovereign subnet locally with specific version
/* #nosec G204 */
func DeploySubnetLocallyWithVersionSOV(subnetName string, version string) string {
	// Call the existing function with version
	// SOV-specific functionality would be added here
	return DeploySubnetLocallyWithVersion(subnetName, version)
}

// DeploySubnetLocallyWithArgsAndOutputSOV deploys a sovereign subnet locally and returns output
/* #nosec G204 */
func DeploySubnetLocallyWithArgsAndOutputSOV(subnetName string, version string, confPath string) ([]byte, error) {
	// Call the existing function
	// SOV-specific functionality would be added here
	return DeploySubnetLocallyWithArgsAndOutput(subnetName, version, confPath)
}

// CreateEVMConfigWithVersionSOV creates a sovereign subnet EVM config with specific version
/* #nosec G204 */
func CreateEVMConfigWithVersionSOV(subnetName string, genesisPath string, version string) {
	// Call the existing function with version
	// SOV-specific functionality would be added here
	CreateEVMConfigWithVersion(subnetName, genesisPath, version)
}

// DeploySubnetLocallyExpectErrorSOV deploys a sovereign subnet locally expecting an error
/* #nosec G204 */
func DeploySubnetLocallyExpectErrorSOV(subnetName string) {
	// Call the existing function
	// SOV-specific functionality would be added here
	DeploySubnetLocallyExpectError(subnetName)
}

// DeploySubnetLocallyWithArgsAndOutputNonSOV deploys a non-sovereign subnet locally and returns output
/* #nosec G204 */
func DeploySubnetLocallyWithArgsAndOutputNonSOV(subnetName string, version string, confPath string) ([]byte, error) {
	// Call the existing function
	return DeploySubnetLocallyWithArgsAndOutput(subnetName, version, confPath)
}

// CreateEVMConfigWithVersionNonSOV creates a non-sovereign subnet EVM config with specific version
/* #nosec G204 */
func CreateEVMConfigWithVersionNonSOV(subnetName string, genesisPath string, version string, bootstrapped bool) {
	// Call the existing function with version
	// The bootstrapped parameter is ignored in the base implementation for now
	CreateEVMConfigWithVersion(subnetName, genesisPath, version)
}

// DeploySubnetLocallyExpectErrorNonSOV deploys a non-sovereign subnet locally expecting an error
/* #nosec G204 */
func DeploySubnetLocallyExpectErrorNonSOV(subnetName string) {
	// Call the existing function
	DeploySubnetLocallyExpectError(subnetName)
}

// SimulateTestnetDeployNonSOV simulates non-sovereign subnet deployment on testnet
/* #nosec G204 */
func SimulateTestnetDeployNonSOV(subnetName string, key string, controlKeys string) string {
	// For now, just call the regular SimulateTestnetDeploy
	return SimulateTestnetDeploy(subnetName, key, controlKeys)
}

// SimulateMainnetDeployNonSOV simulates non-sovereign subnet deployment on mainnet
// Updated to accept chainID and skipPrompt parameters to match usage
/* #nosec G204 */
func SimulateMainnetDeployNonSOV(subnetName string, chainID int, skipPrompt bool) string {
	// Check config exists
	exists, err := utils.SubnetConfigExists(subnetName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		SubnetCmd,
		"deploy",
		subnetName,
		"--mainnet",
		"--" + constants.SkipUpdateFlag,
	}

	// Add chain ID if specified
	if chainID > 0 {
		cmdArgs = append(cmdArgs, "--chain-id", fmt.Sprintf("%d", chainID))
	}

	// Add skip prompt flag if specified
	if skipPrompt {
		cmdArgs = append(cmdArgs, "--yes")
	}

	// Execute command
	cmd := exec.Command(CLIBinary, cmdArgs...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(outputStr)
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	return outputStr
}
