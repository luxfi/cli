// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package testutils provides test utilities for the CLI.
package testutils

import (
	"github.com/luxfi/crypto"
)

// GenerateKeccakAddrs returns `count` test 20-byte addresses derived
// as Keccak256(uncompressed_secp256k1_pubkey)[12:] — the EVM-runtime
// address format. Naming: the derivation primitive (Keccak) is what
// determines the value, not the brand of any chain that consumes it.
func GenerateKeccakAddrs(count int) ([]crypto.Address, error) {
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

// GenerateEthAddrs is the deprecated alias for GenerateKeccakAddrs.
//
// Deprecated: use GenerateKeccakAddrs.
func GenerateEthAddrs(count int) ([]crypto.Address, error) {
	return GenerateKeccakAddrs(count)
}
