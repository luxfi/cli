// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keychain

import (
	"github.com/luxfi/ids"
	nodekeychain "github.com/luxfi/node/utils/crypto/keychain"
	walletkeychain "github.com/luxfi/node/wallet/keychain"
	"github.com/luxfi/math/set"
	"github.com/luxfi/node/vms/secp256k1fx"
	ledgerkeychain "github.com/luxfi/ledger-lux-go/keychain"
)

// NodeToLedgerWrapper wraps a node keychain to implement ledger keychain interface
type NodeToLedgerWrapper struct {
	nodeKC nodekeychain.Keychain
}

// WrapNodeKeychain wraps a node keychain to implement ledger keychain interface
func WrapNodeKeychain(nodeKC nodekeychain.Keychain) ledgerkeychain.Keychain {
	return &NodeToLedgerWrapper{nodeKC: nodeKC}
}

// Get returns the signer for the given address
func (w *NodeToLedgerWrapper) Get(addr ids.ShortID) (ledgerkeychain.Signer, bool) {
	signer, ok := w.nodeKC.Get(addr)
	if !ok {
		return nil, false
	}
	// The node signer already implements the ledger signer interface
	return signer, true
}

// Addresses returns the addresses managed by this keychain as a slice
func (w *NodeToLedgerWrapper) Addresses() []ids.ShortID {
	// Convert set to slice
	addrSet := w.nodeKC.Addresses()
	return addrSet.List()
}

// Secp256k1fxToNodeWrapper wraps a secp256k1fx keychain to implement node keychain interface
type Secp256k1fxToNodeWrapper struct {
	secpKC *secp256k1fx.Keychain
}

// WrapSecp256k1fxKeychain wraps a secp256k1fx keychain to implement node keychain interface
func WrapSecp256k1fxKeychain(secpKC *secp256k1fx.Keychain) nodekeychain.Keychain {
	return &Secp256k1fxToNodeWrapper{secpKC: secpKC}
}

// Get returns the signer for the given address
func (w *Secp256k1fxToNodeWrapper) Get(addr ids.ShortID) (nodekeychain.Signer, bool) {
	return w.secpKC.Get(addr)
}

// Addresses returns the addresses managed by this keychain as a set
func (w *Secp256k1fxToNodeWrapper) Addresses() set.Set[ids.ShortID] {
	// Convert slice to set
	addrs := w.secpKC.Addresses()
	return set.Of(addrs...)
}

// NodeToWalletWrapper wraps a node keychain to implement wallet keychain interface
type NodeToWalletWrapper struct {
	nodeKC nodekeychain.Keychain
}

// WrapNodeToWalletKeychain wraps a node keychain to implement wallet keychain interface
func WrapNodeToWalletKeychain(nodeKC nodekeychain.Keychain) walletkeychain.Keychain {
	return &NodeToWalletWrapper{nodeKC: nodeKC}
}

// Get returns the signer for the given address
func (w *NodeToWalletWrapper) Get(addr ids.ShortID) (walletkeychain.Signer, bool) {
	signer, ok := w.nodeKC.Get(addr)
	if !ok {
		return nil, false
	}
	// The node signer already implements the wallet signer interface
	return signer, true
}

// Addresses returns the addresses managed by this keychain as a slice
func (w *NodeToWalletWrapper) Addresses() []ids.ShortID {
	// Convert set to slice
	addrSet := w.nodeKC.Addresses()
	return addrSet.List()
}
