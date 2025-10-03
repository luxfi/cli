// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import "github.com/luxfi/cli/pkg/constants"

type VMType string

const (
	EVM         = "EVM"
	SubnetEvm   = EVM // Alias for backward compatibility
	BlobVM      = "Blob VM"
	TimestampVM = "Timestamp VM"
	QuantumVM   = "Quantum VM"
	CustomVM    = "Custom"
)

func VMTypeFromString(s string) VMType {
	switch s {
	case EVM:
		return EVM
	case BlobVM:
		return BlobVM
	case TimestampVM:
		return TimestampVM
	case QuantumVM:
		return QuantumVM
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
