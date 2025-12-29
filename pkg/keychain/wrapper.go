// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keychain

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	nodekeychain "github.com/luxfi/keychain"
	"github.com/luxfi/node/vms/secp256k1fx"
	walletkeychain "github.com/luxfi/node/wallet/keychain"
)

// NodeToLedgerWrapper wraps a node keychain to implement ledger keychain interface
type NodeToLedgerWrapper struct {
	nodeKC nodekeychain.Keychain
}

// WrapNodeKeychain wraps a node keychain to implement ledger keychain interface
func WrapNodeKeychain(nodeKC nodekeychain.Keychain) *NodeToLedgerWrapper {
	return &NodeToLedgerWrapper{nodeKC: nodeKC}
}

// Get returns the signer for the given address
func (w *NodeToLedgerWrapper) Get(addr ids.ShortID) (nodekeychain.Signer, bool) {
	signer, ok := w.nodeKC.Get(addr)
	if !ok {
		return nil, false
	}
	return signer, true
}

// Addresses returns the addresses managed by this keychain as a set
func (w *NodeToLedgerWrapper) Addresses() set.Set[ids.ShortID] {
	// Get the set from node keychain
	addrSet := w.nodeKC.Addresses()
	return addrSet
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

// Addresses returns the addresses managed by this keychain
func (w *Secp256k1fxToNodeWrapper) Addresses() set.Set[ids.ShortID] {
	return w.secpKC.Addresses()
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

// Addresses returns the addresses managed by this keychain as a set
func (w *NodeToWalletWrapper) Addresses() set.Set[ids.ShortID] {
	return w.nodeKC.Addresses()
}
