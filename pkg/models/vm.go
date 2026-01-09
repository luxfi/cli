// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

import "github.com/luxfi/constants"

// VMType represents a virtual machine type.
type VMType string

// VM type constants.
const (
	// EVM is the Ethereum Virtual Machine.
	EVM         = "EVM"
	BlobVM      = "Blob VM"
	TimestampVM = "Timestamp VM"
	QuantumVM   = "Quantum VM"
	CustomVM    = "Custom"
)

// VMTypeFromString returns a VMType from its string representation.
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

// RepoName returns the repository name for the VM type.
func (v VMType) RepoName() string {
	switch v {
	case EVM:
		return constants.EVMRepoName
	default:
		return "unknown"
	}
}
