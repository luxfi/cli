// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"sync"

	"github.com/luxfi/cli/pkg/node"

	"github.com/luxfi/cli/cmd/blockchaincmd"
	"github.com/luxfi/cli/pkg/ansible"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

func newUpdateSubnetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subnet [clusterName] [subnetName]",
		Short: "(ALPHA Warning) Update nodes in a cluster with latest subnet configuration and VM for custom VM",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node update subnet command updates all nodes in a cluster with latest Subnet configuration and VM for custom VM.
You can check the updated subnet bootstrap status by calling lux node status <clusterName> --subnet <subnetName>`,
		Args: cobrautils.ExactArgs(2),
		RunE: updateSubnet,
	}

	return cmd
}

func updateSubnet(_ *cobra.Command, args []string) error {
	clusterName := args[0]
	subnetName := args[1]
	if err := node.CheckCluster(app, clusterName); err != nil {
		return err
	}
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	// clusterConfig is a map[string]interface{}, not a struct
	if local, ok := clusterConfig["Local"].(bool); ok && local {
		return notImplementedForLocal("update")
	}
	if _, err := blockchaincmd.ValidateSubnetNameAndGetChains([]string{subnetName}); err != nil {
		return err
	}
	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	defer node.DisconnectHosts(hosts)
	if err := node.CheckHostsAreBootstrapped(hosts); err != nil {
		return err
	}
	if err := node.CheckHostsAreHealthy(hosts); err != nil {
		return err
	}
	if err := node.CheckHostsAreRPCCompatible(app, hosts, subnetName); err != nil {
		return err
	}
	// Extract network from clusterConfig
	networkMap, _ := clusterConfig["Network"].(map[string]interface{})
	// Create a NetworkInfo struct instead
	type NetworkInfo struct {
		Endpoint    string
		ClusterName string
		Kind        string
	}
	network := NetworkInfo{
		Endpoint:    networkMap["Endpoint"].(string),
		ClusterName: networkMap["ClusterName"].(string),
	}
	if kind, ok := networkMap["Kind"].(string); ok {
		network.Kind = kind
	}
	nonUpdatedNodes, err := doUpdateSubnet(hosts, clusterName, network, subnetName)
	if err != nil {
		return err
	}
	if len(nonUpdatedNodes) > 0 {
		return fmt.Errorf("node(s) %s failed to be updated for subnet %s", nonUpdatedNodes, subnetName)
	}
	ux.Logger.PrintToUser("Node(s) successfully updated for Subnet!")
	ux.Logger.PrintToUser("%s", fmt.Sprintf("Check node subnet status with lux node status %s --subnet %s", clusterName, subnetName))
	return nil
}

// NetworkInfo holds network configuration
type NetworkInfo struct {
	Endpoint    string
	ClusterName string
	Kind        string
}

// doUpdateSubnet exports deployed subnet in user's local machine to cloud server and calls node to
// restart tracking the specified subnet (similar to lux blockchain join <subnetName> command)
func doUpdateSubnet(
	hosts []*models.Host,
	clusterName string,
	networkInfo interface{}, // Accept either models.Network or NetworkInfo
	subnetName string,
) ([]string, error) {
	// Convert networkInfo to models.Network
	var network models.Network
	switch n := networkInfo.(type) {
	case models.Network:
		network = n
	case NetworkInfo:
		// Convert NetworkInfo to models.Network
		switch n.Kind {
		case "Mainnet":
			network = models.Mainnet
		case "Testnet":
			network = models.Testnet
		case "Local":
			network = models.Local
		case "Devnet":
			network = models.Devnet
		default:
			network = models.Undefined
		}
	default:
		return nil, fmt.Errorf("unsupported network type")
	}

	// load cluster config
	clusterConf, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}
	// and get list of subnets
	// clusterConf is a map[string]interface{}, not a struct
	existingSubnets, _ := clusterConf["Subnets"].([]interface{})
	subnetsList := []string{}
	for _, s := range existingSubnets {
		if subnet, ok := s.(string); ok {
			subnetsList = append(subnetsList, subnet)
		}
	}
	allSubnets := utils.Unique(append(subnetsList, subnetName))

	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if err := ssh.RunSSHStopNode(host); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
			}
			if err := ssh.RunSSHRenderLuxNodeConfig(
				app,
				host,
				network,
				allSubnets,
				false, // IsAPIHost - simplified for now, would need to check if host is in APINodes list
			); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
			}
			if err := ssh.RunSSHSyncSubnetData(app, host, network, subnetName); err != nil {
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
		return nil, fmt.Errorf("failed to update subnet for node(s) %s", wgResults.GetErrorHostMap())
	}
	return wgResults.GetErrorHosts(), nil
}
