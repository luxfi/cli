// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package testutils

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/node/crypto"
)

func GenerateEthAddrs(count int) ([]common.Address, error) {
	addrs := make([]common.Address, count)
	for i := 0; i < count; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		addrs[i] = crypto.PubkeyToAddress(pk.PublicKey)
	}
	return addrs, nil
}
