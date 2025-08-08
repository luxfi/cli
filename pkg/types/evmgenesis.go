// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package types

import (
	"github.com/luxfi/evm/core"
)

// EvmGenesis represents an EVM genesis configuration
// It embeds core.Genesis for compatibility
type EvmGenesis struct {
	core.Genesis
}