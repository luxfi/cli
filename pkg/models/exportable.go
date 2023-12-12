// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package models

type Exportable struct {
	Sidecar         Sidecar
	Genesis         []byte
	ChainConfig     []byte
	SubnetConfig    []byte
	NetworkUpgrades []byte
	NodeConfig      []byte
}
