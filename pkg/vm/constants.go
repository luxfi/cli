// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"
)

// Units
var OneLux = big.NewInt(1_000_000_000) // 1 LUX = 1e9 nLUX

// Gas limits
const (
	LowGasLimit    = uint64(12_000_000)
	MediumGasLimit = uint64(15_000_000)
	HighGasLimit   = uint64(20_000_000)
	
	LowTargetGas    = uint64(1_500_000)
	MediumTargetGas = uint64(2_000_000)
	HighTargetGas   = uint64(5_000_000)
	
	NoDynamicFeesGasLimitToTargetGasFactor = uint64(10)
)

// GetVMBinaryProtocolVersion returns the protocol version for the VM binary
func GetVMBinaryProtocolVersion(vmBinaryPath string) (int, error) {
	// TODO: Implement VM binary protocol version detection
	// This would involve running the VM binary with a version flag
	// For now, return a default RPC version
	return 37, nil // Latest RPC version
}