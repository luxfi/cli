// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

// DefaultsKind represents the type of defaults to use for VM configuration
type DefaultsKind int

const (
	// NoDefaults means no default configuration will be applied
	NoDefaults DefaultsKind = iota
	// TestDefaults means test-friendly defaults will be applied
	TestDefaults
	// ProductionDefaults means production-ready defaults will be applied
	ProductionDefaults
)

// EVM configuration constants
const (
	EvmDebugConfig    = `{"log-level": "debug"}`
	EvmNonDebugConfig = `{"log-level": "info"}`
)