// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/luxfi/cli/pkg/ansible"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/node"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	sdkutils "github.com/luxfi/sdk/utils"
	luxlog "github.com/luxfi/log"

	"github.com/spf13/cobra"
)

var (
	isParallel      bool
	includeMonitor  bool
	includeLoadTest bool
)

func newSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh [clusterName|nodeID|instanceID|IP] [cmd]",
		Short: "(ALPHA Warning) Execute ssh command on node(s)",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node ssh command execute a given command [cmd] using ssh on all nodes in the cluster if ClusterName is given.
If no command is given, just prints the ssh command to be used to connect to each node in the cluster.
For provided NodeID or InstanceID or IP, the command [cmd] will be executed on that node.
If no [cmd] is provided for the node, it will open ssh shell there.
`,
		Args: cobrautils.MinimumNArgs(0),
		RunE: sshNode,
	}
	cmd.Flags().BoolVar(&isParallel, "parallel", false, "run ssh command on all nodes in parallel")
	cmd.Flags().BoolVar(&includeMonitor, "with-monitor", false, "include monitoring node for ssh cluster operations")
	cmd.Flags().BoolVar(&includeLoadTest, "with-loadtest", false, "include loadtest node for ssh cluster operations")

	return cmd
}

func sshNode(_ *cobra.Command, args []string) error {
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return err
	}
	// clustersConfig is a map[string]interface{}, not a struct
	clusters, ok := clustersConfig["Clusters"].(map[string]interface{})
	if !ok || len(clusters) == 0 {
		ux.Logger.PrintToUser("There are no clusters defined.")
		return nil
	}
	if len(args) == 0 {
		// provide ssh connection string for all clusters
		for clusterName, clusterConfigInterface := range clusters {
			clusterConfig, ok := clusterConfigInterface.(map[string]interface{})
			if !ok {
				continue
			}
			if isLocal, ok := clusterConfig["Local"].(bool); ok && isLocal {
				continue
			}
			// Get network kind if available
			networkKind := ""
			if network, ok := clusterConfig["Network"].(map[string]interface{}); ok {
				if kind, ok := network["Kind"].(string); ok {
					networkKind = kind
				}
			}
			err := printClusterConnectionString(clusterName, networkKind)
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		clusterNameOrNodeID := args[0]
		cmd := strings.Join(args[1:], " ")
		if err := node.CheckCluster(app, clusterNameOrNodeID); err == nil {
			// clusterName detected
			if len(args[1:]) == 0 {
				// clustersConfig is a map[string]interface{}, not a struct
				clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
				cluster, _ := clusters[clusterNameOrNodeID].(map[string]interface{})
				network, _ := cluster["Network"].(map[string]interface{})
				kind, _ := network["Kind"].(string)
				return printClusterConnectionString(clusterNameOrNodeID, kind)
			} else {
				clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
				cluster, _ := clusters[clusterNameOrNodeID].(map[string]interface{})
				if local, ok := cluster["Local"].(bool); ok && local {
					return notImplementedForLocal("ssh")
				}
				clusterHosts, err := GetAllClusterHosts(clusterNameOrNodeID)
				if err != nil {
					return err
				}
				return sshHosts(clusterHosts, cmd, cluster)
			}
		} else {
			// try to detect nodeID
			selectedHost, clusterName := getHostClusterPair(clusterNameOrNodeID)
			if selectedHost != nil && clusterName != "" {
				clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
				cluster, _ := clusters[clusterName].(map[string]interface{})
				return sshHosts([]*models.Host{selectedHost}, cmd, cluster)
			}
		}
		return fmt.Errorf("cluster or node %s not found", clusterNameOrNodeID)
	}
}

func printNodeInfo(host *models.Host, clusterConf map[string]interface{}, result string) error {
	// Extract clusterName from clusterConf (need to find it)
	clusterName := ""
	clustersConfig, _ := app.GetClustersConfig()
	clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
	for name, cluster := range clusters {
		if c, ok := cluster.(map[string]interface{}); ok {
			// Check if this cluster contains our host
			if hosts, ok := c["Nodes"].([]interface{}); ok {
				for _, h := range hosts {
					if h == host.GetCloudID() {
						clusterName = name
						break
					}
				}
			}
		}
		if clusterName != "" {
			break
		}
	}
	nodeConfig, err := app.LoadClusterNodeConfig(clusterName, host.GetCloudID())
	if err != nil {
		return err
	}
	nodeIDStr := "----------------------------------------"
	// Check if host is a luxd host (typically all hosts are luxd hosts unless they're monitoring or loadtest only)
	isLuxdHost := true
	if monitor, ok := nodeConfig["IsMonitor"].(bool); ok && monitor {
		if relayer, ok := nodeConfig["IsWarpRelayer"].(bool); !ok || !relayer {
			if loadtest, ok := nodeConfig["IsLoadTest"].(bool); !ok || !loadtest {
				// Only monitor, not a luxd host
				isLuxdHost = false
			}
		}
	}
	if isLuxdHost {
		nodeID, err := getNodeID(app.GetNodeInstanceDirPath(host.GetCloudID()))
		if err != nil {
			return err
		}
		nodeIDStr = nodeID.String()
	}
	// Map access for clusterConf
	elasticIP, _ := nodeConfig["ElasticIP"].(string)
	roles := []string{}
	if monitor, ok := nodeConfig["IsMonitor"].(bool); ok && monitor {
		roles = append(roles, "Monitor")
	}
	if relayer, ok := nodeConfig["IsWarpRelayer"].(bool); ok && relayer {
		roles = append(roles, "WarpRelayer")
	}
	if loadtest, ok := nodeConfig["IsLoadTest"].(bool); ok && loadtest {
		roles = append(roles, "LoadTest")
	}
	rolesStr := strings.Join(roles, ",")
	if rolesStr != "" {
		rolesStr = " [" + rolesStr + "]"
	}
	ux.Logger.PrintToUser("  [Node %s (%s) %s%s] %s", host.GetCloudID(), nodeIDStr, elasticIP, rolesStr, result)
	return nil
}

func sshHosts(hosts []*models.Host, cmd string, clusterConf map[string]interface{}) error {
	if cmd != "" {
		// execute cmd
		wg := sync.WaitGroup{}
		nowExecutingMutex := sync.Mutex{}
		wgResults := models.NodeResults{}
		for _, host := range hosts {
			wg.Add(1)
			go func(nodeResults *models.NodeResults, host *models.Host) {
				if !isParallel {
					nowExecutingMutex.Lock()
					defer nowExecutingMutex.Unlock()
					if err := printNodeInfo(host, clusterConf, ""); err != nil {
						ux.Logger.RedXToUser("Error getting node %s info due to : %s", host.GetCloudID(), err)
					}
				}
				defer wg.Done()
				cmd := utils.Command(utils.GetSSHConnectionString(host.IP, host.SSHPrivateKeyPath), cmd)
				outBuf, errBuf := utils.SetupRealtimeCLIOutput(cmd, false, false)
				if !isParallel {
					_, _ = utils.SetupRealtimeCLIOutput(cmd, true, true)
				}
				if _, err := outBuf.ReadFrom(errBuf); err != nil {
					nodeResults.AddResult(host.NodeID, outBuf, err)
				}
				if err := cmd.Run(); err != nil {
					nodeResults.AddResult(host.NodeID, outBuf, err)
				} else {
					nodeResults.AddResult(host.NodeID, outBuf, nil)
				}
			}(&wgResults, host)
		}
		wg.Wait()
		if wgResults.HasErrors() {
			return fmt.Errorf("failed to ssh node(s) %s", wgResults.GetErrorHostMap())
		}
		if isParallel {
			for hostID, result := range wgResults.GetResultMap() {
				for _, host := range hosts {
					if host.GetCloudID() == hostID {
						if err := printNodeInfo(host, clusterConf, fmt.Sprintf("%v", result)); err != nil {
							ux.Logger.RedXToUser("Error getting node %s info due to : %s", host.GetCloudID(), err)
						}
					}
				}
			}
		}
	} else {
		// open shell
		switch {
		case len(hosts) > 1:
			return fmt.Errorf("cannot open ssh shell on multiple nodes: %s", strings.Join(sdkutils.Map(hosts, func(h *models.Host) string { return h.GetCloudID() }), ", "))
		case len(hosts) == 0:
			return fmt.Errorf("no nodes found")
		default:
			selectedHost := hosts[0]
			splitCmdLine := strings.Split(utils.GetSSHConnectionString(selectedHost.IP, selectedHost.SSHPrivateKeyPath), " ")
			cmd := exec.Command(splitCmdLine[0], splitCmdLine[1:]...)
			cmd.Env = os.Environ()
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				ux.Logger.PrintToUser("Error: %s", err)
				return err
			}
			ux.Logger.PrintToUser("[%s] shell closed to %s", selectedHost.GetCloudID(), selectedHost.IP)
		}
	}
	return nil
}

func printClusterConnectionString(clusterName string, networkName string) error {
	clusterConf, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	// clusterConf is a map[string]interface{}, not a struct
	if external, ok := clusterConf["External"].(bool); ok && external {
		ux.Logger.PrintToUser("Cluster: %s (%s) EXTERNAL", luxlog.LightBlue.Wrap(clusterName), luxlog.Green.Wrap(networkName))
	} else {
		ux.Logger.PrintToUser("Cluster: %s (%s)", luxlog.LightBlue.Wrap(clusterName), luxlog.Green.Wrap(networkName))
	}
	clusterHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	monitoringInventoryPath := app.GetMonitoringInventoryDir(clusterName)
	if sdkutils.DirExists(monitoringInventoryPath) {
		monitoringHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(monitoringInventoryPath)
		if err != nil {
			return err
		}
		clusterHosts = append(clusterHosts, monitoringHosts...)
	}
	for _, host := range clusterHosts {
		ux.Logger.PrintToUser(utils.GetSSHConnectionString(host.IP, host.SSHPrivateKeyPath))
	}
	ux.Logger.PrintToUser("")
	return nil
}

// GetAllClusterHosts returns all hosts in a cluster including loadtest and monitoring hosts
func GetAllClusterHosts(clusterName string) ([]*models.Host, error) {
	if exists, err := node.CheckClusterExists(app, clusterName); err != nil || !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}
	clusterHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return nil, err
	}
	monitoringInventoryPath := app.GetMonitoringInventoryDir(clusterName)
	if includeMonitor && sdkutils.DirExists(monitoringInventoryPath) {
		monitoringHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(monitoringInventoryPath)
		if err != nil {
			return nil, err
		}
		clusterHosts = append(clusterHosts, monitoringHosts...)
	}
	loadTestInventoryPath := filepath.Join(app.GetAnsibleInventoryDirPath(clusterName), constants.LoadTestDir)
	if includeLoadTest && sdkutils.DirExists(loadTestInventoryPath) {
		loadTestHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(loadTestInventoryPath)
		if err != nil {
			return nil, err
		}
		clusterHosts = append(clusterHosts, loadTestHosts...)
	}
	return clusterHosts, nil
}
