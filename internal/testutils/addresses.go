// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package testutils

import (
	"github.com/luxfi/crypto"
)

func GenerateEthAddrs(count int) ([]crypto.Address, error) {
	addrs := make([]crypto.Address, count)
	for i := 0; i < count; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		addrs[i] = crypto.PubkeyToAddress(pk.PublicKey)
	}
	return addrs, nil
}
