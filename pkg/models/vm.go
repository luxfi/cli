// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

import "github.com/luxfi/constants"

// VMType represents a virtual machine type.
type VMType string

// VM type constants.
const (
	// EVM is the Ethereum Virtual Machine (Go, geth/coreth, sequential).
	EVM = "EVM"
	// EVMGPU is the parallel EVM with GPU acceleration (Go, Block-STM).
	EVMGPU = "EVM-GPU"
	// CEVM is the C++ EVM with native GPU support (evmone, Metal/CUDA/WebGPU).
	CEVM = "CEVM"
	// REVM is the Rust EVM (reth/revm).
	REVM      = "REVM"
	BlobVM      = "Blob VM"
	TimestampVM = "Timestamp VM"
	QuantumVM   = "Quantum VM"
	ParsVM      = "Pars VM"
	CustomVM    = "Custom"
)

// EVMBackend selects which EVM implementation to use for a chain.
type EVMBackend string

const (
	EVMBackendDefault  EVMBackend = "evm"    // Go geth/coreth (production default)
	EVMBackendGPU      EVMBackend = "evmgpu" // Go Block-STM + GPU acceleration
	EVMBackendCEVM     EVMBackend = "cevm"   // C++ evmone + Metal/CUDA GPU
	EVMBackendREVM     EVMBackend = "revm"   // Rust reth/revm
)

// VMTypeFromString returns a VMType from its string representation.
func VMTypeFromString(s string) VMType {
	switch s {
	case EVM:
		return EVM
	case EVMGPU, "evmgpu", "gpu":
		return EVMGPU
	case CEVM, "cevm", "cpp":
		return CEVM
	case REVM, "revm", "rust":
		return REVM
	case BlobVM:
		return BlobVM
	case TimestampVM:
		return TimestampVM
	case QuantumVM:
		return QuantumVM
	case ParsVM:
		return ParsVM
	default:
		return CustomVM
	}
}

// RepoName returns the repository name for the VM type.
func (v VMType) RepoName() string {
	switch v {
	case EVM:
		return constants.EVMRepoName
	case EVMGPU:
		return "evmgpu"
	case CEVM:
		return "evm" // github.com/luxcpp/evm
	case REVM:
		return "evm" // github.com/hanzoai/evm
	case ParsVM:
		return "node"
	default:
		return "unknown"
	}
}

// Org returns the GitHub organization for the VM type.
func (v VMType) Org() string {
	switch v {
	case CEVM:
		return "luxcpp"
	case REVM:
		return "hanzoai"
	case ParsVM:
		return "parsdao"
	default:
		return constants.LuxOrg
	}
}

// IsGPUCapable returns true if this VM type supports GPU acceleration.
func (v VMType) IsGPUCapable() bool {
	return v == EVMGPU || v == CEVM
}
