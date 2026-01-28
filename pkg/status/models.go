// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package status

import (
	"time"
)

// Network represents a Lux network (mainnet, testnet, devnet, custom)
type Network struct {
	Name          string
	Nodes         []Node
	Chains        []ChainStatus
	Endpoints     []EndpointStatus
	Metadata      NetworkMetadata
	Validators    []ValidatorAccount // Validator accounts with addresses and balances
	ActiveAccount *ActiveAccount     // Currently active account for operations
}

// NetworkMetadata contains additional network information
type NetworkMetadata struct {
	GRPCPort   int
	NodesCount int
	VMsCount   int
	Controller string // "on" or "off"
	Status     string // "up", "down", "stopped", "error"
	LastError  string // Error message if Status is "error"
}

// Node represents a network node
type Node struct {
	ID               string
	HTTPURL          string
	NodeID           string
	Version          string
	CoreVersion      string
	EVMVersion       string
	NetrunnerVersion string
	PeerCount        int
	Uptime           string
	OK               bool
	LatencyMS        int
	LastError        string
	GPUAccelerated   bool
	GPUDriverVersion string
	GPUDevice        string
	PChainAddress    string
	XChainAddress    string
	CChainAddress    string
	// Balances (in nLUX for P/X, wei for C)
	PChainBalance uint64
	XChainBalance uint64
	CChainBalance string // hex string for large balances
}

// ValidatorAccount represents a validator's addresses and balances
type ValidatorAccount struct {
	Index         int    `json:"index"`
	NodeID        string `json:"nodeID"`
	PChainAddress string `json:"pChainAddress"`
	XChainAddress string `json:"xChainAddress"`
	CChainAddress string `json:"cChainAddress"` // 0x format
	// Balances
	PChainBalance    uint64 `json:"pChainBalance"`    // nLUX
	XChainBalance    uint64 `json:"xChainBalance"`    // nLUX
	CChainBalance    string `json:"cChainBalance"`    // wei (hex)
	CChainBalanceLUX string `json:"cChainBalanceLUX"` // human readable
	// Staking info
	StakeWeight  uint64 `json:"stakeWeight"`
	DelegatorFee uint64 `json:"delegatorFee"`
	IsActive     bool   `json:"isActive"` // Is this the active account for operations
}

// ActiveAccount represents the currently active account for network operations
type ActiveAccount struct {
	Index         int    `json:"index"`
	PChainAddress string `json:"pChainAddress"`
	XChainAddress string `json:"xChainAddress"`
	CChainAddress string `json:"cChainAddress"`
}

// ChainStatus represents the status of a chain
type ChainStatus struct {
	Alias         string // "c", "p", "x", "dex", etc.
	Kind          string // "evm", "pchain", "xchain", "custom"
	Height        uint64
	BlockTime     *time.Time
	RPC_OK        bool
	LatencyMS     int
	ChainID       string
	Syncing       interface{} // bool or sync progress object
	Metadata      map[string]interface{}
	LastError     string
	PluginVersion string // For custom chains
	PluginName    string // For custom chains
	BlockchainID  string // For custom chains
	VMID          string // For custom chains
}

// EndpointStatus represents the status of an RPC endpoint
type EndpointStatus struct {
	ChainAlias string
	URL        string
	OK         bool
	LatencyMS  int
	LastError  string
}

// TrackedEVM represents a tracked EVM chain (Zoo, Hanzo, SPC, etc.)
type TrackedEVM struct {
	Name         string // zoo, hanzo, spc
	Network      string // mainnet, testnet
	RPCs         []string
	BlockchainID string // if available
	VMID         string // if available
}

// EVMStatus represents the status of a tracked EVM
type EVMStatus struct {
	Name            string
	Network         string
	ChainID         uint64
	Height          uint64
	LatestTime      *time.Time
	Syncing         interface{} // bool or sync progress object
	ClientVersion   string
	PluginVersion   string
	Endpoints       []EndpointStatus
	DriftDetected   bool
	ChainIDMismatch bool
}

// StatusResult contains the complete status information
type StatusResult struct {
	Networks    []Network
	TrackedEVMs []EVMStatus
	Timestamp   time.Time
	DurationMS  int
}

// ProbeResult contains the result of a single probe
type ProbeResult struct {
	OK        bool
	LatencyMS int
	Height    uint64
	Meta      map[string]interface{}
	Error     error
}
