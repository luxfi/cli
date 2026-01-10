// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

// ChainValidator represents a validator configuration for a chain.
type ChainValidator struct {
	NodeID string `json:"NodeID"`

	Weight uint64 `json:"Weight"`

	Balance uint64 `json:"Balance"`

	BLSPublicKey string `json:"BLSPublicKey"`

	BLSProofOfPossession string `json:"BLSProofOfPossession"`

	ChangeOwnerAddr string `json:"ChangeOwnerAddr"`

	ValidationID string `json:"ValidationID"`
}
