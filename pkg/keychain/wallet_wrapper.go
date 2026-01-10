// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keychain

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/keychain"
	wallkeychain "github.com/luxfi/keychain"
	"github.com/luxfi/math/set"
)

// CryptoToWalletWrapper wraps a crypto keychain to implement wallet keychain interface
type CryptoToWalletWrapper struct {
	cryptoKC keychain.Keychain
}

// WrapCryptoKeychain wraps a crypto keychain to implement wallet keychain interface
func WrapCryptoKeychain(cryptoKC keychain.Keychain) wallkeychain.Keychain {
	return &CryptoToWalletWrapper{cryptoKC: cryptoKC}
}

// Get returns the signer for the given address
func (w *CryptoToWalletWrapper) Get(addr ids.ShortID) (wallkeychain.Signer, bool) {
	return w.cryptoKC.Get(addr)
}

// Addresses returns the addresses managed by this keychain as a set
func (w *CryptoToWalletWrapper) Addresses() set.Set[ids.ShortID] {
	return w.cryptoKC.Addresses()
}
