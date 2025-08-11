// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	nodePkg "github.com/luxfi/cli/pkg/node"

	awsAPI "github.com/luxfi/cli/pkg/cloud/aws"
	gcpAPI "github.com/luxfi/cli/pkg/cloud/gcp"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"golang.org/x/exp/maps"
	"golang.org/x/net/context"

	"github.com/spf13/cobra"
)

var (
	authorizeRemove bool
	authorizeAll    bool
	destroyAll      bool
)

func newDestroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy [clusterName]",
		Short: "(ALPHA Warning) Destroys all nodes in a cluster",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node destroy command terminates all running nodes in cloud server and deletes all storage disks.

If there is a static IP address attached, it will be released.`,
		Args: cobrautils.MinimumNArgs(0),
		RunE: destroyNodes,
	}
	cmd.Flags().BoolVar(&authorizeAccess, "authorize-access", false, "authorize CLI to release cloud resources")
	cmd.Flags().BoolVar(&authorizeRemove, "authorize-remove", false, "authorize CLI to remove all local files related to cloud nodes")
	cmd.Flags().BoolVarP(&authorizeAll, "authorize-all", "y", false, "authorize all CLI requests")
	cmd.Flags().BoolVar(&destroyAll, "all", false, "destroy all existing clusters created by Lux CLI")
	cmd.Flags().StringVar(&awsProfile, "aws-profile", constants.AWSDefaultCredential, "aws profile to use")

	return cmd
}

func removeNodeFromClustersConfig(clusterName string) error {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	if clusters, ok := clustersConfig["Clusters"].(map[string]interface{}); ok && clusters != nil {
		delete(clusters, clusterName)
	}
	return app.SaveClustersConfig(clustersConfig)
}

func removeDeletedNodeDirectory(clusterName string) error {
	return os.RemoveAll(app.GetNodeInstanceDirPath(clusterName))
}

func removeClusterInventoryDir(clusterName string) error {
	return os.RemoveAll(app.GetAnsibleInventoryDirPath(clusterName))
}

func getDeleteConfigConfirmation() error {
	if authorizeRemove {
		return nil
	}
	ux.Logger.PrintToUser("Please note that if your node(s) are validating a Subnet, destroying them could cause Subnet instability and it is irreversible")
	confirm := "Running this command will delete all stored files associated with your cloud server. Do you want to proceed? " +
		fmt.Sprintf("Stored files can be found at %s", app.GetNodesDir())
	yes, err := app.Prompt.CaptureYesNo(confirm)
	if err != nil {
		return err
	}
	if !yes {
		return errors.New("abort lux node destroy command")
	}
	return nil
}

func removeClustersConfigFiles(clusterName string) error {
	if err := removeClusterInventoryDir(clusterName); err != nil {
		return err
	}
	return removeNodeFromClustersConfig(clusterName)
}

func CallDestroyNode(clusterName string) error {
	authorizeAll = true
	return destroyNodes(nil, []string{clusterName})
}

// We need to get which cloud service is being used on a cluster
// getFirstAvailableNode gets first node in the cluster that still has its node_config.json
// This is because some nodes might have had their node_config.json file deleted as part of
// deletion process but if an error occurs during deletion process, the node might still exist
// as part of the cluster in cluster_config.json
// If all nodes in the cluster no longer have their node_config.json files, getFirstAvailableNode
// will return false in its second return value
func getFirstAvailableNode(nodesToStop []string) (string, bool) {
	firstAvailableNode := nodesToStop[0]
	noAvailableNodesFound := false
	for index, node := range nodesToStop {
		nodeConfigPath := app.GetNodeConfigPath(node)
		if !utils.FileExists(nodeConfigPath) {
			if index == len(nodesToStop)-1 {
				noAvailableNodesFound = true
			}
			continue
		}
		firstAvailableNode = node
	}
	return firstAvailableNode, noAvailableNodesFound
}

func Cleanup() error {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	var clusterNames []string
	if clusters, ok := clustersConfig["Clusters"].(map[string]interface{}); ok && clusters != nil {
		clusterNames = maps.Keys(clusters)
	}
	for _, clusterName := range clusterNames {
		if err = CallDestroyNode(clusterName); err != nil {
			// we only return error for invalid cloud credentials
			// silence for other errors
			// Differentiate between AWS and GCP credentials
			if strings.Contains(err.Error(), "invalid cloud credentials") {
				if strings.Contains(err.Error(), "GCP") || strings.Contains(err.Error(), "Google") {
					return fmt.Errorf("invalid GCP credentials")
				}
				return fmt.Errorf("invalid AWS credentials")
			}
		}
	}
	ux.Logger.PrintToUser("all existing instances created by Lux CLI successfully destroyed")
	return nil
}

func destroyNodes(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		if !destroyAll {
			return fmt.Errorf("to destroy all existing clusters created by Lux CLI, call lux node destroy --all. To destroy a specified cluster, call lux node destroy CLUSTERNAME")
		}
		return Cleanup()
	}
	clusterName := args[0]
	if err := nodePkg.CheckCluster(app, clusterName); err != nil {
		return err
	}
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	// clusterConfig is a map[string]interface{}, not a struct
	if local, ok := clusterConfig["Local"].(bool); ok && local {
		return notImplementedForLocal("destroy")
	}
	isExternalCluster, err := checkClusterExternal(clusterName)
	if err != nil {
		return err
	}
	if authorizeAll {
		authorizeAccess = true
		authorizeRemove = true
	}
	if err := getDeleteConfigConfirmation(); err != nil {
		return err
	}
	nodesToStop, err := nodePkg.GetClusterNodes(app, clusterName)
	if err != nil {
		return err
	}
	monitoringNode, err := getClusterMonitoringNode(clusterName)
	if err != nil {
		return err
	}
	if monitoringNode != "" {
		nodesToStop = append(nodesToStop, monitoringNode)
	}
	// stop all load test nodes if specified
	ltHosts, err := getLoadTestInstancesInCluster(clusterName)
	if err != nil {
		return err
	}
	for _, loadTestName := range ltHosts {
		ltInstance, err := getExistingLoadTestInstance(clusterName, loadTestName)
		if err != nil {
			return err
		}
		nodesToStop = append(nodesToStop, ltInstance)
	}
	nodeErrors := map[string]error{}
	cloudSecurityGroupList, err := getCloudSecurityGroupList(nodesToStop)
	if err != nil {
		return err
	}
	firstAvailableNodes, noAvailableNodesFound := getFirstAvailableNode(nodesToStop)
	if noAvailableNodesFound {
		return removeClustersConfigFiles(clusterName)
	}
	nodeToStopConfig, err := app.LoadClusterNodeConfig(clusterName, firstAvailableNodes)
	if err != nil {
		return err
	}
	// Filter security groups by cloud service type to support mixed cloud clusters
	cloudService, _ := nodeToStopConfig["CloudService"].(string)
	filteredSGList := utils.Filter(cloudSecurityGroupList, func(sg regionSecurityGroup) bool { return sg.cloud == cloudService })
	if len(filteredSGList) == 0 {
		return fmt.Errorf("no endpoint found in the  %s", cloudService)
	}
	var gcpCloud *gcpAPI.GcpCloud
	ec2SvcMap := make(map[string]*awsAPI.AwsCloud)
	// Handle both AWS and GCP cloud services
	if cloudService == constants.GCPCloudService {
		// Initialize GCP cloud service
		// GCP support is not fully implemented yet
		return fmt.Errorf("GCP cloud service is not yet fully implemented")
	} else if cloudService == constants.AWSCloudService {
		for _, sg := range filteredSGList {
			sgEc2Svc, err := awsAPI.NewAwsCloud(awsProfile, sg.region)
			if err != nil {
				return err
			}
			ec2SvcMap[sg.region] = sgEc2Svc
		}
	}
	for _, node := range nodesToStop {
		if !isExternalCluster {
			// if we can't find node config path, that means node already deleted on console
			// but we didn't get to delete the node from cluster config file
			if !utils.FileExists(app.GetNodeConfigPath(node)) {
				continue
			}
			nodeConfig, err := app.LoadClusterNodeConfig(clusterName, node)
			if err != nil {
				nodeErrors[node] = err
				ux.Logger.RedXToUser("Failed to destroy node %s due to %s", node, err.Error())
				continue
			}
			nodeCloudService, _ := nodeConfig["CloudService"].(string)
			if nodeCloudService == "" || nodeCloudService == constants.AWSCloudService {
				if !(authorizeAccess || nodePkg.AuthorizedAccessFromSettings(app)) && (requestCloudAuth(constants.AWSCloudService) != nil) {
					return fmt.Errorf("cloud access is required")
				}
				// Convert map to NodeConfig struct
				nodeRegion, _ := nodeConfig["Region"].(string)
				nc := models.NodeConfig{
					NodeID:        nodeConfig["NodeID"].(string),
					Region:        nodeRegion,
					AMI:           nodeConfig["AMI"].(string),
					KeyPair:       nodeConfig["KeyPair"].(string),
					CertPath:      nodeConfig["CertPath"].(string),
					SecurityGroup: nodeConfig["SecurityGroup"].(string),
					ElasticIP:     nodeConfig["ElasticIP"].(string),
					CloudService:  nodeCloudService,
					UseStaticIP:   nodeConfig["UseStaticIP"].(bool),
					IsMonitor:     nodeConfig["IsMonitor"].(bool),
					IsWarpRelayer: nodeConfig["IsWarpRelayer"].(bool),
					IsLoadTest:    nodeConfig["IsLoadTest"].(bool),
				}
				if err = ec2SvcMap[nodeRegion].DestroyAWSNode(nc, clusterName); err != nil {
					if isExpiredCredentialError(err) {
						ux.Logger.PrintToUser("")
						printExpiredCredentialsOutput(awsProfile)
						return fmt.Errorf("invalid cloud credentials")
					}
					if !errors.Is(err, awsAPI.ErrNodeNotFoundToBeRunning) {
						nodeErrors[node] = err
						continue
					}
					ux.Logger.PrintToUser("node %s is already destroyed", nc.NodeID)
				}
				for _, sg := range filteredSGList {
					if err = deleteHostSecurityGroupRule(ec2SvcMap[sg.region], nc.ElasticIP, sg.securityGroup); err != nil {
						ux.Logger.RedXToUser("unable to delete IP address %s from security group %s in region %s due to %s, please delete it manually",
							nc.ElasticIP, sg.securityGroup, sg.region, err.Error())
					}
				}
			} else {
				if !(authorizeAccess || nodePkg.AuthorizedAccessFromSettings(app)) && (requestCloudAuth(constants.GCPCloudService) != nil) {
					return fmt.Errorf("cloud access is required")
				}
				if gcpCloud == nil {
					gcpClient, projectName, _, err := getGCPCloudCredentials()
					if err != nil {
						return err
					}
					gcpCloud, err = gcpAPI.NewGcpCloud(gcpClient, projectName, context.Background())
					if err != nil {
						return err
					}
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
					CloudService:  nodeCloudService,
					UseStaticIP:   nodeConfig["UseStaticIP"].(bool),
					IsMonitor:     nodeConfig["IsMonitor"].(bool),
					IsWarpRelayer: nodeConfig["IsWarpRelayer"].(bool),
					IsLoadTest:    nodeConfig["IsLoadTest"].(bool),
				}
				if err = gcpCloud.DestroyGCPNode(gcpNC, clusterName); err != nil {
					if !errors.Is(err, gcpAPI.ErrNodeNotFoundToBeRunning) {
						nodeErrors[node] = err
						continue
					}
					ux.Logger.GreenCheckmarkToUser("node %s is already destroyed", gcpNC.NodeID)
				}
			}
			nodeID, _ := nodeConfig["NodeID"].(string)
			ux.Logger.GreenCheckmarkToUser("Node instance %s in cluster %s successfully destroyed!", nodeID, clusterName)
		}
		if err := removeDeletedNodeDirectory(node); err != nil {
			ux.Logger.RedXToUser("Failed to delete node config for node %s due to %s", node, err.Error())
			return err
		}
	}
	if len(nodeErrors) > 0 {
		ux.Logger.PrintToUser("Failed nodes: ")
		invalidCloudCredentials := false
		for node, nodeErr := range nodeErrors {
			if strings.Contains(nodeErr.Error(), constants.ErrReleasingGCPStaticIP) {
				ux.Logger.RedXToUser("Node is destroyed, but failed to release static ip address for node %s due to %s", node, nodeErr)
			} else {
				if strings.Contains(nodeErr.Error(), "AuthFailure") {
					invalidCloudCredentials = true
				}
				ux.Logger.RedXToUser("Failed to destroy node %s due to %s", node, nodeErr)
			}
		}
		if invalidCloudCredentials {
			return fmt.Errorf("failed to destroy node(s) due to invalid cloud credentials %s", maps.Keys(nodeErrors))
		}
		return fmt.Errorf("failed to destroy node(s) %s", maps.Keys(nodeErrors))
	} else {
		if isExternalCluster {
			ux.Logger.GreenCheckmarkToUser("All nodes in EXTERNAL cluster %s are successfully removed!", clusterName)
		} else {
			ux.Logger.GreenCheckmarkToUser("All nodes in cluster %s are successfully destroyed!", clusterName)
		}
	}

	return removeClustersConfigFiles(clusterName)
}

func getClusterMonitoringNode(clusterName string) (string, error) {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return "", err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	clusters, ok := clustersConfig["Clusters"].(map[string]interface{})
	if !ok || clusters == nil {
		return "", fmt.Errorf("no clusters found")
	}
	cluster, ok := clusters[clusterName].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("cluster %q does not exist", clusterName)
	}
	monitoringInstance, _ := cluster["MonitoringInstance"].(string)
	return monitoringInstance, nil
}
