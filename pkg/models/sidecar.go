// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"github.com/luxdefi/netrunner/utils"
	"github.com/luxdefi/luxgo/ids"
)

type NetworkData struct {
	SubnetID     ids.ID
	BlockchainID ids.ID
	RPCVersion   int
}

type PermissionlessValidators struct {
	TxID ids.ID
}
type ElasticSubnet struct {
	SubnetID    ids.ID
	AssetID     ids.ID
	PChainTXID  ids.ID
	TokenName   string
	TokenSymbol string
	Validators  map[string]PermissionlessValidators
	Txs         map[string]ids.ID
}

type Sidecar struct {
	Name                string
	VM                  VMType
	VMVersion           string
	RPCVersion          int
	Subnet              string
	TokenName           string
	ChainID             string
	Version             string
	Networks            map[string]NetworkData
	ElasticSubnet       map[string]ElasticSubnet
	ImportedFromLPM     bool
	ImportedVMID        string
	CustomVMRepoURL     string
	CustomVMBranch      string
	CustomVMBuildScript string
	// SubnetEVM based VM's only
	SubnetEVMMainnetChainID uint
}

func (sc Sidecar) GetVMID() (string, error) {
	// get vmid
	var vmid string
	if sc.ImportedFromLPM {
		vmid = sc.ImportedVMID
	} else {
		chainVMID, err := utils.VMID(sc.Name)
		if err != nil {
			return "", err
		}
		vmid = chainVMID.String()
	}
	return vmid, nil
}
