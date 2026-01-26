// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpccmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/mpc"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	// Deploy flags
	deployProvider     string
	deployRegion       string
	deployInstanceType string
	deploySSHKey       string
	deploySSHUser      string

	// AWS flags
	deployAWSProfile string
	deployAWSVPC     string

	// GCP flags
	deployGCPProject string
	deployGCPZone    string

	// Azure flags
	deployAzureSubscription  string
	deployAzureResourceGroup string
)

// newDeployCmd creates the deploy command group.
func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy MPC nodes to cloud infrastructure",
		Long: `Deploy MPC nodes to cloud providers for production use.

Each MPC node is deployed to a separate server for security.
Key shards are encrypted and stored securely on each node.

SUPPORTED PROVIDERS:

  aws           Amazon Web Services (EC2)
  gcp           Google Cloud Platform (Compute Engine)
  azure         Microsoft Azure (Virtual Machines)
  digitalocean  DigitalOcean (Droplets)

SECURITY CONSIDERATIONS:

  - Each node should be in a different availability zone/region
  - Key shards are encrypted with age before storage
  - SSH access is required for node management
  - Use private networks where possible

Examples:
  # Deploy to AWS
  lux mpc deploy create mpc-devnet-xxx --provider aws --region us-east-1

  # Deploy to DigitalOcean
  lux mpc deploy create mpc-devnet-xxx --provider digitalocean --region nyc1

  # Check deployment status
  lux mpc deploy status mpc-devnet-xxx

  # SSH to a specific node
  lux mpc deploy ssh mpc-devnet-xxx mpc-node-1

  # Destroy deployment
  lux mpc deploy destroy mpc-devnet-xxx`,
	}

	cmd.AddCommand(newDeployCreateCmd())
	cmd.AddCommand(newDeployStatusCmd())
	cmd.AddCommand(newDeploySSHCmd())
	cmd.AddCommand(newDeployDestroyCmd())
	cmd.AddCommand(newDeployListCmd())

	return cmd
}

func newDeployCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <network-name>",
		Short: "Deploy MPC network to cloud",
		Long: `Deploy an initialized MPC network to cloud infrastructure.

The network must be initialized first with 'lux mpc node init'.
Each node will be deployed to a separate cloud instance.`,
		Args: cobra.ExactArgs(1),
		RunE: runDeployCreate,
	}

	// Common flags
	cmd.Flags().StringVarP(&deployProvider, "provider", "p", "", "Cloud provider (aws, gcp, azure, digitalocean)")
	cmd.Flags().StringVarP(&deployRegion, "region", "r", "", "Cloud region")
	cmd.Flags().StringVar(&deployInstanceType, "instance-type", "", "Instance type (default: provider-specific)")
	cmd.Flags().StringVar(&deploySSHKey, "ssh-key", "", "Path to SSH private key")
	cmd.Flags().StringVar(&deploySSHUser, "ssh-user", "ubuntu", "SSH username")

	// AWS flags
	cmd.Flags().StringVar(&deployAWSProfile, "aws-profile", "", "AWS profile name")
	cmd.Flags().StringVar(&deployAWSVPC, "aws-vpc", "", "AWS VPC ID")

	// GCP flags
	cmd.Flags().StringVar(&deployGCPProject, "gcp-project", "", "GCP project ID")
	cmd.Flags().StringVar(&deployGCPZone, "gcp-zone", "", "GCP zone")

	// Azure flags
	cmd.Flags().StringVar(&deployAzureSubscription, "azure-subscription", "", "Azure subscription ID")
	cmd.Flags().StringVar(&deployAzureResourceGroup, "azure-resource-group", "", "Azure resource group")

	cmd.MarkFlagRequired("provider")

	return cmd
}

func newDeployStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <network-name>",
		Short: "Show deployment status",
		Args:  cobra.ExactArgs(1),
		RunE:  runDeployStatus,
	}
}

func newDeploySSHCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ssh <network-name> <node-name>",
		Short: "SSH to a deployed node",
		Long: `Open an SSH session to a deployed MPC node.

Examples:
  lux mpc deploy ssh mpc-devnet-xxx mpc-node-1`,
		Args: cobra.ExactArgs(2),
		RunE: runDeploySSH,
	}
}

func newDeployDestroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy <network-name>",
		Short: "Destroy cloud deployment",
		Long: `Terminate all cloud instances and clean up resources.

WARNING: This will delete all deployed instances!
Make sure you have backups of key shards before destroying.`,
		Args: cobra.ExactArgs(1),
		RunE: runDeployDestroy,
	}

	cmd.Flags().BoolVarP(&nodeForce, "force", "f", false, "Skip confirmation")

	return cmd
}

func newDeployListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List deployments",
		RunE:  runDeployList,
	}
}

// Command implementations

func runDeployCreate(cmd *cobra.Command, args []string) error {
	networkName := args[0]

	// Validate provider
	provider := mpc.CloudProvider(deployProvider)
	switch provider {
	case mpc.CloudProviderAWS, mpc.CloudProviderGCP, mpc.CloudProviderAzure, mpc.CloudProviderDigitalOcean:
		// Valid
	default:
		return fmt.Errorf("unsupported provider: %s (use aws, gcp, azure, or digitalocean)", deployProvider)
	}

	// Set defaults
	if deployRegion == "" {
		deployRegion = mpc.DefaultRegions()[provider]
	}
	if deployInstanceType == "" {
		deployInstanceType = mpc.DefaultInstanceTypes()[provider]
	}
	if deploySSHKey == "" {
		homeDir, _ := os.UserHomeDir()
		deploySSHKey = filepath.Join(homeDir, ".ssh", "id_rsa")
	}

	cfg := &mpc.DeploymentConfig{
		Provider:     provider,
		Region:       deployRegion,
		InstanceType: deployInstanceType,
		SSHKeyPath:   deploySSHKey,
		SSHUser:      deploySSHUser,

		AWSProfile:         deployAWSProfile,
		AWSVPC:             deployAWSVPC,
		GCPProject:         deployGCPProject,
		GCPZone:            deployGCPZone,
		AzureSubscription:  deployAzureSubscription,
		AzureResourceGroup: deployAzureResourceGroup,
	}

	ux.Logger.PrintToUser("Deploying MPC network %s to %s (%s)...", networkName, provider, deployRegion)
	ux.Logger.PrintToUser("  Instance type: %s", deployInstanceType)
	ux.Logger.PrintToUser("  SSH key:       %s", deploySSHKey)

	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".lux", "mpc")
	mgr := mpc.NewDeploymentManager(baseDir)

	_, err := mgr.DeployNetwork(cmd.Context(), networkName, cfg)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("\nDeployment started!")
	ux.Logger.PrintToUser("Check status with: lux mpc deploy status %s", networkName)

	return nil
}

func runDeployStatus(cmd *cobra.Command, args []string) error {
	networkName := args[0]

	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".lux", "mpc")
	mgr := mpc.NewDeploymentManager(baseDir)

	remoteCfg, err := mgr.LoadDeploymentConfig(networkName)
	if err != nil {
		return fmt.Errorf("deployment not found: %s", networkName)
	}

	ux.Logger.PrintToUser("Deployment: %s", networkName)
	ux.Logger.PrintToUser("Provider:   %s", remoteCfg.Deployment.Provider)
	ux.Logger.PrintToUser("Region:     %s", remoteCfg.Deployment.Region)
	ux.Logger.PrintToUser("Deployed:   %s", remoteCfg.DeployedAt.Format("2006-01-02 15:04:05"))
	ux.Logger.PrintToUser("")

	ux.Logger.PrintToUser("%-15s  %-15s  %-20s  %-10s  %-10s", "NODE", "INSTANCE", "PUBLIC IP", "STATUS", "KEYS")
	ux.Logger.PrintToUser("%s", strings.Repeat("-", 75))

	for _, node := range remoteCfg.Nodes {
		keysStatus := "locked"
		if !node.KeyEncrypted {
			keysStatus = "unlocked"
		}
		ux.Logger.PrintToUser("%-15s  %-15s  %-20s  %-10s  %-10s",
			node.NodeConfig.NodeName,
			node.InstanceID,
			node.PublicIP,
			node.Status,
			keysStatus,
		)
	}

	return nil
}

func runDeploySSH(cmd *cobra.Command, args []string) error {
	networkName := args[0]
	nodeName := args[1]

	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".lux", "mpc")
	mgr := mpc.NewDeploymentManager(baseDir)

	remoteCfg, err := mgr.LoadDeploymentConfig(networkName)
	if err != nil {
		return fmt.Errorf("deployment not found: %s", networkName)
	}

	var targetNode *mpc.RemoteNode
	for _, node := range remoteCfg.Nodes {
		if node.NodeConfig.NodeName == nodeName {
			targetNode = node
			break
		}
	}

	if targetNode == nil {
		return fmt.Errorf("node not found: %s", nodeName)
	}

	// Print SSH command for user to run
	sshCmd := fmt.Sprintf("ssh -i %s %s@%s",
		remoteCfg.Deployment.SSHKeyPath,
		remoteCfg.Deployment.SSHUser,
		targetNode.PublicIP,
	)

	ux.Logger.PrintToUser("Connect to node with:")
	ux.Logger.PrintToUser("  %s", sshCmd)

	return nil
}

func runDeployDestroy(cmd *cobra.Command, args []string) error {
	networkName := args[0]

	if !nodeForce {
		ux.Logger.PrintToUser("WARNING: This will terminate all cloud instances for %s", networkName)
		ux.Logger.PrintToUser("Use --force to confirm")
		return nil
	}

	ux.Logger.PrintToUser("Destroying deployment %s...", networkName)

	// TODO: Implement actual cloud resource cleanup
	ux.Logger.PrintToUser("Cloud resource cleanup not yet implemented")

	return nil
}

func runDeployList(cmd *cobra.Command, args []string) error {
	mgr := getNodeManager()

	networks, err := mgr.ListNetworks()
	if err != nil {
		return err
	}

	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".lux", "mpc")
	deployMgr := mpc.NewDeploymentManager(baseDir)

	ux.Logger.PrintToUser("%-25s  %-12s  %-15s  %-10s  %-8s", "NETWORK", "PROVIDER", "REGION", "STATUS", "NODES")
	ux.Logger.PrintToUser("%s", strings.Repeat("-", 75))

	for _, net := range networks {
		provider := "local"
		region := "-"
		status := "local"
		nodeCount := len(net.Nodes)

		// Check if deployed
		if remoteCfg, err := deployMgr.LoadDeploymentConfig(net.NetworkName); err == nil {
			provider = string(remoteCfg.Deployment.Provider)
			region = remoteCfg.Deployment.Region
			status = "deployed"
		}

		ux.Logger.PrintToUser("%-25s  %-12s  %-15s  %-10s  %-8d",
			net.NetworkName,
			provider,
			region,
			status,
			nodeCount,
		)
	}

	return nil
}
