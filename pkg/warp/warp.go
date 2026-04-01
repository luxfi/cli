// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package warp provides native Lux Warp messaging support for the CLI
package warp

import (
	"math/big"

	"github.com/luxfi/cli/pkg/application"
)

// WarpInfo contains information about Warp configuration
type WarpInfo struct {
	Version       string
	FundedAddress string
	FundedBalance *big.Int
}

// GetWarpInfo returns Warp configuration for the CLI
func GetWarpInfo(app *application.Lux) (*WarpInfo, error) {
	// Default warp configuration
	balance := new(big.Int)
	balance.SetString("600000000000000000000", 10) // 600 tokens

	return &WarpInfo{
		Version:       "v2.0.0",
		FundedAddress: "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714",
		FundedBalance: balance,
	}, nil
}
