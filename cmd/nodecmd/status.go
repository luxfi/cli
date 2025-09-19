// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/luxfi/cli/cmd/blockchaincmd"
	"github.com/luxfi/cli/pkg/ansible"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/node"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/vms/platformvm/status"

	"github.com/olekukonko/tablewriter"
	"github.com/pborman/ansi"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var blockchainName string

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [clusterName]",
		Short: "(ALPHA Warning) Get node bootstrap status",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node status command gets the bootstrap status of all nodes in a cluster with the Primary Network. 
If no cluster is given, defaults to node list behaviour.

To get the bootstrap status of a node with a Blockchain, use --blockchain flag`,
		Args: cobrautils.MinimumNArgs(0),
		RunE: statusNode,
	}
	cmd.Flags().StringVar(&blockchainName, "subnet", "", "specify the blockchain the node is syncing with")
	cmd.Flags().StringVar(&blockchainName, "blockchain", "", "specify the blockchain the node is syncing with")

	return cmd
}

func statusNode(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return list(nil, nil)
	}
	clusterName := args[0]
	if err := node.CheckCluster(app, clusterName); err != nil {
		return err
	}
	clusterConf, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	// local cluster doesn't have nodes
	if isLocal, ok := clusterConf["Local"].(bool); ok && isLocal {
		return notImplementedForLocal("status")
	}
	var blockchainID ids.ID
	if blockchainName != "" {
		sc, err := app.LoadSidecar(blockchainName)
		if err != nil {
			return err
		}
		// Get network name from cluster config
		var networkName string
		if network, ok := clusterConf["Network"].(map[string]interface{}); ok {
			if name, ok := network["Name"].(string); ok {
				networkName = name
			}
		}
		if networkName != "" {
			blockchainID = sc.Networks[networkName].BlockchainID
			if blockchainID == ids.Empty {
				return constants.ErrNoBlockchainID
			}
		}
	}

	// Get cloud IDs from cluster config
	var hostIDs []string
	if nodes, ok := clusterConf["Nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if nodeStr, ok := node.(string); ok {
				hostIDs = append(hostIDs, nodeStr)
			}
		}
	}
	nodeIDs, err := utils.MapWithError(hostIDs, func(s string) (string, error) {
		n, err := getNodeID(app.GetNodeInstanceDirPath(s))
		return n.String(), err
	})
	if err != nil {
		return err
	}
	if blockchainName != "" {
		// check subnet first
		if _, err := blockchaincmd.ValidateSubnetNameAndGetChains([]string{blockchainName}); err != nil {
			return err
		}
	}

	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	defer node.DisconnectHosts(hosts)

	spinSession := ux.NewUserSpinner()
	spinner := spinSession.SpinToUser("Checking node(s) status...")
	notBootstrappedNodes, err := node.GetNotBootstrappedNodes(hosts)
	if err != nil {
		ux.SpinFailWithError(spinner, "", err)
		return err
	}
	ux.SpinComplete(spinner)

	spinner = spinSession.SpinToUser("Checking if node(s) are healthy...")
	unhealthyNodes, err := node.GetUnhealthyNodes(hosts)
	if err != nil {
		ux.SpinFailWithError(spinner, "", err)
		return err
	}
	ux.SpinComplete(spinner)

	spinner = spinSession.SpinToUser("Getting luxd version of node(s)...")
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if resp, err := ssh.RunSSHCheckLuxdVersion(host); err != nil {
				nodeResults.AddResult(host.GetCloudID(), nil, err)
				return
			} else {
				if luxdVersion, _, err := node.ParseLuxdOutput(resp); err != nil {
					nodeResults.AddResult(host.GetCloudID(), nil, err)
				} else {
					nodeResults.AddResult(host.GetCloudID(), luxdVersion, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()

	if wgResults.HasErrors() {
		e := fmt.Errorf("failed to get luxd version for node(s) %s", wgResults.GetErrorHostMap())
		ux.SpinFailWithError(spinner, "", e)
		return e
	}
	ux.SpinComplete(spinner)
	spinSession.Stop()
	luxdVersions := map[string]string{}
	for nodeID, luxdVersion := range wgResults.GetResultMap() {
		luxdVersions[nodeID] = fmt.Sprintf("%v", luxdVersion)
	}

	notSyncedNodes := []string{}
	subnetSyncedNodes := []string{}
	subnetValidatingNodes := []string{}
	if blockchainName != "" {
		hostsToCheckSyncStatus := []string{}
		for _, hostID := range hostIDs {
			if slices.Contains(notBootstrappedNodes, hostID) {
				notSyncedNodes = append(notSyncedNodes, hostID)
			} else {
				hostsToCheckSyncStatus = append(hostsToCheckSyncStatus, hostID)
			}
		}
		if len(hostsToCheckSyncStatus) != 0 {
			ux.Logger.PrintToUser("Getting subnet sync status of node(s)")
			hostsToCheck := utils.Filter(hosts, func(h *models.Host) bool { return slices.Contains(hostsToCheckSyncStatus, h.GetCloudID()) })
			wg := sync.WaitGroup{}
			wgResults := models.NodeResults{}
			for _, host := range hostsToCheck {
				wg.Add(1)
				go func(nodeResults *models.NodeResults, host *models.Host) {
					defer wg.Done()
					if syncstatus, err := ssh.RunSSHSubnetSyncStatus(host, blockchainID.String()); err != nil {
						nodeResults.AddResult(host.GetCloudID(), nil, err)
						return
					} else {
						if subnetSyncStatus, err := parseSubnetSyncOutput(syncstatus); err != nil {
							nodeResults.AddResult(host.GetCloudID(), nil, err)
							return
						} else {
							nodeResults.AddResult(host.GetCloudID(), subnetSyncStatus, err)
						}
					}
				}(&wgResults, host)
			}
			wg.Wait()
			if wgResults.HasErrors() {
				return fmt.Errorf("failed to check sync status for node(s) %s", wgResults.GetErrorHostMap())
			}
			for nodeID, subnetSyncStatus := range wgResults.GetResultMap() {
				switch subnetSyncStatus {
				case status.Syncing.String():
					subnetSyncedNodes = append(subnetSyncedNodes, nodeID)
				case status.Validating.String():
					subnetValidatingNodes = append(subnetValidatingNodes, nodeID)
				default:
					notSyncedNodes = append(notSyncedNodes, nodeID)
				}
			}
		}
	}
	// clusterConf is a map[string]interface{}, not a struct
	if monitoringInstance, ok := clusterConf["MonitoringInstance"].(string); ok && monitoringInstance != "" {
		hostIDs = append(hostIDs, monitoringInstance)
		nodeIDs = append(nodeIDs, "")
	}
	nodeConfigs := []models.NodeConfig{}
	for _, hostID := range hostIDs {
		nodeConfigMap, err := app.LoadClusterNodeConfig(clusterName, hostID)
		if err != nil {
			return err
		}
		// Convert map to NodeConfig struct
		nodeConfig := models.NodeConfig{
			NodeID:        nodeConfigMap["NodeID"].(string),
			Region:        nodeConfigMap["Region"].(string),
			AMI:           nodeConfigMap["AMI"].(string),
			KeyPair:       nodeConfigMap["KeyPair"].(string),
			CertPath:      nodeConfigMap["CertPath"].(string),
			SecurityGroup: nodeConfigMap["SecurityGroup"].(string),
			ElasticIP:     nodeConfigMap["ElasticIP"].(string),
			CloudService:  nodeConfigMap["CloudService"].(string),
			UseStaticIP:   nodeConfigMap["UseStaticIP"].(bool),
			IsMonitor:     nodeConfigMap["IsMonitor"].(bool),
			IsWarpRelayer: nodeConfigMap["IsWarpRelayer"].(bool),
			IsLoadTest:    nodeConfigMap["IsLoadTest"].(bool),
		}
		nodeConfigs = append(nodeConfigs, nodeConfig)
	}
	printOutput(
		clusterConf,
		hostIDs,
		nodeIDs,
		luxdVersions,
		unhealthyNodes,
		notBootstrappedNodes,
		notSyncedNodes,
		subnetSyncedNodes,
		subnetValidatingNodes,
		clusterName,
		blockchainName,
		nodeConfigs,
	)
	return nil
}

func printOutput(
	clusterConf map[string]interface{},
	cloudIDs []string,
	nodeIDs []string,
	luxdVersions map[string]string,
	unhealthyHosts []string,
	notBootstrappedHosts []string,
	notSyncedHosts []string,
	subnetSyncedHosts []string,
	subnetValidatingHosts []string,
	clusterName string,
	blockchainName string,
	nodeConfigs []models.NodeConfig,
) {
	// clusterConf is a map[string]interface{}, not a struct
	if external, ok := clusterConf["External"].(bool); ok && external {
		network, _ := clusterConf["Network"].(map[string]interface{})
		kind, _ := network["Kind"].(string)
		ux.Logger.PrintToUser("Cluster %s (%s) is EXTERNAL", logging.LightBlue.Wrap(clusterName), kind)
	}
	if blockchainName == "" && len(notBootstrappedHosts) == 0 {
		ux.Logger.PrintToUser("All nodes in cluster %s are bootstrapped to Primary Network!", clusterName)
	}
	if blockchainName != "" && len(notSyncedHosts) == 0 {
		// all nodes are either synced to or validating subnet
		status := "synced to"
		if len(subnetSyncedHosts) == 0 {
			status = "validators of"
		}
		ux.Logger.PrintToUser("All nodes in cluster %s are %s Subnet %s", logging.LightBlue.Wrap(clusterName), status, blockchainName)
	}
	ux.Logger.PrintToUser("")
	tit := fmt.Sprintf("STATUS FOR CLUSTER: %s", logging.LightBlue.Wrap(clusterName))
	ux.Logger.PrintToUser(tit)
	ux.Logger.PrintToUser(strings.Repeat("=", len(removeColors(tit))))
	ux.Logger.PrintToUser("")
	header := []string{"Cloud ID", "Node ID", "IP", "Network", "Role", "Luxd Version", "Primary Network", "Healthy"}
	if blockchainName != "" {
		header = append(header, "Subnet "+blockchainName)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetRowLine(true)
	for i, cloudID := range cloudIDs {
		boostrappedStatus := ""
		healthyStatus := ""
		nodeIDStr := ""
		luxdVersion := ""
		// Extract roles from nodeConfig
		nodeConfig := nodeConfigs[i]
		roles := []string{}
		if nodeConfig.IsMonitor {
			roles = append(roles, "Monitor")
		}
		if nodeConfig.IsWarpRelayer {
			roles = append(roles, "WarpRelayer")
		}
		if nodeConfig.IsLoadTest {
			roles = append(roles, "LoadTest")
		}
		
		// Check if it's a luxd host (typically all hosts are luxd hosts unless they're monitoring or loadtest only)
		isLuxdHost := true
		if nodeConfig.IsMonitor && !nodeConfig.IsWarpRelayer && !nodeConfig.IsLoadTest {
			// Only monitor, not a luxd host
			isLuxdHost = false
		}
		if isLuxdHost {
			boostrappedStatus = logging.Green.Wrap("BOOTSTRAPPED")
			if slices.Contains(notBootstrappedHosts, cloudID) {
				boostrappedStatus = logging.Red.Wrap("NOT_BOOTSTRAPPED")
			}
			healthyStatus = logging.Green.Wrap("OK")
			if slices.Contains(unhealthyHosts, cloudID) {
				healthyStatus = logging.Red.Wrap("UNHEALTHY")
			}
			nodeIDStr = nodeIDs[i]
			luxdVersion = luxdVersions[cloudID]
		}
		row := []string{
			cloudID,
			logging.Green.Wrap(nodeIDStr),
			nodeConfigs[i].ElasticIP,
			func() string {
				network, _ := clusterConf["Network"].(map[string]interface{})
				kind, _ := network["Kind"].(string)
				return kind
			}(),
			strings.Join(roles, ","),
			luxdVersion,
			boostrappedStatus,
			healthyStatus,
		}
		if blockchainName != "" {
			syncedStatus := ""
			monitoringInstance, _ := clusterConf["MonitoringInstance"].(string)
			if monitoringInstance != cloudID {
				syncedStatus = logging.Red.Wrap("NOT_BOOTSTRAPPED")
				if slices.Contains(subnetSyncedHosts, cloudID) {
					syncedStatus = logging.Green.Wrap("SYNCED")
				}
				if slices.Contains(subnetValidatingHosts, cloudID) {
					syncedStatus = logging.Green.Wrap("VALIDATING")
				}
			}
			row = append(row, syncedStatus)
		}
		table.Append(row)
	}
	table.Render()
}

func removeColors(s string) string {
	bs, err := ansi.Strip([]byte(s))
	if err != nil {
		return s
	}
	return string(bs)
}
