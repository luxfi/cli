// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chainvalidators provides typed chain validator operations.
package chainvalidators

import (
	"encoding/hex"
	"fmt"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/platformvm/signer"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/models"
)

// ChainValidator represents a typed validator for a chain.
type ChainValidator struct {
	NodeID  ids.NodeID
	Weight  uint64
	Balance uint64
}

// FromModels converts SDK validators to typed ChainValidators.
func FromModels(vs []models.Validator) ([]ChainValidator, error) {
	out := make([]ChainValidator, len(vs))
	for i, v := range vs {
		nid, err := ids.NodeIDFromString(v.NodeID)
		if err != nil {
			return nil, fmt.Errorf("invalid node ID %q: %w", v.NodeID, err)
		}
		out[i] = ChainValidator{
			NodeID:  nid,
			Weight:  v.Weight,
			Balance: v.Balance,
		}
	}
	return out, nil
}

// ToL1Validators converts SDK validators to node L1 validator format.
// This is the format required for P-Chain transactions.
func ToL1Validators(vs []models.Validator) ([]*txs.ConvertChainToL1Validator, error) {
	result := make([]*txs.ConvertChainToL1Validator, len(vs))
	for i, v := range vs {
		nodeID, err := ids.NodeIDFromString(v.NodeID)
		if err != nil {
			return nil, fmt.Errorf("invalid node ID %s: %w", v.NodeID, err)
		}

		// Parse BLS public key
		blsKey, err := hex.DecodeString(trimHexPrefix(v.BLSPublicKey))
		if err != nil {
			return nil, fmt.Errorf("invalid BLS public key: %w", err)
		}
		var blsKeyBytes [48]byte
		copy(blsKeyBytes[:], blsKey)

		// Parse BLS proof of possession
		pop, err := hex.DecodeString(trimHexPrefix(v.BLSProofOfPossession))
		if err != nil {
			return nil, fmt.Errorf("invalid BLS proof of possession: %w", err)
		}
		var popBytes [96]byte
		copy(popBytes[:], pop)

		result[i] = &txs.ConvertChainToL1Validator{
			NodeID:  nodeID[:],
			Weight:  v.Weight,
			Balance: v.Balance,
			Signer: signer.ProofOfPossession{
				PublicKey:         blsKeyBytes,
				ProofOfPossession: popBytes,
			},
		}
	}
	return result, nil
}

// trimHexPrefix removes 0x prefix from hex strings.
func trimHexPrefix(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}
