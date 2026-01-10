// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package monitoring provides utilities for setting up monitoring dashboards.
package monitoring

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/constants"
	"github.com/luxfi/net"
)

type configInputs struct {
	LuxdPorts     string
	MachinePorts  string
	LoadTestPorts string
	IP            string
	Port          string
	Host          string
	NodeID        string
	ChainID       string
}

//go:embed dashboards/*
var dashboards embed.FS

//go:embed configs/*
var configs embed.FS

// Setup initializes the monitoring directory with dashboard files.
func Setup(monitoringDir string) error {
	return WriteMonitoringJSONFiles(monitoringDir)
}

// WriteMonitoringJSONFiles writes the dashboard JSON files to the monitoring directory.
func WriteMonitoringJSONFiles(monitoringDir string) error {
	dashboardDir := filepath.Join(monitoringDir, constants.DashboardsDir)
	files, err := dashboards.ReadDir(constants.DashboardsDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		fileContent, err := dashboards.ReadFile(filepath.Join(constants.DashboardsDir, file.Name()))
		if err != nil {
			return err
		}
		dashboardJSONFile, err := os.Create(filepath.Join(dashboardDir, file.Name())) //nolint:gosec // G304: Creating file in app's config directory
		if err != nil {
			return err
		}
		_, err = dashboardJSONFile.Write(fileContent)
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateConfig generates a configuration file from a template.
func GenerateConfig(configPath string, configDesc string, templateVars configInputs) (string, error) {
	configTemplate, err := configs.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	var config bytes.Buffer
	t, err := template.New(configDesc).Parse(string(configTemplate))
	if err != nil {
		return "", err
	}
	err = t.Execute(&config, templateVars)
	if err != nil {
		return "", err
	}
	return config.String(), nil
}

func WritePrometheusConfig(filePath string, luxdPorts []string, machinePorts []string, loadTestPorts []string) error {
	config, err := GenerateConfig("configs/prometheus.yml", "Prometheus Config", configInputs{
		LuxdPorts:     strings.Join(utils.AddSingleQuotes(luxdPorts), ","),
		MachinePorts:  strings.Join(utils.AddSingleQuotes(machinePorts), ","),
		LoadTestPorts: strings.Join(utils.AddSingleQuotes(loadTestPorts), ","),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(config), constants.WriteReadReadPerms)
}

func WriteLokiConfig(filePath string, port string) error {
	config, err := GenerateConfig("configs/loki.yml", "Loki Config", configInputs{
		Port: port,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(config), constants.WriteReadReadPerms)
}

func WritePromtailConfig(filePath string, lokiIP string, lokiPort string, host string, nodeID string, chainID string) error {
	if !netutil.IsValidIP(lokiIP) {
		return fmt.Errorf("invalid IP address: %s", lokiIP)
	}
	config, err := GenerateConfig("configs/promtail.yml", "Promtail Config", configInputs{
		IP:      lokiIP,
		Port:    lokiPort,
		Host:    host,
		NodeID:  nodeID,
		ChainID: chainID,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(config), constants.WriteReadReadPerms)
}
