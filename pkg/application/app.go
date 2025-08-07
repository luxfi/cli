// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package application

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/lpm"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/types"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
)

const (
	WriteReadReadPerms = 0o644
)

type Lux struct {
	Log        luxlog.Logger
	baseDir    string
	Conf       *config.Config
	Prompt     prompts.Prompter
	Lpm        *lpm.Client
	LpmDir     string
	Apm        *lpm.Client // APM is similar to LPM
	ApmDir     func() string
	Downloader Downloader
	Cmd        interface{} // Current command being executed (cobra.Command)
}

func New() *Lux {
	return &Lux{}
}

func (app *Lux) Setup(baseDir string, log luxlog.Logger, conf *config.Config, prompt prompts.Prompter, downloader Downloader) {
	app.baseDir = baseDir
	app.Log = log
	app.Conf = conf
	app.Prompt = prompt
	app.Downloader = downloader
	app.ApmDir = func() string {
		return filepath.Join(baseDir, "apm")
	}
}

func (app *Lux) GetRunFile() string {
	return filepath.Join(app.GetRunDir(), constants.ServerRunFile)
}

func (app *Lux) GetSnapshotsDir() string {
	return filepath.Join(app.baseDir, constants.SnapshotsDirName)
}

func (app *Lux) GetBaseDir() string {
	return app.baseDir
}

func (app *Lux) GetSubnetDir() string {
	return filepath.Join(app.baseDir, constants.SubnetDir)
}

func (app *Lux) GetReposDir() string {
	return filepath.Join(app.baseDir, constants.ReposDir)
}

func (app *Lux) GetRunDir() string {
	return filepath.Join(app.baseDir, constants.RunDir)
}

func (app *Lux) IsLocalNetworkRunning() bool {
	// Check if the local network is running by checking the gRPC server status
	runFilePath := filepath.Join(app.GetRunDir(), constants.ServerRunFile)
	_, err := os.Stat(runFilePath)
	return err == nil
}

func (app *Lux) GetCustomVMDir() string {
	return filepath.Join(app.baseDir, constants.CustomVMDir)
}

func (app *Lux) GetPluginsDir() string {
	return filepath.Join(app.baseDir, constants.PluginDir)
}

func (app *Lux) GetLuxBinDir() string {
	return filepath.Join(app.baseDir, constants.LuxCliBinDir, constants.LuxInstallDir)
}

func (app *Lux) GetLuxgoBinDir() string {
	return filepath.Join(app.baseDir, constants.LuxCliBinDir, constants.LuxGoInstallDir)
}

func (app *Lux) GetEVMBinDir() string {
	return filepath.Join(app.baseDir, constants.LuxCliBinDir, constants.EVMInstallDir)
}

func (app *Lux) GetUpgradeBytesFilepath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, constants.UpgradeBytesFileName)
}

func (app *Lux) GetCustomVMPath(subnetName string) string {
	return filepath.Join(app.GetCustomVMDir(), subnetName)
}

func (app *Lux) GetLPMVMPath(vmid string) string {
	return filepath.Join(app.GetLPMPluginDir(), vmid)
}

func (app *Lux) GetGenesisPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, constants.GenesisFileName)
}

func (app *Lux) GetSidecarPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, constants.SidecarFileName)
}

func (app *Lux) GetConfigPath() string {
	return filepath.Join(app.baseDir, constants.ConfigDir)
}

func (app *Lux) GetElasticSubnetConfigPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, constants.ElasticSubnetConfigFileName)
}

func (app *Lux) GetKeyDir() string {
	return filepath.Join(app.baseDir, constants.KeyDir)
}

func (*Lux) GetTmpPluginDir() string {
	return os.TempDir()
}

func (app *Lux) GetLPMBaseDir() string {
	return filepath.Join(app.baseDir, "lpm")
}

func (app *Lux) GetLPMLog() string {
	return filepath.Join(app.baseDir, constants.LogDir, constants.LPMLogName)
}

func (app *Lux) GetLPMPluginDir() string {
	return filepath.Join(app.baseDir, constants.LPMPluginDir)
}

func (app *Lux) GetKeyPath(keyName string) string {
	return filepath.Join(app.baseDir, constants.KeyDir, keyName+constants.KeySuffix)
}

func (app *Lux) GetUpgradeBytesFilePath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, constants.UpgradeBytesFileName)
}

func (app *Lux) GetDownloader() Downloader {
	return app.Downloader
}

func (*Lux) GetLuxCompatibilityURL() string {
	return constants.LuxCompatibilityURL
}

func (app *Lux) ReadUpgradeFile(subnetName string) ([]byte, error) {
	upgradeBytesFilePath := app.GetUpgradeBytesFilePath(subnetName)

	return app.readFile(upgradeBytesFilePath)
}

func (app *Lux) ReadLockUpgradeFile(subnetName string) ([]byte, error) {
	upgradeBytesLockFilePath := app.GetUpgradeBytesFilePath(subnetName) + constants.UpgradeBytesLockExtension

	return app.readFile(upgradeBytesLockFilePath)
}

func (app *Lux) WriteUpgradeFile(subnetName string, bytes []byte) error {
	upgradeBytesFilePath := app.GetUpgradeBytesFilePath(subnetName)

	return app.writeFile(upgradeBytesFilePath, bytes)
}

func (app *Lux) WriteLockUpgradeFile(subnetName string, bytes []byte) error {
	upgradeBytesLockFilePath := app.GetUpgradeBytesFilePath(subnetName) + constants.UpgradeBytesLockExtension

	return app.writeFile(upgradeBytesLockFilePath, bytes)
}

func (app *Lux) WriteGenesisFile(subnetName string, genesisBytes []byte) error {
	genesisPath := app.GetGenesisPath(subnetName)

	return app.writeFile(genesisPath, genesisBytes)
}

func (app *Lux) GenesisExists(subnetName string) bool {
	genesisPath := app.GetGenesisPath(subnetName)
	_, err := os.Stat(genesisPath)
	return err == nil
}

func (app *Lux) SidecarExists(subnetName string) bool {
	sidecarPath := app.GetSidecarPath(subnetName)
	_, err := os.Stat(sidecarPath)
	return err == nil
}

func (app *Lux) SubnetConfigExists(subnetName string) bool {
	// There's always a sidecar, but imported subnets don't have a genesis right now
	return app.SidecarExists(subnetName)
}

func (app *Lux) KeyExists(keyName string) bool {
	keyPath := app.GetKeyPath(keyName)
	_, err := os.Stat(keyPath)
	return err == nil
}

func (app *Lux) CopyGenesisFile(inputFilename string, subnetName string) error {
	genesisBytes, err := os.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	genesisPath := app.GetGenesisPath(subnetName)
	if err := os.MkdirAll(filepath.Dir(genesisPath), constants.DefaultPerms755); err != nil {
		return err
	}

	return os.WriteFile(genesisPath, genesisBytes, WriteReadReadPerms)
}

func (app *Lux) CopyVMBinary(inputFilename string, subnetName string) error {
	vmBytes, err := os.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	vmPath := app.GetCustomVMPath(subnetName)
	return os.WriteFile(vmPath, vmBytes, WriteReadReadPerms)
}

func (app *Lux) CopyKeyFile(inputFilename string, keyName string) error {
	keyBytes, err := os.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	keyPath := app.GetKeyPath(keyName)
	return os.WriteFile(keyPath, keyBytes, WriteReadReadPerms)
}

func (app *Lux) LoadEvmGenesis(subnetName string) (core.Genesis, error) {
	genesisPath := app.GetGenesisPath(subnetName)
	jsonBytes, err := os.ReadFile(genesisPath)
	if err != nil {
		return core.Genesis{}, err
	}

	var gen core.Genesis
	err = json.Unmarshal(jsonBytes, &gen)
	return gen, err
}

func (app *Lux) LoadRawGenesis(subnetName string) ([]byte, error) {
	genesisPath := app.GetGenesisPath(subnetName)
	genesisBytes, err := os.ReadFile(genesisPath)
	if err != nil {
		return nil, err
	}

	return genesisBytes, err
}

func (app *Lux) CreateSidecar(sc *models.Sidecar) error {
	if sc.TokenName == "" {
		sc.TokenName = constants.DefaultTokenName
	}

	sidecarPath := app.GetSidecarPath(sc.Name)
	if err := os.MkdirAll(filepath.Dir(sidecarPath), constants.DefaultPerms755); err != nil {
		return err
	}

	// only apply the version on a write
	sc.Version = constants.SidecarVersion
	scBytes, err := json.MarshalIndent(sc, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(sidecarPath, scBytes, WriteReadReadPerms)
}

func (app *Lux) LoadSidecar(subnetName string) (models.Sidecar, error) {
	sidecarPath := app.GetSidecarPath(subnetName)
	jsonBytes, err := os.ReadFile(sidecarPath)
	if err != nil {
		return models.Sidecar{}, err
	}

	var sc models.Sidecar
	err = json.Unmarshal(jsonBytes, &sc)

	if sc.TokenName == "" {
		sc.TokenName = constants.DefaultTokenName
	}

	return sc, err
}

func (app *Lux) WriteSidecarFile(sc *models.Sidecar) error {
	sidecarPath := app.GetSidecarPath(sc.Name)
	jsonBytes, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	return app.writeFile(sidecarPath, jsonBytes)
}

func (app *Lux) UpdateSidecar(sc *models.Sidecar) error {
	sc.Version = constants.SidecarVersion
	scBytes, err := json.MarshalIndent(sc, "", "    ")
	if err != nil {
		return err
	}

	sidecarPath := app.GetSidecarPath(sc.Name)
	return os.WriteFile(sidecarPath, scBytes, WriteReadReadPerms)
}

func (app *Lux) UpdateSidecarNetworks(
	sc *models.Sidecar,
	network models.Network,
	subnetID ids.ID,
	blockchainID ids.ID,
) error {
	if sc.Networks == nil {
		sc.Networks = make(map[string]models.NetworkData)
	}
	sc.Networks[network.String()] = models.NetworkData{
		SubnetID:     subnetID,
		BlockchainID: blockchainID,
		RPCVersion:   sc.RPCVersion,
	}
	if err := app.UpdateSidecar(sc); err != nil {
		return fmt.Errorf("creation of chains and subnet was successful, but failed to update sidecar: %w", err)
	}
	return nil
}

func (app *Lux) UpdateSidecarElasticSubnet(
	sc *models.Sidecar,
	network models.Network,
	subnetID ids.ID,
	assetID ids.ID,
	pchainTXID ids.ID,
	tokenName string,
	tokenSymbol string,
) error {
	if sc.ElasticSubnet == nil {
		sc.ElasticSubnet = make(map[string]models.ElasticSubnet)
	}
	partialTxs := sc.ElasticSubnet[network.String()].Txs
	sc.ElasticSubnet[network.String()] = models.ElasticSubnet{
		SubnetID:    subnetID,
		AssetID:     assetID,
		PChainTXID:  pchainTXID,
		TokenName:   tokenName,
		TokenSymbol: tokenSymbol,
		Txs:         partialTxs,
	}
	if err := app.UpdateSidecar(sc); err != nil {
		return err
	}
	return nil
}

func (app *Lux) UpdateSidecarPermissionlessValidator(
	sc *models.Sidecar,
	network models.Network,
	nodeID string,
	txID ids.ID,
) error {
	elasticSubnet := sc.ElasticSubnet[network.String()]
	if elasticSubnet.Validators == nil {
		elasticSubnet.Validators = make(map[string]models.PermissionlessValidators)
	}
	elasticSubnet.Validators[nodeID] = models.PermissionlessValidators{TxID: txID}
	sc.ElasticSubnet[network.String()] = elasticSubnet
	if err := app.UpdateSidecar(sc); err != nil {
		return err
	}
	return nil
}

func (app *Lux) UpdateSidecarElasticSubnetPartialTx(
	sc *models.Sidecar,
	network models.Network,
	txName string,
	txID ids.ID,
) error {
	if sc.ElasticSubnet == nil {
		sc.ElasticSubnet = make(map[string]models.ElasticSubnet)
	}
	partialTxs := make(map[string]ids.ID)
	if sc.ElasticSubnet[network.String()].Txs != nil {
		partialTxs = sc.ElasticSubnet[network.String()].Txs
	}
	partialTxs[txName] = txID
	sc.ElasticSubnet[network.String()] = models.ElasticSubnet{
		Txs: partialTxs,
	}
	return app.UpdateSidecar(sc)
}

func (app *Lux) GetTokenName(subnetName string) string {
	sidecar, err := app.LoadSidecar(subnetName)
	if err != nil {
		return constants.DefaultTokenName
	}
	return sidecar.TokenName
}

func (app *Lux) GetSidecarNames() ([]string, error) {
	matches, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		return nil, err
	}

	var names []string
	for _, m := range matches {
		if !m.IsDir() {
			continue
		}
		// a subnet dir could theoretically exist without a sidecar yet...
		if _, err := os.Stat(filepath.Join(app.GetSubnetDir(), m.Name(), constants.SidecarFileName)); err == nil {
			names = append(names, m.Name())
		}
	}
	return names, nil
}

func (*Lux) readFile(path string) ([]byte, error) {
	if err := os.MkdirAll(filepath.Dir(path), constants.DefaultPerms755); err != nil {
		return nil, err
	}

	return os.ReadFile(path)
}

func (*Lux) writeFile(path string, bytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), constants.DefaultPerms755); err != nil {
		return err
	}

	return os.WriteFile(path, bytes, WriteReadReadPerms)
}

func (app *Lux) GetVersion() string {
	// Return a default version for now
	return "1.0.0"
}

func (app *Lux) WriteConfigFile(data []byte) error {
	configPath := app.GetConfigPath()
	return app.writeFile(configPath, data)
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

// CaptureYesNo delegates to the internal prompt
func (app *Lux) CaptureYesNo(prompt string) (bool, error) {
	return app.Prompt.CaptureYesNo(prompt)
}


func (app *Lux) CreateElasticSubnetConfig(subnetName string, es *models.ElasticSubnetConfig) error {
	elasticSubetConfigPath := app.GetElasticSubnetConfigPath(subnetName)
	if err := os.MkdirAll(filepath.Dir(elasticSubetConfigPath), constants.DefaultPerms755); err != nil {
		return err
	}

	esBytes, err := json.MarshalIndent(es, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(elasticSubetConfigPath, esBytes, WriteReadReadPerms)
}

func (app *Lux) LoadElasticSubnetConfig(subnetName string) (models.ElasticSubnetConfig, error) {
	elasticSubnetConfigPath := app.GetElasticSubnetConfigPath(subnetName)
	jsonBytes, err := os.ReadFile(elasticSubnetConfigPath)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}

	var esc models.ElasticSubnetConfig
	err = json.Unmarshal(jsonBytes, &esc)

	return esc, err
}

// GetNodesDir returns the nodes directory path
func (app *Lux) GetNodesDir() string {
	return filepath.Join(app.baseDir, "nodes")
}

// GetLogDir returns the log directory path
func (app *Lux) GetLogDir() string {
	return filepath.Join(app.baseDir, constants.LogDir)
}

// GetLocalClustersDir returns the directory for local clusters
func (app *Lux) GetLocalClustersDir() string {
	return filepath.Join(app.baseDir, "clusters")
}

// GetLocalClusterDir returns the directory for a specific local cluster
func (app *Lux) GetLocalClusterDir(clusterName string) string {
	return filepath.Join(app.GetLocalClustersDir(), clusterName)
}

// GetLocalRelayerConfigPath returns the path for local relayer config
func (app *Lux) GetLocalRelayerConfigPath() string {
	return filepath.Join(app.baseDir, "relayer", "config.json")
}

// GetLocalRelayerRunPath returns the path to the relayer run file
func (app *Lux) GetLocalRelayerRunPath(network models.Network) string {
	return filepath.Join(app.GetRunDir(), fmt.Sprintf("relayer-%s.run", network.String()))
}

// GetLocalRelayerLogPath returns the path to the relayer log file
func (app *Lux) GetLocalRelayerLogPath(network models.Network) string {
	return filepath.Join(app.GetLogDir(), fmt.Sprintf("relayer-%s.log", network.String()))
}

// GetLocalRelayerStorageDir returns the path to the relayer storage directory
func (app *Lux) GetLocalRelayerStorageDir(network models.Network) string {
	return filepath.Join(app.baseDir, "relayer-storage", network.String())
}

// GetKey returns the key for a given name
func (app *Lux) GetKey(keyName string) (string, error) {
	keyPath := app.GetKeyPath(keyName)
	if !utils.FileExists(keyPath) {
		return "", fmt.Errorf("key %s not found", keyName)
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return "", err
	}
	return string(keyBytes), nil
}

// GetBasePath returns the base directory path for the CLI
func (app *Lux) GetBasePath() string {
	return app.baseDir
}

// GetLuxdNodeConfigPath returns the node config path for a subnet
func (app *Lux) GetLuxdNodeConfigPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, "node-config.json")
}

// LuxdSubnetConfigExists checks if subnet config exists
func (app *Lux) LuxdSubnetConfigExists(subnetName string) bool {
	configPath := filepath.Join(app.GetSubnetDir(), subnetName, "subnet-config.json")
	_, err := os.Stat(configPath)
	return err == nil
}

// LoadRawLuxdSubnetConfig loads raw subnet config
func (app *Lux) LoadRawLuxdSubnetConfig(subnetName string) ([]byte, error) {
	configPath := filepath.Join(app.GetSubnetDir(), subnetName, "subnet-config.json")
	return os.ReadFile(configPath)
}

// ChainConfigExists checks if chain config exists
func (app *Lux) ChainConfigExists(subnetName string) bool {
	configPath := filepath.Join(app.GetSubnetDir(), subnetName, "chain-config.json")
	_, err := os.Stat(configPath)
	return err == nil
}

// LoadRawChainConfig loads raw chain config
func (app *Lux) LoadRawChainConfig(subnetName string) ([]byte, error) {
	configPath := filepath.Join(app.GetSubnetDir(), subnetName, "chain-config.json")
	return os.ReadFile(configPath)
}

// NetworkUpgradeExists checks if network upgrade file exists
func (app *Lux) NetworkUpgradeExists(subnetName string) bool {
	upgradePath := filepath.Join(app.GetSubnetDir(), subnetName, "upgrade.json")
	_, err := os.Stat(upgradePath)
	return err == nil
}

// LoadRawNetworkUpgrades loads raw network upgrades
func (app *Lux) LoadRawNetworkUpgrades(subnetName string) ([]byte, error) {
	upgradePath := filepath.Join(app.GetSubnetDir(), subnetName, "upgrade.json")
	return os.ReadFile(upgradePath)
}

// GetChainConfigPath returns the chain config path for a subnet
func (app *Lux) GetChainConfigPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, "chain-config.json")
}

// GetLuxdSubnetConfigPath returns the luxd subnet config path for a subnet
func (app *Lux) GetLuxdSubnetConfigPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, "subnet-config.json")
}

// GetPerNodeBlockchainConfig returns per-node blockchain config
func (app *Lux) GetPerNodeBlockchainConfig(subnetName string) map[string]interface{} {
	// Return empty config for now, can be extended later
	return make(map[string]interface{})
}

// LuxdNodeConfigExists checks if luxd node config exists  
func (app *Lux) LuxdNodeConfigExists(subnetName string) bool {
	configPath := app.GetLuxdNodeConfigPath(subnetName)
	_, err := os.Stat(configPath)
	return err == nil
}

// AddDefaultBlockchainRPCsToSidecar adds default RPC endpoints to sidecar
func (app *Lux) AddDefaultBlockchainRPCsToSidecar(
	blockchainName string, 
	network models.Network,
	nodeURIs []string,
) (bool, error) {
	// Stub implementation - return success for now
	// This method would typically update the sidecar with RPC endpoints
	return true, nil
}

// ListClusterNames returns a list of cluster names
func (app *Lux) ListClusterNames() ([]string, error) {
	clustersDir := app.GetLocalClustersDir()
	entries, err := os.ReadDir(clustersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	
	var clusterNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			clusterNames = append(clusterNames, entry.Name())
		}
	}
	return clusterNames, nil
}

// ClustersConfigExists checks if clusters config exists
func (app *Lux) ClustersConfigExists() bool {
	// Stub implementation - check for clusters config file
	configPath := filepath.Join(app.GetBaseDir(), constants.ClustersConfigFileName)
	_, err := os.Stat(configPath)
	return err == nil
}

// LoadClustersConfig loads the clusters configuration
func (app *Lux) LoadClustersConfig() (map[string]interface{}, error) {
	configPath := filepath.Join(app.GetBaseDir(), constants.ClustersConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// LoadClusterNodeConfig loads node configuration for a cluster
func (app *Lux) LoadClusterNodeConfig(clusterName string, nodeID string) (map[string]interface{}, error) {
	configPath := filepath.Join(app.GetLocalClusterDir(clusterName), nodeID, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// GetClusterConfig returns cluster configuration
func (app *Lux) GetClusterConfig(clusterName string) (map[string]interface{}, error) {
	configPath := filepath.Join(app.GetLocalClusterDir(clusterName), "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// GetAnsibleInventoryDirPath returns the ansible inventory directory path
func (app *Lux) GetAnsibleInventoryDirPath(clusterName string) string {
	return filepath.Join(app.GetLocalClusterDir(clusterName), "ansible", "inventories")
}

// GetMonitoringInventoryDir returns the monitoring inventory directory path
func (app *Lux) GetMonitoringInventoryDir(clusterName string) string {
	return filepath.Join(app.GetLocalClusterDir(clusterName), "monitoring", "inventory")
}

// ResetPluginsDir resets the plugins directory
func (app *Lux) ResetPluginsDir() error {
	pluginsDir := filepath.Join(app.baseDir, constants.PluginDir)
	if err := os.RemoveAll(pluginsDir); err != nil {
		return err
	}
	return os.MkdirAll(pluginsDir, constants.DefaultPerms755)
}

// GetSnapshotPath returns the path to a snapshot
func (app *Lux) GetSnapshotPath(snapshotName string) string {
	return filepath.Join(app.baseDir, constants.SnapshotsDirName, snapshotName)
}

// BlockchainConfigExists checks if blockchain config exists
func (app *Lux) BlockchainConfigExists(blockchainName string) bool {
	configPath := filepath.Join(app.GetSubnetDir(), blockchainName, "blockchain-config.json")
	_, err := os.Stat(configPath)
	return err == nil
}

// GetClusterNetwork returns the network for a given cluster
func (app *Lux) GetClusterNetwork(clusterName string) (models.Network, error) {
	// For now, all clusters are local networks
	// This could be extended to read from cluster config
	return models.NewLocalNetwork(), nil
}

// GetNetworkFromSidecarNetworkName returns the network from sidecar network name
func (app *Lux) GetNetworkFromSidecarNetworkName(name string) (models.Network, error) {
	network := models.GetNetworkFromSidecarNetworkName(name)
	if network == models.Undefined {
		return models.Undefined, fmt.Errorf("unknown network name: %s", name)
	}
	return network, nil
}

// GetBlockchainNamesOnNetwork returns blockchain names deployed on a network
func (app *Lux) GetBlockchainNamesOnNetwork(network models.Network, onlySOV bool) ([]string, error) {
	// Get all blockchain names from sidecar files
	blockchainNames := []string{}
	
	subnetDir := app.GetSubnetDir()
	entries, err := os.ReadDir(subnetDir)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if sidecar exists for this blockchain
			sidecarPath := filepath.Join(subnetDir, entry.Name(), "sidecar.json")
			if _, err := os.Stat(sidecarPath); err == nil {
				// Load sidecar to check if it's deployed on this network
				sc, err := app.LoadSidecar(entry.Name())
				if err != nil {
					continue
				}
				
				// Check if blockchain is deployed on the specified network
				if _, ok := sc.Networks[network.Name()]; ok {
					// If onlySOV is true, only include sovereign chains
					if !onlySOV || sc.Sovereign {
						blockchainNames = append(blockchainNames, entry.Name())
					}
				}
			}
		}
	}
	
	return blockchainNames, nil
}
