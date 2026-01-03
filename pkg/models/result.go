// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

import "sync"

// NodeResult contains the result of an operation on a single node.
type NodeResult struct {
	NodeID string
	Value  interface{}
	Err    error
}

// NodeResults contains results from operations on multiple nodes.
type NodeResults struct {
	Results []NodeResult
	Lock    sync.Mutex
}

// AddResult adds a result for a node to the results collection.
func (nr *NodeResults) AddResult(nodeID string, value interface{}, err error) {
	nr.Lock.Lock()
	defer nr.Lock.Unlock()
	nr.Results = append(nr.Results, NodeResult{
		NodeID: nodeID,
		Value:  value,
		Err:    err,
	})
}

// GetResults returns all node results.
func (nr *NodeResults) GetResults() []NodeResult {
	nr.Lock.Lock()
	defer nr.Lock.Unlock()
	return nr.Results
}

// GetResultMap returns results as a map from node ID to value.
func (nr *NodeResults) GetResultMap() map[string]interface{} {
	nr.Lock.Lock()
	defer nr.Lock.Unlock()
	result := map[string]interface{}{}
	for _, node := range nr.Results {
		result[node.NodeID] = node.Value
	}
	return result
}

// Len returns the number of results.
func (nr *NodeResults) Len() int {
	nr.Lock.Lock()
	defer nr.Lock.Unlock()
	return len(nr.Results)
}

// GetNodeList returns a list of all node IDs.
func (nr *NodeResults) GetNodeList() []string {
	nr.Lock.Lock()
	defer nr.Lock.Unlock()
	nodes := []string{}
	for _, node := range nr.Results {
		nodes = append(nodes, node.NodeID)
	}
	return nodes
}

// GetErrorHostMap returns a map from node ID to error for nodes with errors.
func (nr *NodeResults) GetErrorHostMap() map[string]error {
	nr.Lock.Lock()
	defer nr.Lock.Unlock()
	hostErrors := make(map[string]error)
	for _, node := range nr.Results {
		if node.Err != nil {
			hostErrors[node.NodeID] = node.Err
		}
	}
	return hostErrors
}

// HasIDWithError returns true if the given node ID has an error.
func (nr *NodeResults) HasIDWithError(id string) bool {
	nr.Lock.Lock()
	defer nr.Lock.Unlock()
	for _, node := range nr.Results {
		if node.NodeID == id && node.Err != nil {
			return true
		}
	}
	return false
}

// HasErrors returns true if any node has an error.
func (nr *NodeResults) HasErrors() bool {
	return len(nr.GetErrorHostMap()) > 0
}

// GetErrorHosts returns the list of node IDs with errors.
func (nr *NodeResults) GetErrorHosts() []string {
	var nodes []string
	for _, node := range nr.Results {
		if node.Err != nil {
			nodes = append(nodes, node.NodeID)
		}
	}
	return nodes
}
