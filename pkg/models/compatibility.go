// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package models

type VMCompatibility struct {
	RPCChainVMProtocolVersion map[string]int `json:"rpcChainVMProtocolVersion"`
}

type LuxCompatiblity map[string][]string

// LuxdCompatiblity is an alias for backward compatibility  
type LuxdCompatiblity = LuxCompatiblity

// CLIDependencyMap represents CLI dependency versions
type CLIDependencyMap struct {
	RPC       int                        `json:"rpc"`
	Luxd      map[string]NetworkVersions `json:"luxd"`
	SubnetEVM string                     `json:"subnetevm"`
}

// NetworkVersions represents versions for a network
type NetworkVersions struct {
	LatestVersion  string `json:"latestVersion"`
	MinimumVersion string `json:"minimumVersion"`
}
