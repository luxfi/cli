// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

import (
	"slices"

	"github.com/luxfi/constantsants"
)

// filter is a helper function to filter slices based on a predicate
func filter[T any](input []T, f func(T) bool) []T {
	output := make([]T, 0, len(input))
	for _, e := range input {
		if f(e) {
			output = append(output, e)
		}
	}
	return output
}

// GCPConfig contains Google Cloud Platform configuration settings.
type GCPConfig struct {
	ProjectName        string // name of GCP Project
	ServiceAccFilePath string // location of GCP service account key file path
}

// ExtraNetworkData contains additional network-specific data.
type ExtraNetworkData struct {
	CChainTeleporterMessengerAddress string
	CChainTeleporterRegistryAddress  string
}

// ClusterConfig contains configuration for a deployment cluster.
type ClusterConfig struct {
	Nodes              []string
	APINodes           []string
	Network            Network
	MonitoringInstance string            // instance ID of the separate monitoring instance (if any)
	LoadTestInstance   map[string]string // maps load test name to load test cloud instance ID of the separate load test instance (if any)
	ExtraNetworkData   ExtraNetworkData
	Subnets            []string
	External           bool
	Local              bool
	HTTPAccess         constants.HTTPAccess
}

// ClustersConfig contains configuration for all deployment clusters.
type ClustersConfig struct {
	Version   string
	KeyPair   map[string]string        // maps key pair name to cert path
	Clusters  map[string]ClusterConfig // maps clusterName to nodeID list + network kind
	GCPConfig GCPConfig                // stores GCP project name and filepath to service account JSON key
}

// GetAPIHosts returns a filtered list of API hosts from the given hosts.
func (cc *ClusterConfig) GetAPIHosts(hosts []*Host) []*Host {
	return filter(hosts, func(h *Host) bool {
		return slices.Contains(cc.APINodes, h.NodeID)
	})
}

// GetValidatorHosts returns the validator hosts (non-API nodes) from the given hosts.
func (cc *ClusterConfig) GetValidatorHosts(hosts []*Host) []*Host {
	return filter(hosts, func(h *Host) bool {
		return !slices.Contains(cc.APINodes, h.GetCloudID())
	})
}

// IsAPIHost returns true if the given cloud ID corresponds to an API host.
func (cc *ClusterConfig) IsAPIHost(hostCloudID string) bool {
	return cc.Local || slices.Contains(cc.APINodes, hostCloudID)
}

// IsLuxdHost returns true if the given cloud ID corresponds to a Luxd host.
func (cc *ClusterConfig) IsLuxdHost(hostCloudID string) bool {
	return cc.Local || slices.Contains(cc.Nodes, hostCloudID)
}

// GetCloudIDs returns all cloud instance IDs in the cluster.
func (cc *ClusterConfig) GetCloudIDs() []string {
	if cc.Local {
		return nil
	}
	r := cc.Nodes
	if cc.MonitoringInstance != "" {
		r = append(r, cc.MonitoringInstance)
	}
	return r
}

// GetHostRoles returns the roles assigned to a host based on its configuration.
func (cc *ClusterConfig) GetHostRoles(nodeConf NodeConfig) []string {
	roles := []string{}
	if cc.IsLuxdHost(nodeConf.NodeID) {
		if cc.IsAPIHost(nodeConf.NodeID) {
			roles = append(roles, constants.APIRole)
		} else {
			roles = append(roles, constants.ValidatorRole)
		}
	}
	if nodeConf.IsMonitor {
		roles = append(roles, constants.MonitorRole)
	}
	if nodeConf.IsWarpRelayer {
		roles = append(roles, constants.WarpRelayerRole)
	}
	return roles
}
