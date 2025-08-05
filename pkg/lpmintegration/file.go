// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"gopkg.in/yaml.v3"
)

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

func GetSubnets(app *application.Lux, repoAlias string) ([]string, error) {
	subnetDir := filepath.Join(app.LpmDir, "repositories", repoAlias, "subnets")
	subnets, err := os.ReadDir(subnetDir)
	if err != nil {
		return []string{}, err
	}
	subnetOptions := make([]string, len(subnets))
	for i, subnet := range subnets {
		// Remove the .yaml extension
		subnetOptions[i] = strings.TrimSuffix(subnet.Name(), filepath.Ext(subnet.Name()))
	}

	return subnetOptions, nil
}

// Types for LPM compatibility
type Subnet struct {
	ID          string   `yaml:"id"`
	Alias       string   `yaml:"alias"`
	VM          string   `yaml:"vm"`
	VMs         []string `yaml:"vms"`
	Config      string   `yaml:"config"`
	Genesis     string   `yaml:"genesis"`
	Description string   `yaml:"description"`
}

type VM struct {
	ID          string `yaml:"id"`
	Alias       string `yaml:"alias"`
	VMType      string `yaml:"vm_type"`
	Binary      string `yaml:"binary"`
	ChainConfig string `yaml:"chain_config"`
	Subnet      string `yaml:"subnet"`
	Genesis     string `yaml:"genesis"`
	Version     string `yaml:"version"`
	URL         string `yaml:"url"`
	Checksum    string `yaml:"checksum"`
	Runtime     string `yaml:"runtime"`
	Description string `yaml:"description"`
}

type SubnetWrapper struct {
	Subnet Subnet `yaml:"subnet"`
}

type VMWrapper struct {
	VM VM `yaml:"vm"`
}

func LoadSubnetFile(app *application.Lux, subnetKey string) (Subnet, error) {
	repoAlias, subnetName, err := splitKey(subnetKey)
	if err != nil {
		return Subnet{}, err
	}

	subnetYamlPath := filepath.Join(app.LpmDir, "repositories", repoAlias, "subnets", subnetName+".yaml")
	var subnetWrapper SubnetWrapper

	subnetYamlBytes, err := os.ReadFile(subnetYamlPath)
	if err != nil {
		return Subnet{}, err
	}

	err = yaml.Unmarshal(subnetYamlBytes, &subnetWrapper)
	if err != nil {
		return Subnet{}, err
	}

	return subnetWrapper.Subnet, nil
}

func getVMsInSubnet(app *application.Lux, subnetKey string) ([]string, error) {
	subnet, err := LoadSubnetFile(app, subnetKey)
	if err != nil {
		return []string{}, err
	}

	return subnet.VMs, nil
}

func LoadVMFile(app *application.Lux, repo, vm string) (VM, error) {
	vmYamlPath := filepath.Join(app.LpmDir, "repositories", repo, "vms", vm+".yaml")
	var vmWrapper VMWrapper

	vmYamlBytes, err := os.ReadFile(vmYamlPath)
	if err != nil {
		return VM{}, err
	}

	err = yaml.Unmarshal(vmYamlBytes, &vmWrapper)
	if err != nil {
		return VM{}, err
	}

	return vmWrapper.VM, nil
}
