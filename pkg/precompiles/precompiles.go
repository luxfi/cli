// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package precompiles

import (
	_ "embed"

	"github.com/luxfi/crypto"
	"github.com/luxfi/evm/precompile/contracts/nativeminter"
	"github.com/luxfi/evm/precompile/contracts/warp"
)

var (
	NativeMinterPrecompile = crypto.BytesToAddress(nativeminter.ContractAddress.Bytes())
	WarpPrecompile         = crypto.BytesToAddress(warp.ContractAddress.Bytes())
)
