// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chain provides chain configuration management with an overlay model.
//
// Config precedence (highest → lowest):
//  1. CLI flags / inline JSON
//  2. Per-run overrides (in run dir)
//  3. User global chain configs (~/.lux/chains/<chain-id>/config.json)
//  4. Built-in defaults
//
// The run directory chainConfigs are treated as rendered output, not source.
package chain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/models"
)

// Well-known chain IDs
const (
	// PChainID is the well-known P-Chain blockchain ID (all 1s with LpoYY suffix)
	PChainID = "11111111111111111111111111111111LpoYY"
)

// DefaultEVMConfig returns the default EVM chain configuration with admin API enabled.
// This applies to all EVM chains including C-chain and deployed chains.
func DefaultEVMConfig() map[string]interface{} {
	return map[string]interface{}{
		"eth-apis": []string{
			"eth", "eth-filter", "net", "web3",
			"internal-eth", "internal-blockchain", "internal-transaction", "internal-account",
			"admin",
		},
		"admin-api-enabled": true,
		"log-level":         "info",
	}
}

// Config represents a chain configuration with overlay support
type Config struct {
	app     *application.Lux
	chainID string // "C" for C-chain, or blockchain ID for others
	alias   string // Human-readable name like "zoo"

	// Overlay layers (lowest to highest precedence)
	defaults  map[string]interface{}
	global    map[string]interface{}
	runConfig map[string]interface{}
	cliConfig map[string]interface{}
}

// NewConfig creates a new chain config for the given chain ID
func NewConfig(app *application.Lux, chainID string) *Config {
	return &Config{
		app:      app,
		chainID:  chainID,
		defaults: DefaultEVMConfig(),
	}
}

// NewConfigWithAlias creates a config with both chain ID and human-readable alias
func NewConfigWithAlias(app *application.Lux, chainID, alias string) *Config {
	c := NewConfig(app, chainID)
	c.alias = alias
	return c
}

// LoadGlobal loads the global config from ~/.lux/chains/<chainID>/config.json
func (c *Config) LoadGlobal() error {
	configPath := c.globalConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // No global config, use defaults
	}

	data, err := os.ReadFile(configPath) //nolint:gosec // G304: Reading from app's chain config directory
	if err != nil {
		return fmt.Errorf("failed to read global config: %w", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse global config: %w", err)
	}

	c.global = cfg
	return nil
}

// SetCLIOverride sets a CLI override for specific keys
func (c *Config) SetCLIOverride(key string, value interface{}) {
	if c.cliConfig == nil {
		c.cliConfig = make(map[string]interface{})
	}
	c.cliConfig[key] = value
}

// SetCLIOverrides sets multiple CLI overrides
func (c *Config) SetCLIOverrides(overrides map[string]interface{}) {
	if c.cliConfig == nil {
		c.cliConfig = make(map[string]interface{})
	}
	for k, v := range overrides {
		c.cliConfig[k] = v
	}
}

// Effective returns the effective configuration by merging all layers
func (c *Config) Effective() map[string]interface{} {
	result := make(map[string]interface{})

	// Apply in order: defaults → global → run → cli
	for k, v := range c.defaults {
		result[k] = v
	}
	for k, v := range c.global {
		result[k] = v
	}
	for k, v := range c.runConfig {
		result[k] = v
	}
	for k, v := range c.cliConfig {
		result[k] = v
	}

	return result
}

// EffectiveJSON returns the effective config as formatted JSON
func (c *Config) EffectiveJSON() (string, error) {
	data, err := json.MarshalIndent(c.Effective(), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveGlobal saves the effective config as the global config
func (c *Config) SaveGlobal() error {
	configDir := filepath.Dir(c.globalConfigPath())
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c.Effective(), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(c.globalConfigPath(), data, 0o644) //nolint:gosec // G306: Config needs to be readable
}

// Render writes the config to the specified run directory's chainConfigs
func (c *Config) Render(runDir string) error {
	chainConfigDir := filepath.Join(runDir, "chainConfigs", c.chainID)
	if err := os.MkdirAll(chainConfigDir, 0o750); err != nil {
		return fmt.Errorf("failed to create chain config dir: %w", err)
	}

	configPath := filepath.Join(chainConfigDir, "config.json")
	data, err := json.MarshalIndent(c.Effective(), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0o644) //nolint:gosec // G306: Config needs to be readable
}

// globalConfigPath returns the path to the global config file
// Uses ~/.lux/chains/<chainID>/config.json - consolidating all chain configs
func (c *Config) globalConfigPath() string {
	return filepath.Join(c.app.GetChainConfigDir(), c.chainID, "config.json")
}

// EnableAdmin ensures admin API is enabled in the config
func (c *Config) EnableAdmin() {
	c.SetCLIOverride("admin-api-enabled", true)

	// Also ensure "admin" is in eth-apis
	effective := c.Effective()
	if apis, ok := effective["eth-apis"].([]interface{}); ok {
		hasAdmin := false
		for _, api := range apis {
			if api == "admin" {
				hasAdmin = true
				break
			}
		}
		if !hasAdmin {
			apis = append(apis, "admin")
			c.SetCLIOverride("eth-apis", apis)
		}
	}
}

// Manager handles chain configuration for a network run
type Manager struct {
	app     *application.Lux
	configs map[string]*Config // chainID -> Config
}

// NewManager creates a new chain config manager
func NewManager(app *application.Lux) *Manager {
	return &Manager{
		app:     app,
		configs: make(map[string]*Config),
	}
}

// AddChain adds a chain configuration to the manager
func (m *Manager) AddChain(chainID string) *Config {
	cfg := NewConfig(m.app, chainID)
	m.configs[chainID] = cfg
	return cfg
}

// AddChainWithAlias adds a chain with both ID and alias
func (m *Manager) AddChainWithAlias(chainID, alias string) *Config {
	cfg := NewConfigWithAlias(m.app, chainID, alias)
	m.configs[chainID] = cfg
	return cfg
}

// LoadDeployedChains discovers and loads configs for all deployed chains
func (m *Manager) LoadDeployedChains() error {
	// Always add C-chain
	cCfg := m.AddChainWithAlias("C", "c-chain")
	_ = cCfg.LoadGlobal()

	// Load deployed chains from sidecars
	chainDir := m.app.GetChainsDir()
	entries, err := os.ReadDir(chainDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		chainName := entry.Name()
		sc, err := m.app.LoadSidecar(chainName)
		if err != nil {
			continue
		}

		// Check for Local Network deployment
		if network, ok := sc.Networks[models.Local.String()]; ok {
			blockchainID := network.BlockchainID.String()
			if blockchainID != "" && blockchainID != PChainID {
				cfg := m.AddChainWithAlias(blockchainID, chainName)
				_ = cfg.LoadGlobal()
			}
		}
	}

	return nil
}

// GetConfig returns the config for a chain (by ID or alias)
func (m *Manager) GetConfig(chainIDOrAlias string) *Config {
	// Direct lookup by ID
	if cfg, ok := m.configs[chainIDOrAlias]; ok {
		return cfg
	}

	// Search by alias
	for _, cfg := range m.configs {
		if cfg.alias == chainIDOrAlias {
			return cfg
		}
	}

	return nil
}

// RenderAll renders all chain configs to the run directory
func (m *Manager) RenderAll(runDir string) error {
	for _, cfg := range m.configs {
		if err := cfg.Render(runDir); err != nil {
			return fmt.Errorf("failed to render config for %s: %w", cfg.chainID, err)
		}
	}
	return nil
}

// ToNetrunnerMap converts configs to the format expected by netrunner's WithChainConfigs
func (m *Manager) ToNetrunnerMap() map[string]string {
	result := make(map[string]string)
	for chainID, cfg := range m.configs {
		jsonStr, err := cfg.EffectiveJSON()
		if err != nil {
			continue
		}
		result[chainID] = jsonStr
	}
	return result
}

// EnableAdminAll enables admin API on all chains
func (m *Manager) EnableAdminAll() {
	for _, cfg := range m.configs {
		cfg.EnableAdmin()
	}
}
