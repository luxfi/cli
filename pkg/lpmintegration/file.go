// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package lpmintegration provides integration with the Lux Package Manager (LPM).
package lpmintegration

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"gopkg.in/yaml.v3"
)

// GetRepos returns a list of all available LPM repositories.
func GetRepos(app *application.Lux) ([]string, error) {
	repositoryDir := filepath.Join(app.LpmDir, "repositories")
	orgs, err := os.ReadDir(repositoryDir)
	if err != nil {
		return []string{}, err
	}

	output := []string{}

	for _, org := range orgs {
		repoDir := filepath.Join(repositoryDir, org.Name())
		repos, err := os.ReadDir(repoDir)
		if err != nil {
			return []string{}, err
		}
		for _, repo := range repos {
			output = append(output, org.Name()+"/"+repo.Name())
		}
	}

	return output, nil
}

// GetChains returns a list of chains available in a repository.
func GetChains(app *application.Lux, repoAlias string) ([]string, error) {
	chainDir := filepath.Join(app.LpmDir, "repositories", repoAlias, "chains")
	chains, err := os.ReadDir(chainDir)
	if err != nil {
		return []string{}, err
	}
	chainOptions := make([]string, len(chains))
	for i, chain := range chains {
		// Remove the .yaml extension
		chainOptions[i] = strings.TrimSuffix(chain.Name(), filepath.Ext(chain.Name()))
	}

	return chainOptions, nil
}

// Chain represents an LPM chain configuration.
type Chain struct {
	ID          string   `yaml:"id"`
	Alias       string   `yaml:"alias"`
	VM          string   `yaml:"vm"`
	VMs         []string `yaml:"vms"`
	Config      string   `yaml:"config"`
	Genesis     string   `yaml:"genesis"`
	Description string   `yaml:"description"`
}

// VM represents an LPM virtual machine configuration.
type VM struct {
	ID          string `yaml:"id"`
	Alias       string `yaml:"alias"`
	VMType      string `yaml:"vm_type"`
	Binary      string `yaml:"binary"`
	ChainConfig string `yaml:"chain_config"`
	Chain       string `yaml:"chain"`
	Genesis     string `yaml:"genesis"`
	Version     string `yaml:"version"`
	URL         string `yaml:"url"`
	Checksum    string `yaml:"checksum"`
	Runtime     string `yaml:"runtime"`
	Description string `yaml:"description"`
}

// ChainWrapper wraps a Chain for YAML parsing.
type ChainWrapper struct {
	Chain Chain `yaml:"chain"`
}

// VMWrapper wraps a VM for YAML parsing.
type VMWrapper struct {
	VM VM `yaml:"vm"`
}

// LoadChainFile loads a chain configuration from a YAML file.
func LoadChainFile(app *application.Lux, chainKey string) (Chain, error) {
	repoAlias, chainName, err := splitKey(chainKey)
	if err != nil {
		return Chain{}, err
	}

	chainYamlPath := filepath.Join(app.LpmDir, "repositories", repoAlias, "chains", chainName+".yaml")
	var chainWrapper ChainWrapper

	chainYamlBytes, err := os.ReadFile(chainYamlPath) //nolint:gosec // G304: Reading from app's data directory
	if err != nil {
		return Chain{}, err
	}

	err = yaml.Unmarshal(chainYamlBytes, &chainWrapper)
	if err != nil {
		return Chain{}, err
	}

	return chainWrapper.Chain, nil
}

func getVMsInChain(app *application.Lux, chainKey string) ([]string, error) {
	chain, err := LoadChainFile(app, chainKey)
	if err != nil {
		return []string{}, err
	}

	return chain.VMs, nil
}

// LoadVMFile loads a VM configuration from a YAML file.
func LoadVMFile(app *application.Lux, repo, vm string) (VM, error) {
	vmYamlPath := filepath.Join(app.LpmDir, "repositories", repo, "vms", vm+".yaml")
	var vmWrapper VMWrapper

	vmYamlBytes, err := os.ReadFile(vmYamlPath) //nolint:gosec // G304: Reading from app's data directory
	if err != nil {
		return VM{}, err
	}

	err = yaml.Unmarshal(vmYamlBytes, &vmWrapper)
	if err != nil {
		return VM{}, err
	}

	return vmWrapper.VM, nil
}
