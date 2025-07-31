// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package docker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/remoteconfig"
	"github.com/luxfi/cli/pkg/utils"
)

type LuxGoConfigOptions struct {
	BootstrapIPs      []string
	BootstrapIDs      []string
	PartialSync       bool
	GenesisPath       string
	UpgradePath       string
	AllowPublicAccess bool
}

func prepareLuxgoConfig(
	host *models.Host,
	network models.Network,
	luxGoConfig LuxGoConfigOptions,
) (string, string, error) {
	luxdConf := remoteconfig.PrepareLuxConfig(host.IP, network.NetworkIDFlagValue(), nil)
	if luxGoConfig.AllowPublicAccess || utils.IsE2E() {
		luxdConf.HTTPHost = "0.0.0.0"
	}
	luxdConf.PartialSync = luxGoConfig.PartialSync
	luxdConf.BootstrapIPs = strings.Join(luxGoConfig.BootstrapIPs, ",")
	luxdConf.BootstrapIDs = strings.Join(luxGoConfig.BootstrapIDs, ",")
	if luxGoConfig.GenesisPath != "" {
		luxdConf.GenesisPath = filepath.Join(constants.DockerNodeConfigPath, constants.GenesisFileName)
	}
	if luxGoConfig.UpgradePath != "" {
		luxdConf.UpgradePath = filepath.Join(constants.DockerNodeConfigPath, constants.UpgradeFileName)
	}
	nodeConf, err := remoteconfig.RenderLuxNodeConfig(luxdConf)
	if err != nil {
		return "", "", err
	}
	nodeConfFile, err := os.CreateTemp("", "luxcli-node-*.yml")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(nodeConfFile.Name(), nodeConf, constants.WriteReadUserOnlyPerms); err != nil {
		return "", "", err
	}
	cChainConf, err := remoteconfig.RenderLuxCChainConfig(luxdConf)
	if err != nil {
		return "", "", err
	}
	cChainConfFile, err := os.CreateTemp("", "luxcli-cchain-*.yml")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(cChainConfFile.Name(), cChainConf, constants.WriteReadUserOnlyPerms); err != nil {
		return "", "", err
	}
	return nodeConfFile.Name(), cChainConfFile.Name(), nil
}

func prepareGrafanaConfig() (string, string, string, string, error) {
	grafanaDataSource, err := remoteconfig.RenderGrafanaLokiDataSourceConfig()
	if err != nil {
		return "", "", "", "", err
	}
	grafanaDataSourceFile, err := os.CreateTemp("", "luxcli-grafana-datasource-*.yml")
	if err != nil {
		return "", "", "", "", err
	}
	if err := os.WriteFile(grafanaDataSourceFile.Name(), grafanaDataSource, constants.WriteReadUserOnlyPerms); err != nil {
		return "", "", "", "", err
	}

	grafanaPromDataSource, err := remoteconfig.RenderGrafanaPrometheusDataSourceConfigg()
	if err != nil {
		return "", "", "", "", err
	}
	grafanaPromDataSourceFile, err := os.CreateTemp("", "luxcli-grafana-prom-datasource-*.yml")
	if err != nil {
		return "", "", "", "", err
	}
	if err := os.WriteFile(grafanaPromDataSourceFile.Name(), grafanaPromDataSource, constants.WriteReadUserOnlyPerms); err != nil {
		return "", "", "", "", err
	}

	grafanaDashboards, err := remoteconfig.RenderGrafanaDashboardConfig()
	if err != nil {
		return "", "", "", "", err
	}
	grafanaDashboardsFile, err := os.CreateTemp("", "luxcli-grafana-dashboards-*.yml")
	if err != nil {
		return "", "", "", "", err
	}
	if err := os.WriteFile(grafanaDashboardsFile.Name(), grafanaDashboards, constants.WriteReadUserOnlyPerms); err != nil {
		return "", "", "", "", err
	}

	grafanaConfig, err := remoteconfig.RenderGrafanaConfig()
	if err != nil {
		return "", "", "", "", err
	}
	grafanaConfigFile, err := os.CreateTemp("", "luxcli-grafana-config-*.ini")
	if err != nil {
		return "", "", "", "", err
	}
	if err := os.WriteFile(grafanaConfigFile.Name(), grafanaConfig, constants.WriteReadUserOnlyPerms); err != nil {
		return "", "", "", "", err
	}
	return grafanaConfigFile.Name(), grafanaDashboardsFile.Name(), grafanaDataSourceFile.Name(), grafanaPromDataSourceFile.Name(), nil
}
