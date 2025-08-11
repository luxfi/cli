// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/luxfi/cli/pkg/node"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/ux"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "(ALPHA Warning) List all clusters together with their nodes",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node list command lists all clusters together with their nodes.`,
		Args: cobrautils.ExactArgs(0),
		RunE: list,
	}

	return cmd
}

func list(_ *cobra.Command, _ []string) error {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	clusters, ok := clustersConfig["Clusters"].(map[string]interface{})
	if !ok || len(clusters) == 0 {
		ux.Logger.PrintToUser("There are no clusters defined.")
	}
	clusterNames := maps.Keys(clusters)
	sort.Strings(clusterNames)
	for _, clusterName := range clusterNames {
		clusterConf := clusters[clusterName].(map[string]interface{})
		if err := node.CheckCluster(app, clusterName); err != nil {
			return err
		}
		// Get cloud IDs from the Nodes list
		nodes, _ := clusterConf["Nodes"].([]string)
		nodeIDs := []string{}
		for _, cloudID := range nodes {
			nodeIDStr := "----------------------------------------"
			// Check if this is a luxd host (has staking files)
			stakingPath := filepath.Join(app.GetNodeInstanceDirPath(cloudID), "staker.crt")
			if _, err := os.Stat(stakingPath); err == nil {
				if nodeID, err := getNodeID(app.GetNodeInstanceDirPath(cloudID)); err != nil {
					ux.Logger.RedXToUser("could not obtain node ID for nodes %s: %s", cloudID, err)
				} else {
					nodeIDStr = nodeID.String()
				}
			}
			nodeIDs = append(nodeIDs, nodeIDStr)
		}
		
		// Get network info
		networkKind := "Unknown"
		if network, ok := clusterConf["Network"].(map[string]interface{}); ok {
			if kind, ok := network["Kind"].(string); ok {
				networkKind = kind
			}
		}
		
		external, _ := clusterConf["External"].(bool)
		local, _ := clusterConf["Local"].(bool)
		
		switch {
		case external:
			ux.Logger.PrintToUser("cluster %q (%s) EXTERNAL", clusterName, networkKind)
		case local:
			ux.Logger.PrintToUser("cluster %q (%s) LOCAL", clusterName, networkKind)
		default:
			ux.Logger.PrintToUser("Cluster %q (%s)", clusterName, networkKind)
		}
		
		for i, cloudID := range nodes {
			nodeConfig, err := app.LoadClusterNodeConfig(clusterName, cloudID)
			if err != nil {
				return err
			}
			
			// Determine roles
			roles := []string{}
			apiNodes, _ := clusterConf["APINodes"].([]string)
			for _, apiNode := range apiNodes {
				if apiNode == cloudID {
					roles = append(roles, "API")
					break
				}
			}
			
			// Check if it's a monitor or load test node
			if isMonitor, _ := nodeConfig["IsMonitor"].(bool); isMonitor {
				roles = append(roles, "Monitor")
			}
			if isLoadTest, _ := nodeConfig["IsLoadTest"].(bool); isLoadTest {
				roles = append(roles, "LoadTest")
			}
			
			rolesStr := strings.Join(roles, ",")
			if rolesStr != "" {
				rolesStr = " [" + rolesStr + "]"
			}
			
			elasticIP, _ := nodeConfig["ElasticIP"].(string)
			ux.Logger.PrintToUser("  Node %s (%s) %s%s", cloudID, nodeIDs[i], elasticIP, rolesStr)
		}
	}
	return nil
}
