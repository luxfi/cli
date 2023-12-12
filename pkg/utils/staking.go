// Copyright (C) 2023, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package utils

import (
	"github.com/luxdefi/luxgo/ids"
	"github.com/luxdefi/luxgo/staking"
	"github.com/luxdefi/luxgo/utils/crypto/bls"
)

func NewBlsSecretKeyBytes() ([]byte, error) {
	blsSignerKey, err := bls.NewSecretKey()
	if err != nil {
		return nil, err
	}
	return bls.SecretKeyToBytes(blsSignerKey), nil
}

func ToNodeID(certBytes []byte, keyBytes []byte) (ids.NodeID, error) {
	cert, err := staking.LoadTLSCertFromBytes(keyBytes, certBytes)
	if err != nil {
		return ids.NodeID{}, err
	}
	return ids.NodeIDFromCert(staking.CertificateFromX509(cert.Leaf)), nil
}
