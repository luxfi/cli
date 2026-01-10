// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

import (
	"time"

	"github.com/luxfi/ids"
)

// ElasticChainConfig contains configuration for elastic chain transformations.
type ElasticChainConfig struct {
	ChainID                  ids.ID
	AssetID                  ids.ID
	InitialSupply            uint64
	MaxSupply                uint64
	MinConsumptionRate       uint64
	MaxConsumptionRate       uint64
	MinValidatorStake        uint64
	MaxValidatorStake        uint64
	MinStakeDuration         time.Duration
	MaxStakeDuration         time.Duration
	MinDelegationFee         uint32
	MinDelegatorStake        uint64
	MaxValidatorWeightFactor byte
	UptimeRequirement        uint32
}
