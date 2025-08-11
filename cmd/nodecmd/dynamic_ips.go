// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"context"
	"fmt"

	"github.com/luxfi/cli/pkg/ansible"
	awsAPI "github.com/luxfi/cli/pkg/cloud/aws"
	gcpAPI "github.com/luxfi/cli/pkg/cloud/gcp"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	nodePkg "github.com/luxfi/cli/pkg/node"
	"github.com/luxfi/cli/pkg/ux"
	sdkutils "github.com/luxfi/sdk/utils"
)

func getNodesWithDynamicIP(clusterName string, clusterNodes []string) ([]models.NodeConfig, error) {
	nodesWithDynamicIP := []models.NodeConfig{}
	for _, node := range clusterNodes {
		nodeConfig, err := app.LoadClusterNodeConfig(clusterName, node)
		if err != nil {
			return nil, err
		}
		// Convert map to NodeConfig struct
		useStaticIP, _ := nodeConfig["UseStaticIP"].(bool)
		if !useStaticIP {
			nc := models.NodeConfig{
				NodeID:        nodeConfig["NodeID"].(string),
				Region:        nodeConfig["Region"].(string),
				AMI:           nodeConfig["AMI"].(string),
				KeyPair:       nodeConfig["KeyPair"].(string),
				CertPath:      nodeConfig["CertPath"].(string),
				SecurityGroup: nodeConfig["SecurityGroup"].(string),
				ElasticIP:     nodeConfig["ElasticIP"].(string),
				CloudService:  nodeConfig["CloudService"].(string),
				UseStaticIP:   useStaticIP,
				IsMonitor:     nodeConfig["IsMonitor"].(bool),
				IsWarpRelayer: nodeConfig["IsWarpRelayer"].(bool),
				IsLoadTest:    nodeConfig["IsLoadTest"].(bool),
			}
			nodesWithDynamicIP = append(nodesWithDynamicIP, nc)
		}
	}
	return nodesWithDynamicIP, nil
}

func getPublicIPsForNodesWithDynamicIP(nodesWithDynamicIP []models.NodeConfig) (map[string]string, error) {
	publicIPMap := make(map[string]string)
	var (
		err        error
		lastRegion string
		ec2Svc     *awsAPI.AwsCloud
		gcpCloud   *gcpAPI.GcpCloud
	)
	ux.Logger.PrintToUser("Getting Public IP(s) for node(s) with dynamic IP ...")
	for _, node := range nodesWithDynamicIP {
		if lastRegion == "" || node.Region != lastRegion {
			if node.CloudService == "" || node.CloudService == constants.AWSCloudService {
				ec2Svc, err = awsAPI.NewAwsCloud(awsProfile, node.Region)
				if err != nil {
					return nil, err
				}
			}
			lastRegion = node.Region
		}
		var publicIP map[string]string
		if node.CloudService == constants.GCPCloudService {
			if !(authorizeAccess || nodePkg.AuthorizedAccessFromSettings(app)) && (requestCloudAuth(constants.GCPCloudService) != nil) {
				return nil, fmt.Errorf("cloud access is required")
			}
			if gcpCloud == nil {
				gcpClient, projectName, _, err := getGCPCloudCredentials()
				if err != nil {
					return nil, err
				}
				gcpCloud, err = gcpAPI.NewGcpCloud(gcpClient, projectName, context.Background())
				if err != nil {
					return nil, err
				}
			}
			publicIP, err = gcpCloud.GetInstancePublicIPs(node.Region, []string{node.NodeID})
			if err != nil {
				return nil, err
			}
		} else {
			publicIP, err = ec2Svc.GetInstancePublicIPs([]string{node.NodeID})
			if err != nil {
				if isExpiredCredentialError(err) {
					ux.Logger.PrintToUser("")
					printExpiredCredentialsOutput(awsProfile)
				}
				return nil, err
			}
		}
		publicIPMap[node.NodeID] = publicIP[node.NodeID]
	}
	return publicIPMap, nil
}

// update public IPs
// - in ansible inventory file
// - in host config file
func updatePublicIPs(clusterName string) error {
	clusterNodes, err := nodePkg.GetClusterNodes(app, clusterName)
	if err != nil {
		return err
	}
	nodesWithDynamicIP, err := getNodesWithDynamicIP(clusterName, clusterNodes)
	if err != nil {
		return err
	}
	if len(nodesWithDynamicIP) > 0 {
		nodeIDs := sdkutils.Map(nodesWithDynamicIP, func(c models.NodeConfig) string { return c.NodeID })
		ux.Logger.PrintToUser("Nodes with dynamic IPs in cluster: %s", nodeIDs)
		publicIPMap, err := getPublicIPsForNodesWithDynamicIP(nodesWithDynamicIP)
		if err != nil {
			return err
		}
		changed := 0
		for _, node := range nodesWithDynamicIP {
			if node.ElasticIP != publicIPMap[node.NodeID] {
				ux.Logger.PrintToUser("Updating IP information from %s to %s for node %s",
					node.ElasticIP,
					publicIPMap[node.NodeID],
					node.NodeID,
				)
				changed++
			}
			node.ElasticIP = publicIPMap[node.NodeID]
			if err := app.CreateNodeCloudConfigFile(node.NodeID, &node); err != nil {
				return err
			}
		}
		if changed == 0 {
			ux.Logger.PrintToUser("No changes to IPs detected")
			return nil
		}
		if err = ansible.UpdateInventoryHostPublicIP(app.GetAnsibleInventoryDirPath(clusterName), publicIPMap); err != nil {
			return err
		}
	} else {
		ux.Logger.PrintToUser("No nodes with dynamic IPs in cluster")
	}
	return nil
}
