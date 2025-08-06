// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

// Reserved for future use with IDs
// "github.com/luxfi/ids"

// L2Config represents a based rollup configuration
type L2Config struct {
	Name              string `json:"name"`
	BaseChain         string `json:"baseChain"`        // ethereum, lux, lux-l1, op-mainnet
	RollupType        string `json:"rollupType"`       // optimistic, zk, hybrid
	DataAvailability  string `json:"dataAvailability"` // base, celestia, eigenda
	IBCEnabled        bool   `json:"ibcEnabled"`
	PreconfirmEnabled bool   `json:"preconfirmEnabled"`
	BasedRollup       bool   `json:"basedRollup"` // true for L1-sequenced

	// Contracts
	InboxContract  string `json:"inboxContract"`
	RollupContract string `json:"rollupContract"`
	BridgeContract string `json:"bridgeContract"`

	// Chain configuration
	ChainID     uint64 `json:"chainId"`
	L1BlockTime int    `json:"l1BlockTime"` // milliseconds

	// Token info
	TokenInfo TokenInfo `json:"tokenInfo"`

	// Fee configuration
	CongestionFeeShare int `json:"congestionFeeShare"` // percentage to rollup

	// Bridge configuration
	EnabledBridges []string `json:"enabledBridges"` // axelar, layerzero, wormhole, etc
	IBCChannels    []string `json:"ibcChannels"`    // IBC channel IDs

	// Deployment info
	DeployedAt    int64 `json:"deployedAt"`
	LastMigration int64 `json:"lastMigration"`
}

// PreconfirmConfig represents pre-confirmation settings
type PreconfirmConfig struct {
	Enabled          bool   `json:"enabled"`
	Provider         string `json:"provider"`         // eigenlayer, builders, bonded
	ConfirmationTime int    `json:"confirmationTime"` // target ms
	CommitteeSize    int    `json:"committeeSize"`
	BondAmount       string `json:"bondAmount"` // in LUX
}

// BaseMigration represents a base chain migration
type BaseMigration struct {
	FromBase        string `json:"fromBase"`
	ToBase          string `json:"toBase"`
	ProposalID      string `json:"proposalId"`
	ExecutedAt      int64  `json:"executedAt"`
	HotSwap         bool   `json:"hotSwap"`
	CheckpointBlock uint64 `json:"checkpointBlock"`
}

// L3Config for nested rollups
type L3Config struct {
	Name   string `json:"name"`
	L2Base string `json:"l2Base"` // which L2 is the base
	// Inherits most properties from L2Config
	L2Config
}

// BridgeRoute represents a cross-chain route
type BridgeRoute struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Via           string `json:"via"`           // bridge provider
	EstimatedTime int    `json:"estimatedTime"` // seconds
	EstimatedCost string `json:"estimatedCost"` // in native token
}

// GovernanceProposal for base migrations and upgrades
type GovernanceProposal struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // base-migration, upgrade, parameter
	Title       string `json:"title"`
	Description string `json:"description"`
	L2Name      string `json:"l2Name"`

	// For base migrations
	CurrentBase string `json:"currentBase,omitempty"`
	TargetBase  string `json:"targetBase,omitempty"`
	HotSwap     bool   `json:"hotSwap,omitempty"`

	// Voting
	CreatedAt  int64 `json:"createdAt"`
	VotingEnds int64 `json:"votingEnds"`
	Executed   bool  `json:"executed"`
	ExecutedAt int64 `json:"executedAt"`
}
