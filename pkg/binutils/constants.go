// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"fmt"
	"time"

	constants "github.com/luxfi/const"
)

const (
	nodeBinPrefix = "node-"
	evmBinPrefix  = "evm-"
	maxCopy       = 2147483648 // 2 GB

	// gRPC client configuration
	gRPCClientLogLevel = constants.GRPCClientLogLevel
	gRPCDialTimeout    = constants.GRPCDialTimeout
)

// Verify constant types at compile time
var (
	_ string        = gRPCClientLogLevel
	_ time.Duration = gRPCDialTimeout
)

// gRPC server endpoint using centralized constants
var gRPCServerEndpoint = fmt.Sprintf(":%d", constants.GRPCPortMainnet)

// Re-export port constants from centralized package
// Port scheme: aligned with chain IDs (8368-8371 for gRPC, 8378-8381 for gateway)
// - 8368/8378: testnet (chain ID 96368)
// - 8369/8379: mainnet (chain ID 96369)
// - 8370/8380: devnet (chain ID 96370)
// - 8371/8381: custom/local (chain ID 1337)
const (
	GRPCPortMainnet        = constants.GRPCPortMainnet
	GRPCPortTestnet        = constants.GRPCPortTestnet
	GRPCPortDevnet         = constants.GRPCPortDevnet
	GRPCPortCustom         = constants.GRPCPortCustom
	GRPCGatewayPortMainnet = constants.GRPCGatewayPortMainnet
	GRPCGatewayPortTestnet = constants.GRPCGatewayPortTestnet
	GRPCGatewayPortDevnet  = constants.GRPCGatewayPortDevnet
	GRPCGatewayPortCustom  = constants.GRPCGatewayPortCustom

	// Aliases for backward compatibility
	// "local" is deprecated, use "custom" instead
	GRPCPortLocal        = constants.GRPCPortLocal        // deprecated: use GRPCPortCustom
	GRPCGatewayPortLocal = constants.GRPCGatewayPortLocal // deprecated: use GRPCGatewayPortCustom
)

// NetworkGRPCPorts is an alias to centralized type for backward compatibility
type NetworkGRPCPorts = constants.NetworkGRPCPorts

// GetGRPCPorts delegates to centralized constants package
func GetGRPCPorts(networkType string) NetworkGRPCPorts {
	return constants.GetGRPCPorts(networkType)
}
