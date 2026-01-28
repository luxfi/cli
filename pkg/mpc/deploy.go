// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/sdk/models"
)

// CloudProvider represents a cloud provider type.
type CloudProvider string

const (
	CloudProviderLocal        CloudProvider = "local"
	CloudProviderAWS          CloudProvider = "aws"
	CloudProviderGCP          CloudProvider = "gcp"
	CloudProviderAzure        CloudProvider = "azure"
	CloudProviderDigitalOcean CloudProvider = "digitalocean"
)

// DeploymentConfig holds cloud deployment configuration.
type DeploymentConfig struct {
	Provider     CloudProvider `json:"provider"`
	Region       string        `json:"region"`
	InstanceType string        `json:"instanceType"`
	SSHKeyPath   string        `json:"sshKeyPath"`
	SSHKeyName   string        `json:"sshKeyName"`
	SSHUser      string        `json:"sshUser"`

	// AWS specific
	AWSProfile       string `json:"awsProfile,omitempty"`
	AWSSecurityGroup string `json:"awsSecurityGroup,omitempty"`
	AWSVPC           string `json:"awsVpc,omitempty"`
	AWSSubnet        string `json:"awsSubnet,omitempty"`

	// GCP specific
	GCPProject string `json:"gcpProject,omitempty"`
	GCPZone    string `json:"gcpZone,omitempty"`
	GCPNetwork string `json:"gcpNetwork,omitempty"`

	// Azure specific
	AzureSubscription  string `json:"azureSubscription,omitempty"`
	AzureResourceGroup string `json:"azureResourceGroup,omitempty"`

	// DigitalOcean specific
	DOToken   string `json:"doToken,omitempty"`
	DOSSHKeys []int  `json:"doSshKeys,omitempty"`
}

// RemoteNode represents a deployed MPC node.
type RemoteNode struct {
	NodeConfig   *NodeConfig   `json:"nodeConfig"`
	Host         *models.Host  `json:"host"`
	Provider     CloudProvider `json:"provider"`
	InstanceID   string        `json:"instanceId"`
	PublicIP     string        `json:"publicIp"`
	PrivateIP    string        `json:"privateIp"`
	Region       string        `json:"region"`
	DeployedAt   time.Time     `json:"deployedAt"`
	Status       NodeStatus    `json:"status"`
	KeyEncrypted bool          `json:"keyEncrypted"`
}

// RemoteNetworkConfig holds configuration for a deployed MPC network.
type RemoteNetworkConfig struct {
	NetworkConfig *NetworkConfig    `json:"networkConfig"`
	Deployment    *DeploymentConfig `json:"deployment"`
	Nodes         []*RemoteNode     `json:"nodes"`
	DeployedAt    time.Time         `json:"deployedAt"`
}

// DeploymentManager manages MPC node deployments to cloud providers.
type DeploymentManager struct {
	baseDir string
	mgr     *NodeManager
}

// NewDeploymentManager creates a new deployment manager.
func NewDeploymentManager(baseDir string) *DeploymentManager {
	return &DeploymentManager{
		baseDir: baseDir,
		mgr:     NewNodeManager(baseDir),
	}
}

// DeployNetwork deploys an MPC network to cloud infrastructure.
func (d *DeploymentManager) DeployNetwork(ctx context.Context, networkName string, cfg *DeploymentConfig) (*RemoteNetworkConfig, error) {
	// Load network config
	networkCfg, err := d.mgr.LoadNetworkConfig(networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to load network config: %w", err)
	}

	// Validate deployment config
	if err := d.validateDeploymentConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid deployment config: %w", err)
	}

	// Deploy nodes based on provider
	var nodes []*RemoteNode
	switch cfg.Provider {
	case CloudProviderAWS:
		nodes, err = d.deployToAWS(ctx, networkCfg, cfg)
	case CloudProviderGCP:
		nodes, err = d.deployToGCP(ctx, networkCfg, cfg)
	case CloudProviderAzure:
		nodes, err = d.deployToAzure(ctx, networkCfg, cfg)
	case CloudProviderDigitalOcean:
		nodes, err = d.deployToDigitalOcean(ctx, networkCfg, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	remoteCfg := &RemoteNetworkConfig{
		NetworkConfig: networkCfg,
		Deployment:    cfg,
		Nodes:         nodes,
		DeployedAt:    time.Now(),
	}

	// Save deployment config
	if err := d.saveDeploymentConfig(networkName, remoteCfg); err != nil {
		return nil, fmt.Errorf("failed to save deployment config: %w", err)
	}

	return remoteCfg, nil
}

// ConnectToNode establishes SSH connection to a remote node.
func (d *DeploymentManager) ConnectToNode(ctx context.Context, networkName, nodeName string) (*models.Host, error) {
	remoteCfg, err := d.LoadDeploymentConfig(networkName)
	if err != nil {
		return nil, err
	}

	for _, node := range remoteCfg.Nodes {
		if node.NodeConfig.NodeName == nodeName {
			return node.Host, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeName)
}

// RunCommandOnNode runs a command on a remote MPC node via SSH.
func (d *DeploymentManager) RunCommandOnNode(ctx context.Context, host *models.Host, command string, timeout time.Duration) ([]byte, error) {
	return host.Command(command, nil, timeout)
}

// StartRemoteNode starts an MPC node on a remote server.
func (d *DeploymentManager) StartRemoteNode(ctx context.Context, networkName, nodeName string) error {
	host, err := d.ConnectToNode(ctx, networkName, nodeName)
	if err != nil {
		return err
	}

	// Start the MPC node service
	cmd := "sudo systemctl start mpc-node"
	if _, err := d.RunCommandOnNode(ctx, host, cmd, 30*time.Second); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	return nil
}

// StopRemoteNode stops an MPC node on a remote server.
func (d *DeploymentManager) StopRemoteNode(ctx context.Context, networkName, nodeName string) error {
	host, err := d.ConnectToNode(ctx, networkName, nodeName)
	if err != nil {
		return err
	}

	// Stop the MPC node service
	cmd := "sudo systemctl stop mpc-node"
	if _, err := d.RunCommandOnNode(ctx, host, cmd, 30*time.Second); err != nil {
		return fmt.Errorf("failed to stop node: %w", err)
	}

	return nil
}

// GetRemoteNodeStatus gets the status of a remote MPC node.
func (d *DeploymentManager) GetRemoteNodeStatus(ctx context.Context, networkName, nodeName string) (*NodeInfo, error) {
	host, err := d.ConnectToNode(ctx, networkName, nodeName)
	if err != nil {
		return nil, err
	}

	// Check systemd service status
	output, err := d.RunCommandOnNode(ctx, host, "systemctl is-active mpc-node", 10*time.Second)

	status := NodeStatusStopped
	if err == nil && string(output) == "active\n" {
		status = NodeStatusRunning
	}

	remoteCfg, _ := d.LoadDeploymentConfig(networkName)
	for _, node := range remoteCfg.Nodes {
		if node.NodeConfig.NodeName == nodeName {
			return &NodeInfo{
				Config:   node.NodeConfig,
				Status:   status,
				Endpoint: fmt.Sprintf("http://%s:%d", node.PublicIP, node.NodeConfig.APIPort),
			}, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeName)
}

// UnlockNodeKeys unlocks encrypted keys on a remote node.
// Keys are encrypted with age and stored in ~/.lux/keys/mpc/
// The identity file (private key) is needed to decrypt.
func (d *DeploymentManager) UnlockNodeKeys(ctx context.Context, networkName, nodeName string, ageIdentityPath string) error {
	host, err := d.ConnectToNode(ctx, networkName, nodeName)
	if err != nil {
		return err
	}

	// Upload the identity file temporarily
	identityData, err := os.ReadFile(ageIdentityPath)
	if err != nil {
		return fmt.Errorf("failed to read identity file: %w", err)
	}

	// Create temp file on remote
	tmpCmd := "mktemp"
	tmpOutput, err := d.RunCommandOnNode(ctx, host, tmpCmd, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := string(tmpOutput)

	// Write identity to temp file (would use SCP in real implementation)
	// For now, this is a placeholder
	_ = identityData
	_ = tmpPath

	// Decrypt the key shares
	decryptCmd := fmt.Sprintf("age -d -i %s ~/.lux/keys/mpc/%s/share.age > ~/.lux/keys/mpc/%s/share.key", tmpPath, nodeName, nodeName)
	if _, err := d.RunCommandOnNode(ctx, host, decryptCmd, 30*time.Second); err != nil {
		return fmt.Errorf("failed to decrypt keys: %w", err)
	}

	// Remove temp identity file
	rmCmd := fmt.Sprintf("rm -f %s", tmpPath)
	d.RunCommandOnNode(ctx, host, rmCmd, 10*time.Second)

	return nil
}

// BackupRemoteNode creates a backup of a remote node and uploads to cloud storage.
func (d *DeploymentManager) BackupRemoteNode(ctx context.Context, networkName, nodeName, destination string) error {
	host, err := d.ConnectToNode(ctx, networkName, nodeName)
	if err != nil {
		return err
	}

	// Create backup on remote node
	backupCmd := fmt.Sprintf("lux mpc backup create --destination %s", destination)
	if _, err := d.RunCommandOnNode(ctx, host, backupCmd, 5*time.Minute); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	return nil
}

// LoadDeploymentConfig loads deployment configuration from disk.
func (d *DeploymentManager) LoadDeploymentConfig(networkName string) (*RemoteNetworkConfig, error) {
	configPath := filepath.Join(d.baseDir, networkName, "deployment.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read deployment config: %w", err)
	}

	var cfg RemoteNetworkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse deployment config: %w", err)
	}

	return &cfg, nil
}

// Helper functions

func (d *DeploymentManager) validateDeploymentConfig(cfg *DeploymentConfig) error {
	if cfg.SSHKeyPath == "" {
		return fmt.Errorf("SSH key path is required")
	}
	if _, err := os.Stat(cfg.SSHKeyPath); err != nil {
		return fmt.Errorf("SSH key not found: %s", cfg.SSHKeyPath)
	}

	switch cfg.Provider {
	case CloudProviderAWS:
		if cfg.Region == "" {
			return fmt.Errorf("AWS region is required")
		}
	case CloudProviderGCP:
		if cfg.GCPProject == "" {
			return fmt.Errorf("GCP project is required")
		}
	case CloudProviderAzure:
		if cfg.AzureSubscription == "" {
			return fmt.Errorf("Azure subscription is required")
		}
	case CloudProviderDigitalOcean:
		if cfg.DOToken == "" {
			cfg.DOToken = os.Getenv("DIGITALOCEAN_TOKEN")
			if cfg.DOToken == "" {
				return fmt.Errorf("DigitalOcean token is required")
			}
		}
	}

	return nil
}

func (d *DeploymentManager) saveDeploymentConfig(networkName string, cfg *RemoteNetworkConfig) error {
	configPath := filepath.Join(d.baseDir, networkName, "deployment.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0640)
}

func (d *DeploymentManager) deployToAWS(ctx context.Context, networkCfg *NetworkConfig, cfg *DeploymentConfig) ([]*RemoteNode, error) {
	// This would use pkg/cloud/aws to:
	// 1. Create security group for MPC ports
	// 2. Launch EC2 instances (one per node)
	// 3. Install MPC node software
	// 4. Configure and start nodes

	// Placeholder implementation
	return nil, fmt.Errorf("AWS deployment not yet implemented - use 'lux mpc deploy --provider aws'")
}

func (d *DeploymentManager) deployToGCP(ctx context.Context, networkCfg *NetworkConfig, cfg *DeploymentConfig) ([]*RemoteNode, error) {
	// This would use pkg/cloud/gcp to deploy to Google Cloud
	return nil, fmt.Errorf("GCP deployment not yet implemented - use 'lux mpc deploy --provider gcp'")
}

func (d *DeploymentManager) deployToAzure(ctx context.Context, networkCfg *NetworkConfig, cfg *DeploymentConfig) ([]*RemoteNode, error) {
	// This would use Azure SDK to deploy
	return nil, fmt.Errorf("Azure deployment not yet implemented - use 'lux mpc deploy --provider azure'")
}

func (d *DeploymentManager) deployToDigitalOcean(ctx context.Context, networkCfg *NetworkConfig, cfg *DeploymentConfig) ([]*RemoteNode, error) {
	// This would use DigitalOcean API to deploy droplets
	return nil, fmt.Errorf("DigitalOcean deployment not yet implemented - use 'lux mpc deploy --provider digitalocean'")
}

// DefaultInstanceTypes returns recommended instance types per provider.
func DefaultInstanceTypes() map[CloudProvider]string {
	return map[CloudProvider]string{
		CloudProviderAWS:          "t3.medium",    // 2 vCPU, 4GB RAM
		CloudProviderGCP:          "e2-medium",    // 2 vCPU, 4GB RAM
		CloudProviderAzure:        "Standard_B2s", // 2 vCPU, 4GB RAM
		CloudProviderDigitalOcean: "s-2vcpu-4gb",  // 2 vCPU, 4GB RAM
	}
}

// DefaultRegions returns default regions per provider.
func DefaultRegions() map[CloudProvider]string {
	return map[CloudProvider]string{
		CloudProviderAWS:          "us-east-1",
		CloudProviderGCP:          "us-central1",
		CloudProviderAzure:        "eastus",
		CloudProviderDigitalOcean: "nyc1",
	}
}
