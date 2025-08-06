// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// LPM (Lux Plugin Manager) client wrapper for CLI integration
package lpm

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	luxlpm "github.com/luxfi/lpm/lpm"
	"github.com/spf13/afero"
)

// Client wraps the LPM functionality for CLI use
type Client struct {
	lpm *luxlpm.LPM
}

// NewClient creates a new LPM client
func NewClient(lpmDir string, pluginDir string, adminAPIEndpoint string) (*Client, error) {
	config := luxlpm.Config{
		Directory:        lpmDir,
		Auth:             http.BasicAuth{},
		AdminAPIEndpoint: adminAPIEndpoint,
		PluginDir:        pluginDir,
		Fs:               afero.NewOsFs(),
	}

	lpmInstance, err := luxlpm.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LPM instance: %w", err)
	}

	return &Client{lpm: lpmInstance}, nil
}

// AddRepository adds a new repository
func (c *Client) AddRepository(alias string, url string, branch string) error {
	return c.lpm.AddRepository(alias, url, branch)
}

// Update updates all repositories
func (c *Client) Update() error {
	return c.lpm.Update()
}

// Install installs a plugin/VM
func (c *Client) Install(alias string) error {
	return c.lpm.Install(alias)
}

// Uninstall removes a plugin/VM
func (c *Client) Uninstall(alias string) error {
	return c.lpm.Uninstall(alias)
}

// Upgrade upgrades plugins/VMs
func (c *Client) Upgrade(alias string) error {
	return c.lpm.Upgrade(alias)
}

// ListRepositories lists all configured repositories
func (c *Client) ListRepositories() error {
	return c.lpm.ListRepositories()
}

// JoinSubnet installs all VMs required for a subnet
func (c *Client) JoinSubnet(alias string) error {
	return c.lpm.JoinSubnet(alias)
}

// Placeholder methods to maintain compatibility with existing LPM interface

// GetVM is a placeholder to maintain compatibility
func (c *Client) GetVM(alias string, version string) (*VMUpload, error) {
	return nil, fmt.Errorf("GetVM not implemented in LPM - use Install instead")
}

// AddVM is a placeholder to maintain compatibility
func (c *Client) AddVM(vm *VMUpload) error {
	return fmt.Errorf("AddVM not implemented in LPM - use repository-based installation")
}
