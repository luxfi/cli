// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package binutils

import (
	"fmt"
	"time"

	"github.com/luxfi/constants"
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

// gRPC server and gateway endpoints using centralized constants
var (
	gRPCServerEndpoint  = fmt.Sprintf(":%d", constants.GRPCPortMainnet)
	gRPCGatewayEndpoint = fmt.Sprintf(":%d", constants.GRPCGatewayPortMainnet)
)

// Re-export port constants from centralized package
const (
	GRPCPortMainnet        = constants.GRPCPortMainnet
	GRPCPortTestnet        = constants.GRPCPortTestnet
	GRPCPortDevnet         = constants.GRPCPortDevnet
	GRPCPortCustom         = constants.GRPCPortCustom
	GRPCGatewayPortMainnet = constants.GRPCGatewayPortMainnet
	GRPCGatewayPortTestnet = constants.GRPCGatewayPortTestnet
	GRPCGatewayPortDevnet  = constants.GRPCGatewayPortDevnet
	GRPCGatewayPortCustom  = constants.GRPCGatewayPortCustom
)

// NetworkGRPCPorts is an alias to centralized type for backward compatibility
type NetworkGRPCPorts = constants.NetworkGRPCPorts

// GetGRPCPorts delegates to centralized constants package
func GetGRPCPorts(networkType string) NetworkGRPCPorts {
	return constants.GetGRPCPorts(networkType)
}
