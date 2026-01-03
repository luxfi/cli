// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

// ClustersConfigV0 is a legacy clusters configuration format for backwards compatibility.
type ClustersConfigV0 struct {
	KeyPair   map[string]string   // maps key pair name to cert path
	Clusters  map[string][]string // maps clusterName to nodeID list
	GCPConfig GCPConfig           // stores GCP project name and filepath to service account JSON key
}
