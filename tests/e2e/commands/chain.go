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

	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	"github.com/onsi/gomega"
)

/* #nosec G204 */
func CreateEVMConfig(chainName string, genesisPath string) (string, string) {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())
	// let's use a EVM version which has a guaranteed compatible lux
	CreateEVMConfigWithVersion(chainName, genesisPath, mapping[utils.LatestEVM2LuxKey])
	return mapping[utils.LatestEVM2LuxKey], mapping[utils.LatestLux2EVMKey]
}

/* #nosec G204 */
func CreateEVMConfigWithVersion(chainName string, genesisPath string, version string) {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmdArgs := []string{ChainCmd, "create", "--genesis", genesisPath, "--evm", chainName, "--" + constants.SkipUpdateFlag}
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
	exists, err = utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

func ConfigureChainConfig(chainName string, genesisPath string) {
	// run configure
	cmdArgs := []string{ChainCmd, "configure", chainName, "--chain-config", genesisPath, "--" + constants.SkipUpdateFlag}
	cmd := exec.Command(CLIBinary, cmdArgs...) //nolint:gosec // G204: Running our own CLI binary in tests
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

func ConfigurePerNodeChainConfig(chainName string, perNodeChainConfigPath string) {
	// run configure
	cmdArgs := []string{ChainCmd, "configure", chainName, "--per-node-chain-config", perNodeChainConfigPath, "--" + constants.SkipUpdateFlag}
	cmd := exec.Command(CLIBinary, cmdArgs...) //nolint:gosec // G204: Running our own CLI binary in tests
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should now exist
	exists, err := utils.PerNodeChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func CreateCustomVMConfig(chainName string, genesisPath string, vmPath string) {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.ChainCustomVMExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"create",
		"--genesis",
		genesisPath,
		"--vm",
		vmPath,
		"--custom",
		chainName,
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
	exists, err = utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.ChainCustomVMExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

func DeleteChainConfig(chainName string) {
	// Config should exist
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Now delete config
	cmd := exec.Command(CLIBinary, ChainCmd, "delete", chainName, "--"+constants.SkipUpdateFlag) //nolint:gosec // G204: Running our own CLI binary in tests
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	gomega.Expect(err).Should(gomega.BeNil())

	// Config should no longer exist
	exists, err = utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
}

func DeleteElasticChainConfig(chainName string) {
	var err error
	elasticChainConfig := filepath.Join(utils.GetBaseDir(), constants.ChainsDir, chainName, constants.ElasticChainConfigFileName)
	if _, err = os.Stat(elasticChainConfig); errors.Is(err, os.ErrNotExist) {
		// does *not* exist
		err = nil
	} else {
		err = os.Remove(elasticChainConfig)
	}
	gomega.Expect(err).Should(gomega.BeNil())
}

// Returns the deploy output
/* #nosec G204 */
func DeployChainLocally(chainName string) string {
	return DeployChainLocallyWithArgs(chainName, "", "")
}

/* #nosec G204 */
func DeployChainLocallyExpectError(chainName string) {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())

	DeployChainLocallyWithArgsExpectError(chainName, mapping[utils.OnlyLuxKey], "")
}

// Returns the deploy output
/* #nosec G204 */
func DeployChainLocallyWithViperConf(chainName string, confPath string) string {
	mapper := utils.NewVersionMapper()
	mapping, err := utils.GetVersionMapping(mapper)
	gomega.Expect(err).Should(gomega.BeNil())

	return DeployChainLocallyWithArgs(chainName, mapping[utils.OnlyLuxKey], confPath)
}

// Returns the deploy output
/* #nosec G204 */
func DeployChainLocallyWithVersion(chainName string, version string) string {
	return DeployChainLocallyWithArgs(chainName, version, "")
}

// Returns the deploy output
/* #nosec G204 */
func DeployChainLocallyWithArgs(chainName string, version string, confPath string) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy chain locally
	cmdArgs := []string{ChainCmd, "deploy", "--local", chainName, "--" + constants.SkipUpdateFlag}
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

func DeployChainLocallyWithArgsAndOutput(chainName string, version string, confPath string) ([]byte, error) {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Deploy chain locally
	cmdArgs := []string{ChainCmd, "deploy", "--local", chainName, "--" + constants.SkipUpdateFlag}
	if version != "" {
		cmdArgs = append(cmdArgs, "--node-version", version)
	}
	if confPath != "" {
		cmdArgs = append(cmdArgs, "--config", confPath)
	}
	cmd := exec.Command(CLIBinary, cmdArgs...) //nolint:gosec // G204: Running our own CLI binary in tests
	return cmd.CombinedOutput()
}

func DeployChainLocallyWithArgsExpectError(chainName string, version string, confPath string) {
	_, err := DeployChainLocallyWithArgsAndOutput(chainName, version, confPath)
	gomega.Expect(err).Should(gomega.HaveOccurred())
}

// simulates testnet deploy execution path on a local network
/* #nosec G204 */
func SimulateTestnetDeploy(
	chainName string,
	key string,
	controlKeys string,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	// Deploy chain locally
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"deploy",
		"--testnet",
		"--threshold",
		"1",
		"--key",
		key,
		"--control-keys",
		controlKeys,
		chainName,
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
	chainName string,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	// Deploy chain locally
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"deploy",
		"--mainnet",
		"--threshold",
		"1",
		"--same-control-key",
		chainName,
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
	chainName string,
	key string,
	nodeID string,
	start string,
	period string,
	weight string,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
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
		chainName,
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
	chainName string,
	key string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"removeValidator",
		"--testnet",
		"--key",
		key,
		"--nodeID",
		nodeID,
		chainName,
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

func SimulateTestnetTransformChain(
	chainName string,
	key string,
) (string, error) {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
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
		chainName,
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
	chainName string,
	nodeID string,
	start string,
	period string,
	weight string,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
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
		chainName,
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
	chainName string,
	nodeConfig string,
	pluginDir string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
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
		chainName,
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
	chainName string,
	nodeConfig string,
	pluginDir string,
	nodeID string,
) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// enable simulation of public network execution paths on a local network
	err = os.Setenv(constants.SimulatePublicNetwork, "true")
	gomega.Expect(err).Should(gomega.BeNil())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
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
		chainName,
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
func ImportChainConfig(repoAlias string, chainName string) {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.ChainCustomVMExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"import",
		"file",
		"--repo",
		repoAlias,
		"--chain",
		chainName,
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
	exists, err = utils.LPMConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.ChainLPMVMExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func ImportChainConfigFromURL(repoURL string, branch string, chainName string) {
	// Check config does not already exist
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())
	// Check vm binary does not already exist
	exists, err = utils.ChainCustomVMExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeFalse())

	// Create config
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"import",
		"file",
		"--repo",
		repoURL,
		"--branch",
		branch,
		"--chain",
		chainName,
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
	exists, err = utils.LPMConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
	exists, err = utils.ChainLPMVMExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())
}

/* #nosec G204 */
func DescribeChain(chainName string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"describe",
		chainName,
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
func SimulateGetChainStatsTestnet(chainName, chainID string) string {
	// Check config does already exist:
	// We want to run stats on an existing chain
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// add the chain ID to the `testnet` section so that the `stats` command
	// can find it (as this is a simulation with a `local` network,
	// it got written in to the `local` network section)
	err = utils.AddChainIDToSidecar(chainName, models.Testnet, chainID)
	gomega.Expect(err).Should(gomega.BeNil())
	// run stats
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"stats",
		chainName,
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

func TransformElasticChainLocally(chainName string) (string, error) {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		ElasticTransformCmd,
		"--local",
		"--tokenName",
		"BLIZZARD",
		"--tokenSymbol",
		"BRRR",
		"--default",
		"--force",
		chainName,
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

func TransformElasticChainLocallyandTransformValidators(chainName string, stakeAmount string) (string, error) {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
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
		chainName,
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

func RemoveValidator(chainName string, nodeID string) (string, error) {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		RemoveValidatorCmd,
		"--local",
		"--nodeID",
		nodeID,
		chainName,
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

func AddPermissionlessValidator(chainName string, nodeID string, stakeAmount string, stakingPeriod string) (string, error) {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	startTimeStr := time.Now().Add(constants.StakingStartLeadTime).UTC().Format(constants.TimeParseLayout)
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		ChainCmd,
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
		chainName,
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
func ListValidators(chainName string, network string) (string, error) {
	// Create config
	cmd := exec.Command(
		CLIBinary,
		ChainCmd,
		"validators",
		chainName,
		"--"+network,
		"--"+constants.SkipUpdateFlag,
	)

	out, err := cmd.Output()
	return string(out), err
}

// CreateEVMConfigNonSOV creates a non-sovereign chain EVM config
/* #nosec G204 */
func CreateEVMConfigNonSOV(chainName string, genesisPath string, _ bool) (string, string) {
	// For now, just call the regular CreateEVMConfig
	// The bootstrapped parameter is ignored in the base implementation
	return CreateEVMConfig(chainName, genesisPath)
}

// CreateEVMConfigSOV creates a sovereign chain EVM config
/* #nosec G204 */
func CreateEVMConfigSOV(chainName string, genesisPath string) (string, string) {
	// For now, just call the regular CreateEVMConfig
	// SOV-specific functionality would be added here
	return CreateEVMConfig(chainName, genesisPath)
}

// CreateCustomVMConfigNonSOV creates a non-sovereign custom VM config
/* #nosec G204 */
func CreateCustomVMConfigNonSOV(chainName string, genesisPath string, vmPath string) {
	// For now, just call the regular CreateCustomVMConfig
	CreateCustomVMConfig(chainName, genesisPath, vmPath)
}

// CreateCustomVMConfigSOV creates a sovereign custom VM config
/* #nosec G204 */
func CreateCustomVMConfigSOV(chainName string, genesisPath string, vmPath string) {
	// For now, just call the regular CreateCustomVMConfig
	// SOV-specific functionality would be added here
	CreateCustomVMConfig(chainName, genesisPath, vmPath)
}

// DeployChainLocallyNonSOV deploys a non-sovereign chain locally
/* #nosec G204 */
func DeployChainLocallyNonSOV(chainName string) string {
	// For now, just call the regular DeployChainLocally
	return DeployChainLocally(chainName)
}

// DeployChainLocallyWithVersionNonSOV deploys a non-sovereign chain locally with specific version
/* #nosec G204 */
func DeployChainLocallyWithVersionNonSOV(chainName string, version string) string {
	// Call the existing function with version
	return DeployChainLocallyWithVersion(chainName, version)
}

// DeployChainLocallyWithViperConfNonSOV deploys a non-sovereign chain locally with viper config
/* #nosec G204 */
func DeployChainLocallyWithViperConfNonSOV(chainName string, confPath string) string {
	// Call the existing function with config path
	return DeployChainLocallyWithViperConf(chainName, confPath)
}

// SimulateTestnetDeploySOV simulates sovereign chain deployment on testnet
/* #nosec G204 */
func SimulateTestnetDeploySOV(chainName string, key string, controlKeys string) string {
	// For now, just call the regular SimulateTestnetDeploy
	// SOV-specific functionality would be added here
	return SimulateTestnetDeploy(chainName, key, controlKeys)
}

// DeployChainLocallySOV deploys a sovereign chain locally
/* #nosec G204 */
func DeployChainLocallySOV(chainName string) string {
	// For now, just call the regular DeployChainLocally
	// SOV-specific functionality would be added here
	return DeployChainLocally(chainName)
}

// DeployChainLocallyWithViperConfSOV deploys a sovereign chain locally with viper config
/* #nosec G204 */
func DeployChainLocallyWithViperConfSOV(chainName string, confPath string) string {
	// Call the existing function with config path
	// SOV-specific functionality would be added here
	return DeployChainLocallyWithViperConf(chainName, confPath)
}

// DeployChainLocallyWithVersionSOV deploys a sovereign chain locally with specific version
/* #nosec G204 */
func DeployChainLocallyWithVersionSOV(chainName string, version string) string {
	// Call the existing function with version
	// SOV-specific functionality would be added here
	return DeployChainLocallyWithVersion(chainName, version)
}

// DeployChainLocallyWithArgsAndOutputSOV deploys a sovereign chain locally and returns output
/* #nosec G204 */
func DeployChainLocallyWithArgsAndOutputSOV(chainName string, version string, confPath string) ([]byte, error) {
	// Call the existing function
	// SOV-specific functionality would be added here
	return DeployChainLocallyWithArgsAndOutput(chainName, version, confPath)
}

// CreateEVMConfigWithVersionSOV creates a sovereign chain EVM config with specific version
/* #nosec G204 */
func CreateEVMConfigWithVersionSOV(chainName string, genesisPath string, version string) {
	// Call the existing function with version
	// SOV-specific functionality would be added here
	CreateEVMConfigWithVersion(chainName, genesisPath, version)
}

// DeployChainLocallyExpectErrorSOV deploys a sovereign chain locally expecting an error
/* #nosec G204 */
func DeployChainLocallyExpectErrorSOV(chainName string) {
	// Call the existing function
	// SOV-specific functionality would be added here
	DeployChainLocallyExpectError(chainName)
}

// DeployChainLocallyWithArgsAndOutputNonSOV deploys a non-sovereign chain locally and returns output
/* #nosec G204 */
func DeployChainLocallyWithArgsAndOutputNonSOV(chainName string, version string, confPath string) ([]byte, error) {
	// Call the existing function
	return DeployChainLocallyWithArgsAndOutput(chainName, version, confPath)
}

// CreateEVMConfigWithVersionNonSOV creates a non-sovereign chain EVM config with specific version
/* #nosec G204 */
func CreateEVMConfigWithVersionNonSOV(chainName string, genesisPath string, version string, _ bool) {
	// Call the existing function with version
	// The bootstrapped parameter is ignored in the base implementation for now
	CreateEVMConfigWithVersion(chainName, genesisPath, version)
}

// DeployChainLocallyExpectErrorNonSOV deploys a non-sovereign chain locally expecting an error
/* #nosec G204 */
func DeployChainLocallyExpectErrorNonSOV(chainName string) {
	// Call the existing function
	DeployChainLocallyExpectError(chainName)
}

// SimulateTestnetDeployNonSOV simulates non-sovereign chain deployment on testnet
/* #nosec G204 */
func SimulateTestnetDeployNonSOV(chainName string, key string, controlKeys string) string {
	// For now, just call the regular SimulateTestnetDeploy
	return SimulateTestnetDeploy(chainName, key, controlKeys)
}

// SimulateMainnetDeployNonSOV simulates non-sovereign chain deployment on mainnet
// Updated to accept chainID and skipPrompt parameters to match usage
/* #nosec G204 */
func SimulateMainnetDeployNonSOV(chainName string, chainID int, skipPrompt bool) string {
	// Check config exists
	exists, err := utils.ChainConfigExists(chainName)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(exists).Should(gomega.BeTrue())

	// Build command args
	cmdArgs := []string{
		ChainCmd,
		"deploy",
		chainName,
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
