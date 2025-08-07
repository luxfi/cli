// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package node

import (
	"github.com/luxfi/cli/pkg/ansible"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/utils"
)

func GetHostWithCloudID(app *application.Lux, clusterName string, cloudID string) (*models.Host, error) {
	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return nil, err
	}
	monitoringInventoryFile := app.GetMonitoringInventoryDir(clusterName)
	if utils.FileExists(monitoringInventoryFile) {
		monitoringHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(monitoringInventoryFile)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, monitoringHosts...)
	}
	for _, host := range hosts {
		if host.GetCloudID() == cloudID {
			return host, nil
		}
	}
	return nil, nil
}

func GetWarpRelayerHost(app *application.Lux, clusterName string) (*models.Host, error) {
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}
	relayerCloudID := ""
	
	// Type assertion for nodes field
	nodes, _ := clusterConfig["nodes"].([]interface{})
	for _, nodeData := range nodes {
		var cloudID string
		switch v := nodeData.(type) {
		case string:
			cloudID = v
		case map[string]interface{}:
			if id, ok := v["cloudID"].(string); ok {
				cloudID = id
			} else if id, ok := v["nodeID"].(string); ok {
				cloudID = id
			}
		}
		
		if cloudID != "" {
			if nodeConfig, err := app.LoadClusterNodeConfig(clusterName, cloudID); err == nil {
				// Check if this node is a warp relayer
				if isRelayer, ok := nodeConfig["isWarpRelayer"].(bool); ok && isRelayer {
					if nodeID, ok := nodeConfig["nodeID"].(string); ok {
						relayerCloudID = nodeID
					}
				}
			}
		}
	}
	return GetHostWithCloudID(app, clusterName, relayerCloudID)
}
