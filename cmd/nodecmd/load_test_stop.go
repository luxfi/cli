// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	nodePkg "github.com/luxfi/cli/pkg/node"

	"github.com/luxfi/cli/pkg/ansible"
	awsAPI "github.com/luxfi/cli/pkg/cloud/aws"
	gcpAPI "github.com/luxfi/cli/pkg/cloud/gcp"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

var loadTestsToStop []string

func newLoadTestStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [clusterName]",
		Short: "(ALPHA Warning) Stops load test for an existing devnet cluster",
		Long: `(ALPHA Warning) This command is currently in experimental mode. 

The node loadtest stop command stops load testing for an existing devnet cluster and terminates the 
separate cloud server created to host the load test.`,

		Args: cobrautils.ExactArgs(1),
		RunE: stopLoadTest,
	}
	cmd.Flags().StringSliceVar(&loadTestsToStop, "load-test", []string{}, "stop specified load test node(s). Use comma to separate multiple load test instance names")
	return cmd
}

func getLoadTestInstancesInCluster(clusterName string) ([]string, error) {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return nil, err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	clusters, ok := clustersConfig["Clusters"].(map[string]interface{})
	if !ok || clusters == nil {
		return nil, fmt.Errorf("no clusters found")
	}

	cluster, ok := clusters[clusterName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cluster %s doesn't exist", clusterName)
	}

	if loadTestInstance, ok := cluster["LoadTestInstance"].(map[string]string); ok && loadTestInstance != nil {
		return maps.Keys(loadTestInstance), nil
	}
	return nil, fmt.Errorf("no load test instances found")
}

func checkLoadTestExists(clusterName, loadTestName string) (bool, error) {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return false, err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	clusters, ok := clustersConfig["Clusters"].(map[string]interface{})
	if !ok || clusters == nil {
		return false, fmt.Errorf("no clusters found")
	}

	cluster, ok := clusters[clusterName].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("cluster %s doesn't exist", clusterName)
	}

	if loadTestInstance, ok := cluster["LoadTestInstance"].(map[string]string); ok && loadTestInstance != nil {
		_, exists := loadTestInstance[loadTestName]
		return exists, nil
	}
	return false, nil
}

func stopLoadTest(_ *cobra.Command, args []string) error {
	clusterName := args[0]
	var err error
	if len(loadTestsToStop) == 0 {
		loadTestsToStop, err = getLoadTestInstancesInCluster(clusterName)
		if err != nil {
			return err
		}
	}
	separateHostInventoryPath := app.GetLoadTestInventoryDir(clusterName)
	separateHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(separateHostInventoryPath)
	if err != nil {
		return err
	}
	removedLoadTestHosts := []*models.Host{}
	if len(loadTestsToStop) == 0 {
		return fmt.Errorf("no load test instances to stop in cluster %s", clusterName)
	}
	existingLoadTestInstance, err := getExistingLoadTestInstance(clusterName, loadTestsToStop[0])
	if err != nil {
		return err
	}
	nodeToStopConfig, err := app.LoadClusterNodeConfig(clusterName, existingLoadTestInstance)
	if err != nil {
		return err
	}
	clusterNodes, err := nodePkg.GetClusterNodes(app, clusterName)
	if err != nil {
		return err
	}
	cloudSecurityGroupList, err := getCloudSecurityGroupList(clusterNodes)
	if err != nil {
		return err
	}
	cloudService, _ := nodeToStopConfig["CloudService"].(string)
	filteredSGList := utils.Filter(cloudSecurityGroupList, func(sg regionSecurityGroup) bool { return sg.cloud == cloudService })
	if len(filteredSGList) == 0 {
		return fmt.Errorf("no hosts with cloud service %s found in cluster %s", cloudService, clusterName)
	}
	ec2SvcMap := make(map[string]*awsAPI.AwsCloud)
	for _, sg := range filteredSGList {
		sgEc2Svc, err := awsAPI.NewAwsCloud(awsProfile, sg.region)
		if err != nil {
			return err
		}
		if _, ok := ec2SvcMap[sg.region]; !ok {
			ec2SvcMap[sg.region] = sgEc2Svc
		}
	}
	for _, loadTestName := range loadTestsToStop {
		existingSeparateInstance, err = getExistingLoadTestInstance(clusterName, loadTestName)
		if err != nil {
			return err
		}
		if existingSeparateInstance == "" {
			return fmt.Errorf("no existing load test instance found in cluster %s", clusterName)
		}
		nodeConfig, err := app.LoadClusterNodeConfig(clusterName, existingSeparateInstance)
		if err != nil {
			return err
		}
		nodeID, _ := nodeConfig["NodeID"].(string)
		hosts := utils.Filter(separateHosts, func(h *models.Host) bool { return h.GetCloudID() == nodeID })
		if len(hosts) == 0 {
			return fmt.Errorf("host %s is not found in hosts inventory file", nodeID)
		}
		host := hosts[0]
		loadTestResultFileName := fmt.Sprintf("loadtest_%s.txt", loadTestName)
		// Download the load test result from remote cloud server to local machine
		if err = ssh.RunSSHDownloadFile(host, fmt.Sprintf("/home/ubuntu/%s", loadTestResultFileName), filepath.Join(app.GetAnsibleInventoryDirPath(clusterName), loadTestResultFileName)); err != nil {
			ux.Logger.RedXToUser("Unable to download load test result %s to local machine due to %s", loadTestResultFileName, err.Error())
		}
		cloudServiceStr, _ := nodeConfig["CloudService"].(string)
		switch cloudServiceStr {
		case constants.AWSCloudService:
			loadTestNodeConfig, separateHostRegion, err := getNodeCloudConfig(clusterName, existingSeparateInstance)
			if err != nil {
				return err
			}
			loadTestEc2SvcMap, err := getAWSMonitoringEC2Svc(awsProfile, separateHostRegion)
			if err != nil {
				return err
			}
			if err = destroyNode(existingSeparateInstance, clusterName, loadTestName, loadTestEc2SvcMap[separateHostRegion], nil); err != nil {
				return err
			}
			for _, sg := range filteredSGList {
				if err = deleteHostSecurityGroupRule(ec2SvcMap[sg.region], loadTestNodeConfig.PublicIPs[0], sg.securityGroup); err != nil {
					ux.Logger.RedXToUser("unable to delete IP address %s from security group %s in region %s due to %s, please delete it manually",
						loadTestNodeConfig.PublicIPs[0], sg.securityGroup, sg.region, err.Error())
				}
			}
		case constants.GCPCloudService:
			var gcpClient *gcpAPI.GcpCloud
			gcpClient, _, _, _, _, err = getGCPConfig(true)
			if err != nil {
				return err
			}
			if err = destroyNode(existingSeparateInstance, clusterName, loadTestName, nil, gcpClient); err != nil {
				return err
			}
		default:
			return fmt.Errorf("cloud service %s is not supported", cloudServiceStr)
		}
		removedLoadTestHosts = append(removedLoadTestHosts, host)
	}
	return updateLoadTestInventory(separateHosts, removedLoadTestHosts, clusterName, separateHostInventoryPath)
}

func updateLoadTestInventory(separateHosts, removedLoadTestHosts []*models.Host, clusterName, separateHostInventoryPath string) error {
	var remainingLoadTestHosts []*models.Host
	for _, loadTestHost := range separateHosts {
		filteredHosts := utils.Filter(removedLoadTestHosts, func(h *models.Host) bool { return h.IP == loadTestHost.IP })
		if len(filteredHosts) == 0 {
			remainingLoadTestHosts = append(remainingLoadTestHosts, loadTestHost)
		}
	}
	if err := removeLoadTestInventoryDir(clusterName); err != nil {
		return err
	}
	if len(remainingLoadTestHosts) > 0 {
		for _, loadTestHost := range remainingLoadTestHosts {
			nodeConfig, err := app.LoadClusterNodeConfig(clusterName, loadTestHost.GetCloudID())
			if err != nil {
				return err
			}
			cloudServiceStr, _ := nodeConfig["CloudService"].(string)
			nodeIDStr, _ := nodeConfig["NodeID"].(string)
			elasticIPStr, _ := nodeConfig["ElasticIP"].(string)
			if err = ansible.CreateAnsibleHostInventory(separateHostInventoryPath, loadTestHost.SSHPrivateKeyPath, cloudServiceStr, map[string]string{nodeIDStr: elasticIPStr}, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func destroyNode(node, clusterName, loadTestName string, ec2Svc *awsAPI.AwsCloud, gcpClient *gcpAPI.GcpCloud) error {
	nodeConfig, err := app.LoadClusterNodeConfig(clusterName, node)
	if err != nil {
		ux.Logger.RedXToUser("Failed to destroy node %s", node)
		return err
	}
	cloudServiceStr, _ := nodeConfig["CloudService"].(string)
	if cloudServiceStr == "" || cloudServiceStr == constants.AWSCloudService {
		if !(authorizeAccess || nodePkg.AuthorizedAccessFromSettings(app)) && (requestCloudAuth(constants.AWSCloudService) != nil) {
			return fmt.Errorf("cloud access is required")
		}
		// Convert map to NodeConfig struct
		nc := models.NodeConfig{
			NodeID:        nodeConfig["NodeID"].(string),
			Region:        nodeConfig["Region"].(string),
			AMI:           nodeConfig["AMI"].(string),
			KeyPair:       nodeConfig["KeyPair"].(string),
			CertPath:      nodeConfig["CertPath"].(string),
			SecurityGroup: nodeConfig["SecurityGroup"].(string),
			ElasticIP:     nodeConfig["ElasticIP"].(string),
			CloudService:  cloudServiceStr,
			UseStaticIP:   nodeConfig["UseStaticIP"].(bool),
			IsMonitor:     nodeConfig["IsMonitor"].(bool),
			IsWarpRelayer: nodeConfig["IsWarpRelayer"].(bool),
			IsLoadTest:    nodeConfig["IsLoadTest"].(bool),
		}
		if err = ec2Svc.DestroyAWSNode(nc, ""); err != nil {
			if isExpiredCredentialError(err) {
				ux.Logger.PrintToUser("")
				printExpiredCredentialsOutput(awsProfile)
				return nil
			}
			if !errors.Is(err, awsAPI.ErrNodeNotFoundToBeRunning) {
				return err
			}
			nodeIDStr, _ := nodeConfig["NodeID"].(string)
			ux.Logger.PrintToUser("node %s is already destroyed", nodeIDStr)
		}
	} else {
		if !(authorizeAccess || nodePkg.AuthorizedAccessFromSettings(app)) && (requestCloudAuth(constants.GCPCloudService) != nil) {
			return fmt.Errorf("cloud access is required")
		}
		// Convert map to NodeConfig struct for GCP
		gcpNC := models.NodeConfig{
			NodeID:        nodeConfig["NodeID"].(string),
			Region:        nodeConfig["Region"].(string),
			AMI:           nodeConfig["AMI"].(string),
			KeyPair:       nodeConfig["KeyPair"].(string),
			CertPath:      nodeConfig["CertPath"].(string),
			SecurityGroup: nodeConfig["SecurityGroup"].(string),
			ElasticIP:     nodeConfig["ElasticIP"].(string),
			CloudService:  cloudServiceStr,
			UseStaticIP:   nodeConfig["UseStaticIP"].(bool),
			IsMonitor:     nodeConfig["IsMonitor"].(bool),
			IsWarpRelayer: nodeConfig["IsWarpRelayer"].(bool),
			IsLoadTest:    nodeConfig["IsLoadTest"].(bool),
		}
		if err = gcpClient.DestroyGCPNode(gcpNC, ""); err != nil {
			if !errors.Is(err, gcpAPI.ErrNodeNotFoundToBeRunning) {
				return err
			}
			nodeIDStr, _ := nodeConfig["NodeID"].(string)
			ux.Logger.PrintToUser("node %s is already destroyed", nodeIDStr)
		}
	}
	nodeIDStr, _ := nodeConfig["NodeID"].(string)
	ux.Logger.GreenCheckmarkToUser("Node instance %s successfully destroyed!", nodeIDStr)
	if err := removeDeletedNodeDirectory(node); err != nil {
		ux.Logger.RedXToUser("Failed to delete node config for node %s due to %s", node, err.Error())
		return err
	}
	if err := removeLoadTestNodeFromClustersConfig(clusterName, loadTestName); err != nil {
		ux.Logger.RedXToUser("Failed to delete node config for node %s due to %s", node, err.Error())
		return err
	}
	return nil
}

func removeLoadTestNodeFromClustersConfig(clusterName, loadTestName string) error {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	clusters, ok := clustersConfig["Clusters"].(map[string]interface{})
	if !ok || clusters == nil {
		return fmt.Errorf("no clusters found")
	}

	cluster, ok := clusters[clusterName].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cluster %s is not found in cluster config", clusterName)
	}

	if loadTestInstance, ok := cluster["LoadTestInstance"].(map[string]string); ok {
		if _, exists := loadTestInstance[loadTestName]; exists {
			delete(loadTestInstance, loadTestName)
		}
	}

	return app.SaveClustersConfig(clustersConfig)
}

func removeLoadTestInventoryDir(clusterName string) error {
	return os.RemoveAll(app.GetLoadTestInventoryDir(clusterName))
}
