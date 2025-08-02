// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

// CLIDependencyMap maps network names to dependency versions
type CLIDependencyMap map[string]struct {
	Luxd      string `json:"luxd"`
	LPM       string `json:"lpm"`
	LuxdRPC int    `json:"luxdRPC"`
}

// LuxdCompatiblity represents compatibility information
type LuxdCompatiblity struct {
	RPCChainVMProtocolVersion map[string][]string `json:"rpcChainVMProtocolVersion"`
}