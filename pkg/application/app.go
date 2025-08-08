// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package application

import (
	"encoding/json"
	"os"
	"path/filepath"

	sdkapp "github.com/luxfi/sdk/application"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/prompts"
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

// The CLI's prompts.Prompter already implements the SDK's prompts.Prompter interface
// So we just embed it directly

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

// All the SDK methods are now provided by embedded type
// These duplicate SDK functionality and should be removed
