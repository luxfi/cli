// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package txutils provides transaction utilities for creating, signing, and managing transactions.
package txutils

import (
	"fmt"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/protocol/p/txs"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/luxfi/vm/components/verify"
)

// GetAuthSigners returns all chain auth addresses that are required to sign a given tx.
// It gets chain control keys as string slice using P-Chain API (GetOwners),
// gets chain auth indices from the tx (field tx.UnsignedTx.ChainAuth),
// and creates the string slice of required chain auth addresses by applying
// the indices to the control keys slice.
//
// Expected tx.Unsigned types: txs.CreateChainTx, txs.AddChainValidatorTx, txs.RemoveChainValidatorTx.
// controlKeys must be in the same order as in the chain creation tx (as obtained by GetOwners).
func GetAuthSigners(tx *txs.Tx, controlKeys []string) ([]string, error) {
	unsignedTx := tx.Unsigned
	var chainAuth verify.Verifiable
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveChainValidatorTx:
		chainAuth = unsignedTx.ChainAuth
	case *txs.AddChainValidatorTx:
		chainAuth = unsignedTx.ChainAuth
	case *txs.CreateChainTx:
		chainAuth = unsignedTx.ChainAuth
	case *txs.ConvertChainToL1Tx:
		chainAuth = unsignedTx.ChainAuth
	default:
		return nil, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	chainInput, ok := chainAuth.(*secp256k1fx.Input)
	if !ok {
		return nil, fmt.Errorf("expected chainAuth of type *secp256k1fx.Input, got %T", chainAuth)
	}
	authSigners := []string{}
	for _, sigIndex := range chainInput.SigIndices {
		if sigIndex >= uint32(len(controlKeys)) { //nolint:gosec // G115: Length is small, safe conversion
			return nil, fmt.Errorf("signer index %d exceeds number of control keys", sigIndex)
		}
		authSigners = append(authSigners, controlKeys[sigIndex])
	}
	return authSigners, nil
}

// GetRemainingSigners returns chain auth addresses that have not yet signed a given tx.
// It verifies that all creds in tx.Creds (except the last one) are fully signed,
// and computes remaining signers by iterating the last cred in tx.Creds.
// If the tx is fully signed, returns empty slice.
//
// controlKeys must be in the same order as in the chain creation tx (as obtained by GetOwners).
func GetRemainingSigners(tx *txs.Tx, controlKeys []string) ([]string, []string, error) {
	authSigners, err := GetAuthSigners(tx, controlKeys)
	if err != nil {
		return nil, nil, err
	}
	emptySig := [secp256k1.SignatureLen]byte{}
	// we should have at least 1 cred for output owners and 1 cred for chain auth
	if len(tx.Creds) < 2 {
		return nil, nil, fmt.Errorf("expected tx.Creds of len 2, got %d", len(tx.Creds))
	}
	// signatures for output owners should be filled (all creds except last one)
	for credIndex := range tx.Creds[:len(tx.Creds)-1] {
		cred, ok := tx.Creds[credIndex].(*secp256k1fx.Credential)
		if !ok {
			return nil, nil, fmt.Errorf("expected cred to be of type *secp256k1fx.Credential, got %T", tx.Creds[credIndex])
		}
		for i, sig := range cred.Sigs {
			if sig == emptySig {
				return nil, nil, fmt.Errorf("expected funding sig %d of cred %d to be filled", i, credIndex)
			}
		}
	}
	// signatures for chain auth (last cred)
	cred, ok := tx.Creds[len(tx.Creds)-1].(*secp256k1fx.Credential)
	if !ok {
		return nil, nil, fmt.Errorf("expected cred to be of type *secp256k1fx.Credential, got %T", tx.Creds[1])
	}
	if len(cred.Sigs) != len(authSigners) {
		return nil, nil, fmt.Errorf("expected number of cred's signatures %d to equal number of auth signers %d",
			len(cred.Sigs),
			len(authSigners),
		)
	}
	remainingSigners := []string{}
	for i, sig := range cred.Sigs {
		if sig == emptySig {
			remainingSigners = append(remainingSigners, authSigners[i])
		}
	}
	return authSigners, remainingSigners, nil
}
