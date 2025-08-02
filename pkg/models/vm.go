// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import "github.com/luxfi/cli/v2/v2/pkg/constants"

type VMType string

const (
	EVM         = "EVM"
	SubnetEvm   = "SubnetEVM"
	BlobVM      = "Blob VM"
	TimestampVM = "Timestamp VM"
	CustomVM    = "Custom"
)

func VMTypeFromString(s string) VMType {
	switch s {
	case EVM:
		return EVM
	case SubnetEvm:
		return SubnetEvm
	case BlobVM:
		return BlobVM
	case TimestampVM:
		return TimestampVM
	default:
		return CustomVM
	}
}

func (v VMType) RepoName() string {
	switch v {
	case EVM:
		return constants.EVMRepoName
	default:
		return "unknown"
	}
}
