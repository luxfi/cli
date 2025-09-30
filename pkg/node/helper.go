// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package node

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/luxfi/cli/pkg/ansible"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/sdk/models"
	sdkutils "github.com/luxfi/sdk/utils"
)

const (
	HealthCheckPoolTime = 60 * time.Second
	HealthCheckTimeout  = 3 * time.Minute
)

func AuthorizedAccessFromSettings(app *application.Lux) bool {
	return app.Conf.GetConfigBoolValue(constants.ConfigAuthorizeCloudAccessKey)
}

// isAPIOnlyNode checks if a host is configured as an API-only node
func isAPIOnlyNode(clusterData map[string]interface{}, host models.Host) bool {
	// Check if node has API-only role in cluster configuration
	if nodes, ok := clusterData["nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if nodeID, hasID := nodeMap["id"].(string); hasID && nodeID == host.GetCloudID() {
					// Check if node is marked as API-only
					if nodeType, hasType := nodeMap["type"].(string); hasType && nodeType == "api" {
						return true
					}
					// Also check roles field
					if roles, hasRoles := nodeMap["roles"].([]interface{}); hasRoles {
						for _, role := range roles {
							if roleStr, ok := role.(string); ok && roleStr == "api-only" {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func CheckCluster(app *application.Lux, clusterName string) error {
	_, err := GetClusterNodes(app, clusterName)
	return err
}

func GetClusterNodes(app *application.Lux, clusterName string) ([]string, error) {
	if exists, err := CheckClusterExists(app, clusterName); err != nil || !exists {
		return nil, fmt.Errorf("cluster %q not found", clusterName)
	}
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}

	// Type assertions for map[string]interface{}
	nodesData, ok := clusterConfig["nodes"].([]interface{})
	if !ok {
		// Try as empty slice
		nodesData = []interface{}{}
	}

	// Convert to []string (node IDs or names)
	clusterNodes := make([]string, 0, len(nodesData))
	for _, nodeData := range nodesData {
		switch v := nodeData.(type) {
		case string:
			clusterNodes = append(clusterNodes, v)
		case map[string]interface{}:
			// Try to get node ID or IP
			if nodeID, ok := v["nodeID"].(string); ok {
				clusterNodes = append(clusterNodes, nodeID)
			} else if ip, ok := v["ip"].(string); ok {
				clusterNodes = append(clusterNodes, ip)
			}
		}
	}

	isLocal, _ := clusterConfig["local"].(bool)
	if len(clusterNodes) == 0 && !isLocal {
		return nil, fmt.Errorf("no nodes found in cluster %s", clusterName)
	}
	return clusterNodes, nil
}

func CheckClusterExists(app *application.Lux, clusterName string) (bool, error) {
	if !app.ClustersConfigExists() {
		return false, nil
	}

	clustersConfig, err := app.LoadClustersConfig()
	if err != nil {
		return false, err
	}

	// Type assertion for clusters field
	clusters, ok := clustersConfig["clusters"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	_, exists := clusters[clusterName]
	return exists, nil
}

func CheckHostsAreRPCCompatible(app *application.Lux, hosts []*models.Host, subnetName string) error {
	incompatibleNodes, err := getRPCIncompatibleNodes(app, hosts, subnetName)
	if err != nil {
		return err
	}
	if len(incompatibleNodes) > 0 {
		sc, err := app.LoadSidecar(subnetName)
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser("Either modify your Lux Go version or modify your VM version")
		ux.Logger.PrintToUser("To modify your Lux Go version: https://docs.lux.network/nodes/maintain/upgrade-your-luxd-node")
		switch sc.VM {
		case models.SubnetEvm:
			ux.Logger.PrintToUser("To modify your Subnet-EVM version: https://docs.lux.network/build/subnet/upgrade/upgrade-subnet-vm")
		case models.CustomVM:
			ux.Logger.PrintToUser("To modify your Custom VM binary: lux blockchain upgrade vm %s --config", subnetName)
		}
		ux.Logger.PrintToUser("Yoy can use \"lux node upgrade\" to upgrade Lux Go and/or Subnet-EVM to their latest versions")
		return fmt.Errorf("the Lux Go version of node(s) %s is incompatible with VM RPC version of %s", incompatibleNodes, subnetName)
	}
	return nil
}

func getRPCIncompatibleNodes(app *application.Lux, hosts []*models.Host, subnetName string) ([]string, error) {
	ux.Logger.PrintToUser("Checking compatibility of node(s) lux go RPC protocol version with Subnet EVM RPC of blockchain %s ...", subnetName)
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return nil, err
	}
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
				if _, rpcVersion, err := ParseLuxdOutput(resp); err != nil {
					nodeResults.AddResult(host.GetCloudID(), nil, err)
				} else {
					nodeResults.AddResult(host.GetCloudID(), rpcVersion, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return nil, fmt.Errorf("failed to get rpc protocol version for node(s) %s", wgResults.GetErrorHostMap())
	}
	incompatibleNodes := []string{}
	for nodeID, rpcVersionI := range wgResults.GetResultMap() {
		rpcVersion := rpcVersionI.(uint32)
		if rpcVersion != uint32(sc.RPCVersion) {
			incompatibleNodes = append(incompatibleNodes, nodeID)
		}
	}
	if len(incompatibleNodes) > 0 {
		ux.Logger.PrintToUser(fmt.Sprintf("Compatible Lux Go RPC version is %d", sc.RPCVersion))
	}
	return incompatibleNodes, nil
}

func ParseLuxdOutput(byteValue []byte) (string, uint32, error) {
	reply := map[string]interface{}{}
	if err := json.Unmarshal(byteValue, &reply); err != nil {
		return "", 0, err
	}
	resultMap := reply["result"]
	resultJSON, err := json.Marshal(resultMap)
	if err != nil {
		return "", 0, err
	}

	nodeVersionReply := info.GetNodeVersionReply{}
	if err := json.Unmarshal(resultJSON, &nodeVersionReply); err != nil {
		return "", 0, err
	}
	return nodeVersionReply.VMVersions["platform"], uint32(nodeVersionReply.RPCProtocolVersion), nil
}

func DisconnectHosts(hosts []*models.Host) {
	for _, host := range hosts {
		_ = host.Disconnect()
	}
}

func getPublicEndpoints(
	app *application.Lux,
	clusterName string,
	trackers []*models.Host,
) ([]string, error) {
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}

	// Type assertions for map[string]interface{}
	apiNodes, _ := clusterConfig["apiNodes"].([]string)
	nodes, _ := clusterConfig["nodes"].([]string)
	networkStr, _ := clusterConfig["network"].(string)

	network := models.NetworkFromString(networkStr)
	publicNodes := apiNodes
	if network.Kind() == models.Devnet {
		publicNodes = nodes
	}
	publicTrackers := utils.Filter(trackers, func(tracker *models.Host) bool {
		return sdkutils.Belongs(publicNodes, tracker.GetCloudID())
	})
	endpoints := sdkutils.Map(publicTrackers, func(tracker *models.Host) string {
		return GetLuxdEndpoint(tracker.IP)
	})
	return endpoints, nil
}

func GetLuxdEndpoint(ip string) string {
	return fmt.Sprintf("http://%s:%d", ip, constants.LuxdAPIPort)
}

func GetUnhealthyNodes(hosts []*models.Host) ([]string, error) {
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if resp, err := ssh.RunSSHCheckHealthy(host); err != nil {
				nodeResults.AddResult(host.GetCloudID(), nil, err)
				return
			} else {
				if isHealthy, err := parseHealthyOutput(resp); err != nil {
					nodeResults.AddResult(host.GetCloudID(), nil, err)
				} else {
					nodeResults.AddResult(host.GetCloudID(), isHealthy, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return nil, fmt.Errorf("failed to get health status for node(s) %s", wgResults.GetErrorHostMap())
	}
	return utils.Filter(wgResults.GetNodeList(), func(nodeID string) bool {
		return !wgResults.GetResultMap()[nodeID].(bool)
	}), nil
}

func parseHealthyOutput(byteValue []byte) (bool, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return false, err
	}
	isHealthyInterface, ok := result["result"].(map[string]interface{})
	if ok {
		isHealthy, ok := isHealthyInterface["healthy"].(bool)
		if ok {
			return isHealthy, nil
		}
	}
	return false, fmt.Errorf("unable to parse node healthy status")
}

func WaitForHealthyCluster(
	app *application.Lux,
	clusterName string,
	timeout time.Duration,
	poolTime time.Duration,
) error {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Waiting for node(s) in cluster %s to be healthy...", clusterName)
	clustersConfig, err := app.LoadClustersConfig()
	if err != nil {
		return err
	}

	// Type assertion for clusters field
	clusters, ok := clustersConfig["clusters"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid clusters configuration")
	}

	clusterData, ok := clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster %s does not exist", clusterName)
	}

	// For now, we can't use cluster.GetValidatorHosts as clusterData is a map
	// We'll need to get all hosts from ansible inventory
	allHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}

	// Filter out API nodes - only include validator and monitor nodes
	hosts := []*models.Host{}
	clusterDataMap, _ := clusterData.(map[string]interface{})
	for _, host := range allHosts {
		// API nodes typically have a specific role set in ansible inventory
		// Include all nodes except those specifically marked as API-only
		if host.GetCloudID() != "" && !isAPIOnlyNode(clusterDataMap, *host) {
			hosts = append(hosts, host)
		}
	}
	defer DisconnectHosts(hosts)
	startTime := time.Now()
	spinSession := ux.NewUserSpinner()
	spinner := spinSession.SpinToUser("Checking if node(s) are healthy...")
	for {
		unhealthyNodes, err := GetUnhealthyNodes(hosts)
		if err != nil {
			ux.SpinFailWithError(spinner, "", err)
			return err
		}
		if len(unhealthyNodes) == 0 {
			ux.SpinComplete(spinner)
			spinSession.Stop()
			ux.Logger.GreenCheckmarkToUser("Nodes healthy after %d seconds", uint32(time.Since(startTime).Seconds()))
			return nil
		}
		if time.Since(startTime) > timeout {
			ux.SpinFailWithError(spinner, "", fmt.Errorf("cluster not healthy after %d seconds", uint32(timeout.Seconds())))
			spinSession.Stop()
			ux.Logger.PrintToUser("")
			ux.Logger.RedXToUser("Unhealthy Nodes")
			for _, failedNode := range unhealthyNodes {
				ux.Logger.PrintToUser("  " + failedNode)
			}
			ux.Logger.PrintToUser("")
			return fmt.Errorf("cluster not healthy after %d seconds", uint32(timeout.Seconds()))
		}
		time.Sleep(poolTime)
	}
}
