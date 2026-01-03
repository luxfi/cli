// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

// ExportNode represents an exportable node configuration with keys.
type ExportNode struct {
	NodeConfig NodeConfig `json:"nodeConfig"`
	SignerKey  string     `json:"signerKey"`
	StakerKey  string     `json:"stakerKey"`
	StakerCrt  string     `json:"stakerCrt"`
}

// ExportCluster represents an exportable cluster configuration.
type ExportCluster struct {
	ClusterConfig ClusterConfig `json:"clusterConfig"`
	Nodes         []ExportNode  `json:"nodes"`
	MonitorNode   ExportNode    `json:"monitorNode"`
	LoadTestNodes []ExportNode  `json:"loadTestNodes"`
}
