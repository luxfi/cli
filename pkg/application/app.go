// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package application

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sdkapp "github.com/luxfi/sdk/application"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/prompts"
	sdkprompts "github.com/luxfi/sdk/prompts"
	"github.com/luxfi/cli/pkg/types"
	luxlog "github.com/luxfi/log"
)

const (
	WriteReadReadPerms = 0o644
)

// Lux extends the SDK's application.Lux type with CLI-specific functionality
type Lux struct {
	*sdkapp.Lux // Embed SDK's Lux type
	Conf        *config.Config // CLI-specific config
}

func New() *Lux {
	return &Lux{
		Lux: sdkapp.New(),
	}
}

func (app *Lux) Setup(baseDir string, log luxlog.Logger, conf *config.Config, prompt prompts.Prompter, downloader Downloader) {
	// Call the embedded SDK's Setup method with SDK types
	// Note: we need to convert CLI types to SDK types
	var sdkPrompt sdkapp.Prompter
	if prompt != nil {
		sdkPrompt = promptAdapter{prompt}
	}
	var sdkDownloader sdkapp.Downloader
	if downloader != nil {
		sdkDownloader = downloaderAdapter{downloader}
	}
	// SDK config is different from CLI config, so we pass nil for now
	app.Lux.Setup(baseDir, log, nil, sdkPrompt, sdkDownloader)
	app.Conf = conf // Store CLI-specific config
}

// GetSDKApp returns the embedded SDK application for compatibility with SDK-based functions
func (app *Lux) GetSDKApp() *sdkapp.Lux {
	return app.Lux
}

// GetAggregatorLogDir returns the signature aggregator log directory
func (app *Lux) GetAggregatorLogDir(clusterName string) string {
	if clusterName != "" {
		return filepath.Join(app.GetBaseDir(), "aggregator", clusterName, "logs")
	}
	return filepath.Join(app.GetBaseDir(), "aggregator", "logs")
}

// Adapter types to bridge CLI and SDK interfaces
// promptAdapter wraps CLI's Prompter to implement SDK's Prompter interface
type promptAdapter struct {
	prompts.Prompter
}

// CaptureURL adapts the CLI's CaptureURL (2 params) to SDK's CaptureURL (1 param)
func (p promptAdapter) CaptureURL(promptStr string) (string, error) {
	// Call the CLI's CaptureURL with validateConnection=false by default
	return p.Prompter.CaptureURL(promptStr, false)
}

// CaptureWeight adapts the CLI's CaptureWeight to SDK's signature
func (p promptAdapter) CaptureWeight(promptStr string) (uint64, error) {
	// Call CLI's CaptureWeight with nil validator since SDK doesn't pass one
	return p.Prompter.CaptureWeight(promptStr, nil)
}

// CapturePositiveInt adapts between the different Comparator types
func (p promptAdapter) CapturePositiveInt(promptStr string, comparators []sdkprompts.Comparator) (int, error) {
	// Convert SDK comparators to CLI comparators
	cliComparators := make([]prompts.Comparator, len(comparators))
	for i, comp := range comparators {
		cliComparators[i] = prompts.Comparator{
			Label: comp.Label,
			Type:  comp.Type,
			Value: comp.Value,
		}
	}
	return p.Prompter.CapturePositiveInt(promptStr, cliComparators)
}

// CaptureUint64Compare adapts between the different Comparator types
func (p promptAdapter) CaptureUint64Compare(promptStr string, comparators []sdkprompts.Comparator) (uint64, error) {
	// Convert SDK comparators to CLI comparators
	cliComparators := make([]prompts.Comparator, len(comparators))
	for i, comp := range comparators {
		cliComparators[i] = prompts.Comparator{
			Label: comp.Label,
			Type:  comp.Type,
			Value: comp.Value,
		}
	}
	return p.Prompter.CaptureUint64Compare(promptStr, cliComparators)
}

type downloaderAdapter struct {
	Downloader
}

// The CLI and SDK Downloader interfaces are identical, so we just delegate directly

// These methods are now provided by the embedded SDK Lux type
// Only add CLI-specific methods that don't exist in SDK

func (app *Lux) GetLuxBinDir() string {
	return filepath.Join(app.GetBaseDir(), constants.LuxCliBinDir, constants.LuxInstallDir)
}

func (app *Lux) GetLuxgoBinDir() string {
	return filepath.Join(app.GetBaseDir(), constants.LuxCliBinDir, constants.LuxGoInstallDir)
}

func (app *Lux) GetEVMBinDir() string {
	return filepath.Join(app.GetBaseDir(), constants.LuxCliBinDir, constants.EVMInstallDir)
}

func (app *Lux) GetUpgradeBytesFilepath(subnetName string) string {
	return app.GetUpgradeBytesFilePath(subnetName) // Use SDK method
}

func (app *Lux) GetReposDir() string {
	return filepath.Join(app.GetBaseDir(), constants.ReposDir)
}

func (app *Lux) GetWarpRelayerBinDir() string {
	return filepath.Join(app.GetBaseDir(), "bin", "warp-relayer")
}

func (app *Lux) GetMonitoringDashboardDir() string {
	return filepath.Join(app.GetBaseDir(), "monitoring", "dashboards")
}

func (app *Lux) GetSSHCertFilePath(certName string) (string, error) {
	certPath := filepath.Join(app.GetBaseDir(), "ssh", certName+constants.CertSuffix)
	// Check if file exists
	if _, err := os.Stat(certPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("certificate %s not found", certName)
		}
		return "", err
	}
	return certPath, nil
}

// SetupMonitoringEnv sets up monitoring environment
func (app *Lux) SetupMonitoringEnv(clusterName string) error {
	// Create monitoring directory structure
	monitoringDir := filepath.Join(app.GetBaseDir(), "monitoring", clusterName)
	if err := os.MkdirAll(monitoringDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create monitoring directory: %w", err)
	}
	
	// Create config files for monitoring
	configPath := filepath.Join(monitoringDir, "config.json")
	config := map[string]interface{}{
		"clusterName": clusterName,
		"enabled":     true,
		"metrics":     []string{"cpu", "memory", "network", "disk"},
		"interval":    60, // seconds
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal monitoring config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, WriteReadReadPerms); err != nil {
		return fmt.Errorf("failed to write monitoring config: %w", err)
	}
	
	return nil
}

func (app *Lux) GetNodeInstanceDirPath(nodeName string) string {
	return filepath.Join(app.GetBaseDir(), "nodes", nodeName)
}

func (app *Lux) GetWarpRelayerServiceStorageDir() string {
	return filepath.Join(app.GetBaseDir(), "services", "warp-relayer")
}

func (app *Lux) CreateNodeCloudConfigFile(clusterName string, nodeConfig interface{}) error {
	// Create cloud configuration file for the node
	nodeDir := app.GetNodeInstanceDirPath(clusterName)
	if err := os.MkdirAll(nodeDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create node directory: %w", err)
	}
	
	// Write the node configuration
	configPath := filepath.Join(nodeDir, "cloud-config.json")
	data, err := json.MarshalIndent(nodeConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal node config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, WriteReadReadPerms); err != nil {
		return fmt.Errorf("failed to write node config: %w", err)
	}
	
	return nil
}

func (app *Lux) GetClustersConfig() (map[string]interface{}, error) {
	return app.LoadClustersConfig()
}

// ClusterExists checks if a cluster exists
func (app *Lux) ClusterExists(clusterName string) (bool, error) {
	clusterDir := app.GetLocalClusterDir(clusterName)
	info, err := os.Stat(clusterDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// HasSubnetEVMGenesis checks if the blockchain has a Subnet-EVM genesis
func (app *Lux) HasSubnetEVMGenesis(blockchainName string) (bool, string, error) {
	genesisPath := app.GetGenesisPath(blockchainName)
	genesisBytes, err := os.ReadFile(genesisPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	
	// Check if it's a Subnet-EVM genesis by looking for key fields
	var genesis map[string]interface{}
	if err := json.Unmarshal(genesisBytes, &genesis); err != nil {
		return false, "", nil // Not JSON, so not Subnet-EVM
	}
	
	// Subnet-EVM genesis should have "alloc" and "config" fields
	_, hasAlloc := genesis["alloc"]
	_, hasConfig := genesis["config"]
	
	if hasAlloc && hasConfig {
		return true, string(genesisBytes), nil
	}
	
	return false, "", nil
}

// LoadEvmGenesis loads EVM genesis for a blockchain
func (app *Lux) LoadEvmGenesis(blockchainName string) (*types.EvmGenesis, error) {
	genesisPath := app.GetGenesisPath(blockchainName)
	genesisBytes, err := os.ReadFile(genesisPath)
	if err != nil {
		return nil, err
	}
	
	var genesis types.EvmGenesis
	if err := json.Unmarshal(genesisBytes, &genesis); err != nil {
		return nil, err
	}
	
	return &genesis, nil
}

func (*Lux) GetLuxCompatibilityURL() string {
	return constants.LuxCompatibilityURL
}

// All the above methods are provided by embedded SDK type
// No need to duplicate them here

// CLI-specific config methods
func (app *Lux) WriteConfigFile(data []byte) error {
	configPath := app.GetConfigPath()
	// Use SDK's private writeFile method through a wrapper
	if err := os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755); err != nil {
		return err
	}
	return os.WriteFile(configPath, data, WriteReadReadPerms)
}

func (app *Lux) LoadConfig() (types.Config, error) {
	configPath := app.GetConfigPath()
	jsonBytes, err := os.ReadFile(configPath)
	if err != nil {
		return types.Config{}, err
	}

	var cfg types.Config
	err = json.Unmarshal(jsonBytes, &cfg)
	return cfg, err
}

func (app *Lux) ConfigFileExists() bool {
	configPath := app.GetConfigPath()
	_, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// GetBasePath returns the base directory path for the CLI
func (app *Lux) GetBasePath() string {
	return app.GetBaseDir()
}

// GetClusterConfig loads cluster configuration from disk
func (app *Lux) GetClusterConfig(clusterName string) (map[string]interface{}, error) {
	clusterConfigPath := filepath.Join(app.GetBaseDir(), "clusters", clusterName, "config.json")
	data, err := os.ReadFile(clusterConfigPath)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// SetClusterConfig saves cluster configuration to disk
func (app *Lux) SetClusterConfig(clusterName string, config map[string]interface{}) error {
	clusterDir := filepath.Join(app.GetBaseDir(), "clusters", clusterName)
	clusterConfigPath := filepath.Join(clusterDir, "config.json")
	
	// Ensure directory exists
	if err := os.MkdirAll(clusterDir, constants.DefaultPerms755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(clusterConfigPath, data, WriteReadReadPerms)
}

// SaveClustersConfig saves the clusters configuration
func (app *Lux) SaveClustersConfig(config map[string]interface{}) error {
	clustersPath := filepath.Join(app.GetBaseDir(), constants.ClustersConfigFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(clustersPath, data, WriteReadReadPerms)
}

// LoadClusterNodeConfig loads node configuration for a cluster
func (app *Lux) LoadClusterNodeConfig(clusterName string, nodeName string) (map[string]interface{}, error) {
	nodeConfigPath := filepath.Join(app.GetBaseDir(), "clusters", clusterName, "nodes", nodeName, "config.json")
	data, err := os.ReadFile(nodeConfigPath)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// GetNodeStakingDir returns the staking directory for a node
func (app *Lux) GetNodeStakingDir(nodeName string) string {
	return filepath.Join(app.GetBaseDir(), "nodes", nodeName, "staking")
}

// GetLoadTestInventoryDir returns the load test inventory directory
func (app *Lux) GetLoadTestInventoryDir(clusterName string) string {
	return filepath.Join(app.GetBaseDir(), "inventory", "load-test", clusterName)
}

// CheckCertInSSHDir checks if a certificate exists in the SSH directory
func (app *Lux) CheckCertInSSHDir(certName string) bool {
	certPath := filepath.Join(app.GetBaseDir(), "ssh", certName+constants.CertSuffix)
	_, err := os.Stat(certPath)
	return err == nil
}

// GetNodeConfigPath returns the path to a node's config file
func (app *Lux) GetNodeConfigPath(nodeName string) string {
	return filepath.Join(app.GetNodeInstanceDirPath(nodeName), "node.json")
}

// GetClusterYAMLFilePath returns the path to a cluster's YAML config file
func (app *Lux) GetClusterYAMLFilePath(clusterName string) string {
	return filepath.Join(app.GetBaseDir(), "clusters", clusterName, constants.ClusterYAMLFileName)
}

// All the SDK methods are now provided by embedded type
// These duplicate SDK functionality and should be removed
