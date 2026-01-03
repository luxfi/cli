// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/luxfi/cli/pkg/ansible"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/models"
)

func SyncSubnet(app *application.Lux, clusterName, blockchainName string, avoidChecks bool, subnetAliases []string) error {
	if err := CheckCluster(app, clusterName); err != nil {
		return err
	}
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	if err := chain.ValidateSubnetNameAndGetChains(blockchainName); err != nil {
		return err
	}
	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	defer DisconnectHosts(hosts)
	if !avoidChecks {
		if err := CheckHostsAreBootstrapped(hosts); err != nil {
			return err
		}
		if err := CheckHostsAreHealthy(hosts); err != nil {
			return err
		}
		if err := CheckHostsAreRPCCompatible(app, hosts, blockchainName); err != nil {
			return err
		}
	}
	if err := prepareSubnetPlugin(app, hosts, blockchainName); err != nil {
		return err
	}
	// Type assertion for network field
	networkStr, _ := clusterConfig["network"].(string)
	network := models.NetworkFromString(networkStr)
	if err := trackSubnet(app, hosts, clusterName, network, blockchainName, subnetAliases); err != nil {
		return err
	}
	ux.Logger.PrintToUser("Node(s) successfully started syncing with blockchain!")
	ux.Logger.PrintToUser("%s", fmt.Sprintf("Check node blockchain syncing status with lux node status %s --blockchain %s", clusterName, blockchainName))
	return nil
}

// prepareSubnetPlugin creates subnet plugin to all nodes in the cluster
func prepareSubnetPlugin(app *application.Lux, hosts []*models.Host, blockchainName string) error {
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if err := ssh.RunSSHCreatePlugin(host, sc); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return fmt.Errorf("failed to upload plugin to node(s) %s", wgResults.GetErrorHostMap())
	}
	return nil
}

func trackSubnet(
	app *application.Lux,
	hosts []*models.Host,
	clusterName string,
	network models.Network,
	blockchainName string,
	subnetAliases []string,
) error {
	// load cluster config
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	// and get list of subnets
	subnets, _ := clusterConfig["subnets"].([]string)
	allSubnets := utils.Unique(append(subnets, blockchainName))

	// load sidecar to get subnet blockchain ID
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}
	blockchainID := sc.Networks[network.Name()].BlockchainID

	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if err := ssh.RunSSHStopNode(host); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
			}

			if err := ssh.RunSSHRenderLuxdAliasConfigFile(
				host,
				blockchainID.String(),
				subnetAliases,
			); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
			}
			// Check if this host is an API host
			apiNodes, _ := clusterConfig["apiNodes"].([]string)
			isAPIHost := false
			for _, apiNode := range apiNodes {
				if apiNode == host.GetCloudID() {
					isAPIHost = true
					break
				}
			}

			if err := ssh.RunSSHRenderLuxNodeConfig(
				app,
				host,
				network,
				allSubnets,
				isAPIHost,
			); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
			}
			if err := ssh.RunSSHSyncSubnetData(app, host, network, blockchainName); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
			}
			if err := ssh.RunSSHStartNode(host); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
				return
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return fmt.Errorf("failed to track network for node(s) %s", wgResults.GetErrorHostMap())
	}

	// update slice of subnets synced by the cluster
	clusterConfig["subnets"] = allSubnets
	// Save the updated cluster configuration
	if err := app.SetClusterConfig(clusterName, clusterConfig); err != nil {
		return fmt.Errorf("failed to save cluster config: %w", err)
	}

	// update slice of blockchain endpoints with the cluster ones
	// Type assertion for network field
	networkStr, _ := clusterConfig["network"].(string)
	network = models.NetworkFromString(networkStr)
	networkInfo := sc.Networks[network.Name()]
	rpcEndpoints := set.Of(networkInfo.RPCEndpoints...)
	wsEndpoints := set.Of(networkInfo.WSEndpoints...)
	publicEndpoints, err := getPublicEndpoints(app, clusterName, hosts)
	if err != nil {
		return err
	}
	for _, publicEndpoint := range publicEndpoints {
		rpcEndpoints.Add(models.GetRPCEndpoint(publicEndpoint, networkInfo.BlockchainID.String()))
		wsEndpoints.Add(models.GetWSEndpoint(publicEndpoint, networkInfo.BlockchainID.String()))
	}
	networkInfo.RPCEndpoints = rpcEndpoints.List()
	networkInfo.WSEndpoints = wsEndpoints.List()
	sc.Networks[network.Name()] = networkInfo
	return app.UpdateSidecar(&sc)
}

func CheckHostsAreBootstrapped(hosts []*models.Host) error {
	notBootstrappedNodes, err := GetNotBootstrappedNodes(hosts)
	if err != nil {
		return err
	}
	if len(notBootstrappedNodes) > 0 {
		return fmt.Errorf("node(s) %s are not bootstrapped yet, please try again later", notBootstrappedNodes)
	}
	return nil
}

func CheckHostsAreHealthy(hosts []*models.Host) error {
	ux.Logger.PrintToUser("Checking if node(s) are healthy...")
	unhealthyNodes, err := GetUnhealthyNodes(hosts)
	if err != nil {
		return err
	}
	if len(unhealthyNodes) > 0 {
		return fmt.Errorf("node(s) %s are not healthy, please check the issue and try again later", unhealthyNodes)
	}
	return nil
}

func GetNotBootstrappedNodes(hosts []*models.Host) ([]string, error) {
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if resp, err := ssh.RunSSHCheckBootstrapped(host); err != nil {
				nodeResults.AddResult(host.GetCloudID(), nil, err)
				return
			} else {
				if isBootstrapped, err := parseBootstrappedOutput(resp); err != nil {
					nodeResults.AddResult(host.GetCloudID(), nil, err)
				} else {
					nodeResults.AddResult(host.GetCloudID(), isBootstrapped, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return nil, fmt.Errorf("failed to get luxd bootstrap status for node(s) %s", wgResults.GetErrorHostMap())
	}
	return utils.Filter(wgResults.GetNodeList(), func(nodeID string) bool {
		return !wgResults.GetResultMap()[nodeID].(bool)
	}), nil
}

func parseBootstrappedOutput(byteValue []byte) (bool, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return false, err
	}
	isBootstrappedInterface, ok := result["result"].(map[string]interface{})
	if ok {
		isBootstrapped, ok := isBootstrappedInterface["isBootstrapped"].(bool)
		if ok {
			return isBootstrapped, nil
		}
	}
	return false, errors.New("unable to parse node bootstrap status")
}
