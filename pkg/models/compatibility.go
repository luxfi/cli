// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package models

type VMCompatibility struct {
	RPCChainVMProtocolVersion map[string]int `json:"rpcChainVMProtocolVersion"`
}

type LuxdCompatiblity map[string][]string
