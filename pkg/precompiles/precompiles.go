// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package precompiles

import (
	"github.com/luxfi/crypto/common"
	"github.com/luxfi/evm/precompile/contracts/nativeminter"
	"github.com/luxfi/evm/precompile/contracts/warp"
)

var (
	NativeMinterPrecompile = common.BytesToAddress(nativeminter.ContractAddress.Bytes())
	WarpPrecompile         = common.BytesToAddress(warp.ContractAddress.Bytes())
)
