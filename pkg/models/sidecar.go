// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/netrunner/utils"
)

type TokenInfo struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
	Supply   string `json:"supply"`
}

type NetworkData struct {
	SubnetID                   ids.ID
	BlockchainID               ids.ID
	RPCVersion                 int
	RPCEndpoints               []string // RPC endpoints for the network
	WSEndpoints                []string // WebSocket endpoints for the network
	TeleporterRegistryAddress  string   // Teleporter registry address
	TeleporterMessengerAddress string   // Teleporter messenger address
	ValidatorManagerAddress    string   // Validator manager contract address
}

type MultisigTxInfo struct {
	Threshold uint32   `json:"threshold"`
	Addresses []string `json:"addresses"`
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
	Name            string
	VM              VMType
	VMVersion       string
	RPCVersion      int
	Subnet          string
	SubnetID        ids.ID
	BlockchainID    ids.ID
	TokenName       string
	TokenSymbol     string
	ChainID         string
	Version         string
	Networks        map[string]NetworkData
	ElasticSubnet   map[string]ElasticSubnet
	ImportedFromLPM bool
	ImportedVMID    string

	// Custom VM support
	CustomVMRepoURL     string
	CustomVMBranch      string
	CustomVMBuildScript string

	// L1/L2 Architecture (2025)
	Sovereign     bool   `json:"sovereign"`     // true for L1, false for L2/subnet
	BaseChain     string `json:"baseChain"`     // For L2s: ethereum, lux-l1, lux, op-mainnet
	BasedRollup   bool   `json:"basedRollup"`   // true for L1-sequenced rollups
	SequencerType string `json:"sequencerType"` // based, centralized, distributed

	// Based Rollup Configuration
	InboxContract     string `json:"inboxContract"`     // Contract on base chain
	L1BlockTime       int    `json:"l1BlockTime"`       // Base chain block time in ms
	PreconfirmEnabled bool   `json:"preconfirmEnabled"` // Fast confirmations

	// Token & Economics
	TokenInfo  TokenInfo `json:"tokenInfo"`
	RentalPlan string    `json:"rentalPlan"` // For L1s: monthly, annual, perpetual

	// Validator Management
	ValidatorManagement string `json:"validatorManagement"` // proof-of-authority, proof-of-stake

	// Migration info
	MigratedAt int64 `json:"migratedAt"` // When subnet became L1

	// Chain layer (1=L1, 2=L2, 3=L3)
	ChainLayer int `json:"chainLayer"` // Default 2 for backward compat
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

// MigrationTx represents a subnet to L1 migration transaction
type MigrationTx struct {
	SubnetID            ids.ID `json:"subnetId"`
	BlockchainID        ids.ID `json:"blockchainId"`
	ValidatorManagement string `json:"validatorManagement"`
	RentalPlan          string `json:"rentalPlan"`
	Timestamp           int64  `json:"timestamp"`
}

// NetworkDataIsEmpty checks if the sidecar has no network data
func (sc *Sidecar) NetworkDataIsEmpty() bool {
	return sc.Networks == nil || len(sc.Networks) == 0
}
