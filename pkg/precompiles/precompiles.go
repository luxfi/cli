// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package precompiles

import (
	_ "embed"

	"github.com/luxfi/evm/v2/precompile/contracts/nativeminter"
	"github.com/luxfi/evm/v2/precompile/contracts/warp"
)

var (
	NativeMinterPrecompile = nativeminter.ContractAddress
	WarpPrecompile         = warp.ContractAddress
)
