// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
)

// ConvertSubnetToL1Validator contains validator information for subnet-to-L1 conversion
type ConvertSubnetToL1Validator struct {
	NodeID ids.NodeID `serialize:"true" json:"nodeID"`
	Weight uint64     `serialize:"true" json:"weight"`
	Signer Signer     `serialize:"true" json:"signer"`
}

// Signer contains the BLS signature for a validator
type Signer struct {
	PublicKey [bls.PublicKeyLen]byte `serialize:"true" json:"publicKey"`
	Signature [bls.SignatureLen]byte `serialize:"true" json:"signature"`
}