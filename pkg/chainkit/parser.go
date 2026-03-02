// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a chain.yaml file, resolving relative paths.
func Load(path string) (*ChainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read chain.yaml: %w", err)
	}

	var cfg ChainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse chain.yaml: %w", err)
	}

	// Resolve relative paths against the directory containing chain.yaml
	dir := filepath.Dir(path)
	if cfg.Genesis.File != "" && !filepath.IsAbs(cfg.Genesis.File) {
		cfg.Genesis.File = filepath.Join(dir, cfg.Genesis.File)
	}
	if cfg.Brand.Logo != "" && !filepath.IsAbs(cfg.Brand.Logo) {
		cfg.Brand.Logo = filepath.Join(dir, cfg.Brand.Logo)
	}
	if cfg.Brand.Favicon != "" && !filepath.IsAbs(cfg.Brand.Favicon) {
		cfg.Brand.Favicon = filepath.Join(dir, cfg.Brand.Favicon)
	}

	if err := cfg.applyDefaults(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefaults fills in missing fields with sensible defaults.
func (c *ChainConfig) applyDefaults() error {
	if c.Version == "" {
		c.Version = "1"
	}
	if c.Chain.Type == "" {
		c.Chain.Type = "l2"
	}
	if c.Chain.Sequencer == "" {
		c.Chain.Sequencer = "lux"
	}
	if c.Chain.VM == "" {
		c.Chain.VM = "evm"
	}
	if c.Chain.DBType == "" {
		c.Chain.DBType = "zapdb"
	}
	if c.Chain.Compression == "" {
		c.Chain.Compression = "zstd"
	}
	if c.Token.Decimals == 0 {
		c.Token.Decimals = 18
	}
	if c.Deploy.Platform == "" {
		c.Deploy.Platform = "hanzo"
	}
	if c.Deploy.IngressClass == "" {
		c.Deploy.IngressClass = "hanzo"
	}
	if c.Deploy.SecretsProvider == "" {
		c.Deploy.SecretsProvider = "kms.hanzo.ai"
	}
	if c.Deploy.Registry == "" {
		c.Deploy.Registry = "ghcr.io/luxfi"
	}

	// Service defaults
	if c.Services.Node.Image == "" {
		c.Services.Node.Image = "ghcr.io/luxfi/node"
	}
	if c.Services.Node.StorageSize == "" {
		c.Services.Node.StorageSize = "100Gi"
	}
	if c.Services.Indexer.Image == "" {
		c.Services.Indexer.Image = "registry.digitalocean.com/hanzo/lux-indexer"
	}
	if c.Services.Indexer.ImageTag == "" {
		c.Services.Indexer.ImageTag = "v0.1.0"
	}
	if c.Services.Indexer.Replicas == 0 {
		c.Services.Indexer.Replicas = 1
	}
	if c.Services.Indexer.DBStorageSize == "" {
		c.Services.Indexer.DBStorageSize = "20Gi"
	}
	if c.Services.Indexer.PollInterval == 0 {
		c.Services.Indexer.PollInterval = 2
	}
	if c.Services.Explorer.Image == "" {
		c.Services.Explorer.Image = "ghcr.io/luxfi/explore"
	}
	if c.Services.Explorer.Replicas == 0 {
		c.Services.Explorer.Replicas = 1
	}
	if c.Services.Explorer.IngressClass == "" {
		c.Services.Explorer.IngressClass = c.Deploy.IngressClass
	}
	if c.Services.Gateway.Replicas == 0 {
		c.Services.Gateway.Replicas = 1
	}
	if c.Services.Gateway.RateLimitRPS == 0 {
		c.Services.Gateway.RateLimitRPS = 100
	}
	if c.Services.Gateway.RateLimitBurst == 0 {
		c.Services.Gateway.RateLimitBurst = 200
	}

	return nil
}

// Validate checks that the chain.yaml is well-formed and self-consistent.
func (c *ChainConfig) Validate() error {
	var errs []string

	if c.Chain.Slug == "" {
		errs = append(errs, "chain.slug is required")
	}
	if c.Chain.Name == "" {
		errs = append(errs, "chain.name is required")
	}

	// Validate chain type
	switch c.Chain.Type {
	case "l1", "l2", "l3":
	default:
		errs = append(errs, fmt.Sprintf("chain.type must be l1, l2, or l3 (got %q)", c.Chain.Type))
	}

	// Validate VM
	switch c.Chain.VM {
	case "evm", "pars", "custom":
	default:
		errs = append(errs, fmt.Sprintf("chain.vm must be evm, pars, or custom (got %q)", c.Chain.VM))
	}

	// Must have at least one network
	if len(c.Networks) == 0 {
		errs = append(errs, "at least one network must be defined in 'networks'")
	}

	for name, net := range c.Networks {
		if net.ChainID == 0 {
			errs = append(errs, fmt.Sprintf("networks.%s.chainId is required", name))
		}
		if net.Validators == 0 && c.Services.Node.Enabled {
			errs = append(errs, fmt.Sprintf("networks.%s.validators must be > 0 when node service is enabled", name))
		}
	}

	// Token
	if c.Token.Symbol == "" {
		errs = append(errs, "token.symbol is required")
	}
	if c.Token.Name == "" {
		errs = append(errs, "token.name is required")
	}

	// Brand
	if c.Brand.DisplayName == "" {
		errs = append(errs, "brand.displayName is required")
	}

	// Ingress class MUST NOT be nginx or caddy
	if c.Deploy.IngressClass == "nginx" || c.Deploy.IngressClass == "caddy" {
		errs = append(errs, "deploy.ingressClass must be 'hanzo' — nginx and caddy are not supported")
	}

	// Genesis: either file or inline, not both
	if c.Genesis.File != "" && c.Genesis.Inline != nil {
		errs = append(errs, "genesis: specify either 'file' or 'inline', not both")
	}
	if c.Genesis.File != "" {
		if _, err := os.Stat(c.Genesis.File); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("genesis.file does not exist: %s", c.Genesis.File))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("chain.yaml validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// NamespaceFor returns the K8s namespace for a given network name.
func (c *ChainConfig) NamespaceFor(network string) string {
	tmpl := c.Deploy.Namespace
	if tmpl == "" {
		tmpl = c.Chain.Slug + "-{network}"
	}
	return strings.ReplaceAll(tmpl, "{network}", network)
}

// LoadGenesisJSON reads the genesis file (or inline) as raw JSON bytes.
func (c *ChainConfig) LoadGenesisJSON() (json.RawMessage, error) {
	if c.Genesis.File != "" {
		data, err := os.ReadFile(c.Genesis.File)
		if err != nil {
			return nil, fmt.Errorf("read genesis file: %w", err)
		}
		return json.RawMessage(data), nil
	}
	if c.Genesis.Inline != nil {
		data, err := json.Marshal(c.Genesis.Inline)
		if err != nil {
			return nil, fmt.Errorf("marshal inline genesis: %w", err)
		}
		return json.RawMessage(data), nil
	}
	return nil, nil
}
