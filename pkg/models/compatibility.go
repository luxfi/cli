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
type CLIDependencyMap map[string]string
