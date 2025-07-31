// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/remoteconfig"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
)

// ValidateComposeFile validates a docker-compose file on a remote host.
func ValidateComposeFile(host *models.Host, composeFile string, timeout time.Duration) error {
	if output, err := host.Command(fmt.Sprintf("docker compose -f %s config", composeFile), nil, timeout); err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// ComposeSSHSetupNode sets up an LuxGo node and dependencies on a remote host over SSH.
func ComposeSSHSetupNode(
	host *models.Host,
	network models.Network,
	luxGoVersion string,
	luxdBootstrapIDs []string,
	luxdBootstrapIPs []string,
	partialSync bool,
	luxdGenesisFilePath string,
	luxdUpgradeFilePath string,
	withMonitoring bool,
	publicAccessToHTTPPort bool,
) error {
	startTime := time.Now()
	folderStructure := remoteconfig.RemoteFoldersToCreateLuxgo()
	for _, dir := range folderStructure {
		if err := host.MkdirAll(dir, constants.SSHFileOpsTimeout); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	ux.Logger.Info("luxCLI folder structure created on remote host %s after %s ", folderStructure, time.Since(startTime))

	avagoDockerImage := fmt.Sprintf("%s:%s", constants.LuxGoDockerImage, luxGoVersion)
	ux.Logger.Info("Preparing LuxGo Docker image %s on %s[%s]", avagoDockerImage, host.NodeID, host.IP)
	if err := PrepareDockerImageWithRepo(host, avagoDockerImage, constants.LuxGoGitRepo, luxGoVersion); err != nil {
		return err
	}
	ux.Logger.Info("LuxGo Docker image %s ready on %s[%s] after %s", avagoDockerImage, host.NodeID, host.IP, time.Since(startTime))
	nodeConfFile, cChainConfFile, err := prepareLuxgoConfig(
		host,
		network,
		LuxGoConfigOptions{
			BootstrapIDs:      luxdBootstrapIDs,
			BootstrapIPs:      luxdBootstrapIPs,
			PartialSync:       partialSync,
			GenesisPath:       luxdGenesisFilePath,
			UpgradePath:       luxdUpgradeFilePath,
			AllowPublicAccess: publicAccessToHTTPPort,
		},
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(nodeConfFile); err != nil {
			ux.Logger.Error("Error removing temporary file %s: %s", nodeConfFile, err)
		}
		if err := os.Remove(cChainConfFile); err != nil {
			ux.Logger.Error("Error removing temporary file %s: %s", cChainConfFile, err)
		}
	}()

	if err := host.Upload(nodeConfFile, remoteconfig.GetRemoteLuxNodeConfig(), constants.SSHFileOpsTimeout); err != nil {
		return err
	}
	if err := host.Upload(cChainConfFile, remoteconfig.GetRemoteLuxCChainConfig(), constants.SSHFileOpsTimeout); err != nil {
		return err
	}
	if luxdGenesisFilePath != "" {
		if err := host.Upload(luxdGenesisFilePath, remoteconfig.GetRemoteLuxGenesis(), constants.SSHFileOpsTimeout); err != nil {
			return err
		}
	}
	if luxdUpgradeFilePath != "" {
		if err := host.Upload(luxdUpgradeFilePath, remoteconfig.GetRemoteLuxUpgrade(), constants.SSHFileOpsTimeout); err != nil {
			return err
		}
	}
	ux.Logger.Info("LuxGo configs uploaded to %s[%s] after %s", host.NodeID, host.IP, time.Since(startTime))
	return ComposeOverSSH("Compose Node",
		host,
		constants.SSHScriptTimeout,
		"templates/luxd.docker-compose.yml",
		DockerComposeInputs{
			LuxgoVersion: luxGoVersion,
			WithMonitoring:     withMonitoring,
			WithLuxgo:    true,
			E2E:                utils.IsE2E(),
			E2EIP:              utils.E2EConvertIP(host.IP),
			E2ESuffix:          utils.E2ESuffix(host.IP),
		})
}

func ComposeSSHSetupLoadTest(host *models.Host) error {
	return ComposeOverSSH("Compose Node",
		host,
		constants.SSHScriptTimeout,
		"templates/luxd.docker-compose.yml",
		DockerComposeInputs{
			WithMonitoring:  true,
			WithLuxgo: false,
		})
}

// WasNodeSetupWithMonitoring checks if an LuxGo node was setup with monitoring on a remote host.
func WasNodeSetupWithMonitoring(host *models.Host) (bool, error) {
	return HasRemoteComposeService(host, utils.GetRemoteComposeFile(), "promtail", constants.SSHScriptTimeout)
}

// ComposeSSHSetupMonitoring sets up monitoring using docker-compose.
func ComposeSSHSetupMonitoring(host *models.Host) error {
	grafanaConfigFile, grafanaDashboardsFile, grafanaLokiDatasourceFile, grafanaPromDatasourceFile, err := prepareGrafanaConfig()
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(grafanaLokiDatasourceFile); err != nil {
			ux.Logger.Error("Error removing temporary file %s: %s", grafanaLokiDatasourceFile, err)
		}
		if err := os.Remove(grafanaPromDatasourceFile); err != nil {
			ux.Logger.Error("Error removing temporary file %s: %s", grafanaPromDatasourceFile, err)
		}
		if err := os.Remove(grafanaDashboardsFile); err != nil {
			ux.Logger.Error("Error removing temporary file %s: %s", grafanaDashboardsFile, err)
		}
		if err := os.Remove(grafanaConfigFile); err != nil {
			ux.Logger.Error("Error removing temporary file %s: %s", grafanaConfigFile, err)
		}
	}()

	grafanaLokiDatasourceRemoteFileName := filepath.Join(utils.GetRemoteComposeServicePath("grafana", "provisioning", "datasources"), "loki.yml")
	if err := host.Upload(grafanaLokiDatasourceFile, grafanaLokiDatasourceRemoteFileName, constants.SSHFileOpsTimeout); err != nil {
		return err
	}
	grafanaPromDatasourceFileName := filepath.Join(utils.GetRemoteComposeServicePath("grafana", "provisioning", "datasources"), "prometheus.yml")
	if err := host.Upload(grafanaPromDatasourceFile, grafanaPromDatasourceFileName, constants.SSHFileOpsTimeout); err != nil {
		return err
	}
	grafanaDashboardsRemoteFileName := filepath.Join(utils.GetRemoteComposeServicePath("grafana", "provisioning", "dashboards"), "dashboards.yml")
	if err := host.Upload(grafanaDashboardsFile, grafanaDashboardsRemoteFileName, constants.SSHFileOpsTimeout); err != nil {
		return err
	}
	grafanaConfigRemoteFileName := filepath.Join(utils.GetRemoteComposeServicePath("grafana"), "grafana.ini")
	if err := host.Upload(grafanaConfigFile, grafanaConfigRemoteFileName, constants.SSHFileOpsTimeout); err != nil {
		return err
	}

	return ComposeOverSSH("Setup Monitoring",
		host,
		constants.SSHScriptTimeout,
		"templates/monitoring.docker-compose.yml",
		DockerComposeInputs{})
}

func ComposeSSHSetupICMRelayer(host *models.Host, relayerVersion string) error {
	return ComposeOverSSH("Setup AWM Relayer",
		host,
		constants.SSHScriptTimeout,
		"templates/awmrelayer.docker-compose.yml",
		DockerComposeInputs{
			ICMRelayerVersion: relayerVersion,
		})
}
