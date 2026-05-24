// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package testutils provides test utilities for the CLI.
package testutils

import (
	"github.com/luxfi/crypto"
)

// GenerateEVMAddrs returns `count` test 20-byte EVM-runtime account
// addresses. The internal derivation hashes a fresh secp256k1 pubkey
// with Keccak256 (that's HOW); the values ARE EVM-runtime account
// addresses consumed by every EVM-compatible chain (that's WHAT).
func GenerateEVMAddrs(count int) ([]crypto.Address, error) {
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
