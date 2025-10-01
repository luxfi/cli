// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/node"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	sdkconstants "github.com/luxfi/sdk/constants"
	"github.com/luxfi/sdk/models"

	"github.com/spf13/cobra"
)

var (
	clusterFileName string
	force           bool
	includeSecrets  bool
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [clusterName]",
		Short: "(ALPHA Warning) Export cluster configuration to a file",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node export command exports cluster configuration and its nodes config to a text file.

If no file is specified, the configuration is printed to the stdout.

Use --include-secrets to include keys in the export. In this case please keep the file secure as it contains sensitive information.

Exported cluster configuration without secrets can be imported by another user using node import command.`,
		Args: cobrautils.ExactArgs(1),
		RunE: exportFile,
	}
	cmd.Flags().StringVar(&clusterFileName, "file", "", "specify the file to export the cluster configuration to")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite the file if it exists")
	cmd.Flags().BoolVar(&includeSecrets, "include-secrets", false, "include keys in the export")
	return cmd
}

func exportFile(_ *cobra.Command, args []string) error {
	clusterName := args[0]
	if clusterFileName != "" && utils.FileExists(utils.ExpandHome(clusterFileName)) && !force {
		ux.Logger.RedXToUser("file already exists, use --force to overwrite")
		return nil
	}
	if err := node.CheckCluster(app, clusterName); err != nil {
		ux.Logger.RedXToUser("cluster not found: %v", err)
		return err
	}
	clusterConf, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	// clusterConf is a map[string]interface{}, not a struct
	if network, ok := clusterConf["Network"].(map[string]interface{}); ok {
		network["ClusterName"] = "" // hide cluster name
	}
	clusterConf["External"] = true // mark cluster as external

	// Get the nodes list
	nodesList, _ := clusterConf["Nodes"].([]string)
	exportNodes, err := utils.MapWithError(nodesList, func(nodeName string) (models.ExportNode, error) {
		var err error
		nodeConf, err := app.LoadClusterNodeConfig(clusterName, nodeName)
		if err != nil {
			return models.ExportNode{}, err
		}
		// Hide sensitive information
		nodeConf["CertPath"] = ""
		nodeConf["SecurityGroup"] = ""
		nodeConf["KeyPair"] = ""

		nodeID, _ := nodeConf["NodeID"].(string)
		signerKey, stakerKey, stakerCrt, err := readKeys(filepath.Join(app.GetNodesDir(), nodeID))
		if err != nil {
			return models.ExportNode{}, err
		}
		// Convert map to NodeConfig struct
		nc := models.NodeConfig{
			NodeID:        nodeID,
			Region:        nodeConf["Region"].(string),
			AMI:           nodeConf["AMI"].(string),
			KeyPair:       "", // Already cleared
			CertPath:      "", // Already cleared
			SecurityGroup: "", // Already cleared
			ElasticIP:     nodeConf["ElasticIP"].(string),
			CloudService:  nodeConf["CloudService"].(string),
			UseStaticIP:   nodeConf["UseStaticIP"].(bool),
			IsMonitor:     nodeConf["IsMonitor"].(bool),
			IsWarpRelayer: nodeConf["IsWarpRelayer"].(bool),
			IsLoadTest:    nodeConf["IsLoadTest"].(bool),
		}
		return models.ExportNode{
			NodeConfig: nc,
			SignerKey:  signerKey,
			StakerKey:  stakerKey,
			StakerCrt:  stakerCrt,
		}, nil
	})
	if err != nil {
		ux.Logger.RedXToUser("could not load node configuration: %v", err)
		return err
	}
	// monitoring instance
	monitor := models.ExportNode{}
	monitoringInstance, _ := clusterConf["MonitoringInstance"].(string)
	if monitoringInstance != "" {
		monitoringHost, err := app.LoadClusterNodeConfig(clusterName, monitoringInstance)
		if err != nil {
			ux.Logger.RedXToUser("could not load monitoring host configuration: %v", err)
			return err
		}
		// Hide sensitive information
		monitoringHost["CertPath"] = ""
		monitoringHost["SecurityGroup"] = ""
		monitoringHost["KeyPair"] = ""

		// Convert map to NodeConfig struct
		monitorNC := models.NodeConfig{
			NodeID:        monitoringHost["NodeID"].(string),
			Region:        monitoringHost["Region"].(string),
			AMI:           monitoringHost["AMI"].(string),
			KeyPair:       "", // Already cleared
			CertPath:      "", // Already cleared
			SecurityGroup: "", // Already cleared
			ElasticIP:     monitoringHost["ElasticIP"].(string),
			CloudService:  monitoringHost["CloudService"].(string),
			UseStaticIP:   monitoringHost["UseStaticIP"].(bool),
			IsMonitor:     true,
			IsWarpRelayer: monitoringHost["IsWarpRelayer"].(bool),
			IsLoadTest:    monitoringHost["IsLoadTest"].(bool),
		}
		monitor = models.ExportNode{
			NodeConfig: monitorNC,
			SignerKey:  "",
			StakerKey:  "",
			StakerCrt:  "",
		}
	}
	// loadtest nodes
	loadTestNodes := []models.ExportNode{}
	loadTestInstances, _ := clusterConf["LoadTestInstance"].([]string)
	for _, loadTestNode := range loadTestInstances {
		loadTestNodeConf, err := app.LoadClusterNodeConfig(clusterName, loadTestNode)
		if err != nil {
			ux.Logger.RedXToUser("could not load load test node configuration: %v", err)
			return err
		}
		// Hide sensitive information
		loadTestNodeConf["CertPath"] = ""
		loadTestNodeConf["SecurityGroup"] = ""
		loadTestNodeConf["KeyPair"] = ""

		// Convert map to NodeConfig struct
		ltNC := models.NodeConfig{
			NodeID:        loadTestNodeConf["NodeID"].(string),
			Region:        loadTestNodeConf["Region"].(string),
			AMI:           loadTestNodeConf["AMI"].(string),
			KeyPair:       "", // Already cleared
			CertPath:      "", // Already cleared
			SecurityGroup: "", // Already cleared
			ElasticIP:     loadTestNodeConf["ElasticIP"].(string),
			CloudService:  loadTestNodeConf["CloudService"].(string),
			UseStaticIP:   loadTestNodeConf["UseStaticIP"].(bool),
			IsMonitor:     loadTestNodeConf["IsMonitor"].(bool),
			IsWarpRelayer: loadTestNodeConf["IsWarpRelayer"].(bool),
			IsLoadTest:    true,
		}
		loadTestNodes = append(loadTestNodes, models.ExportNode{
			NodeConfig: ltNC,
			SignerKey:  "",
			StakerKey:  "",
			StakerCrt:  "",
		})
	}

	// Convert clusterConf map to ClusterConfig struct
	nodes, _ := clusterConf["Nodes"].([]string)
	apiNodes, _ := clusterConf["APINodes"].([]string)
	subnets, _ := clusterConf["Subnets"].([]string)
	external, _ := clusterConf["External"].(bool)
	local, _ := clusterConf["Local"].(bool)
	httpAccess, _ := clusterConf["HTTPAccess"].(string)

	// Handle Network field
	var network models.Network
	if networkData, ok := clusterConf["Network"].(map[string]interface{}); ok {
		// Try to reconstruct the network
		if kind, ok := networkData["Kind"].(string); ok {
			switch kind {
			case "Mainnet":
				network = models.NewMainnetNetwork()
			case "Testnet":
				network = models.NewTestnetNetwork()
			case "Devnet":
				network = models.NewDevnetNetwork()
			default:
				network = models.NewLocalNetwork()
			}
		}
	} else if net, ok := clusterConf["Network"].(models.Network); ok {
		network = net
	}

	// Handle LoadTestInstance
	loadTestMap := make(map[string]string)
	if ltInst, ok := clusterConf["LoadTestInstance"].(map[string]string); ok {
		loadTestMap = ltInst
	}

	// Handle ExtraNetworkData
	extraNetworkData := models.ExtraNetworkData{}
	if extra, ok := clusterConf["ExtraNetworkData"].(models.ExtraNetworkData); ok {
		extraNetworkData = extra
	}

	cc := models.ClusterConfig{
		Nodes:              nodes,
		APINodes:           apiNodes,
		Network:            network,
		MonitoringInstance: monitoringInstance,
		LoadTestInstance:   loadTestMap,
		ExtraNetworkData:   extraNetworkData,
		Subnets:            subnets,
		External:           external,
		Local:              local,
		HTTPAccess:         sdkconstants.HTTPAccess(httpAccess),
	}

	exportCluster := models.ExportCluster{
		ClusterConfig: cc,
		Nodes:         exportNodes,
		MonitorNode:   monitor,
		LoadTestNodes: loadTestNodes,
	}
	if clusterFileName != "" {
		outFile, err := os.Create(utils.ExpandHome(clusterFileName))
		if err != nil {
			ux.Logger.RedXToUser("could not create file: %v", err)
			return err
		}
		defer outFile.Close()
		if err := writeExportFile(exportCluster, outFile); err != nil {
			ux.Logger.RedXToUser("could not write to file: %v", err)
			return err
		}
		ux.Logger.GreenCheckmarkToUser("exported cluster [%s] configuration to %s", clusterName, utils.ExpandHome(outFile.Name()))
	} else {
		if err := writeExportFile(exportCluster, os.Stdout); err != nil {
			ux.Logger.RedXToUser("could not write to stdout: %v", err)
			return err
		}
	}
	return nil
}

// readKeys reads the keys from the node configuration
func readKeys(nodeConfPath string) (string, string, string, error) {
	stakerCrt, err := utils.ReadFile(filepath.Join(nodeConfPath, constants.StakerCertFileName))
	if err != nil {
		return "", "", "", err
	}
	if !includeSecrets {
		return "", "", stakerCrt, nil // return only the certificate
	}
	signerKey, err := utils.ReadFile(filepath.Join(nodeConfPath, constants.BLSKeyFileName))
	if err != nil {
		return "", "", "", err
	}
	stakerKey, err := utils.ReadFile(filepath.Join(nodeConfPath, constants.StakerKeyFileName))
	if err != nil {
		return "", "", "", err
	}

	return signerKey, stakerKey, stakerCrt, nil
}

// writeExportFile writes the exportCluster to the out writer
func writeExportFile(exportCluster models.ExportCluster, out io.Writer) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportCluster); err != nil {
		return err
	}
	return nil
}
