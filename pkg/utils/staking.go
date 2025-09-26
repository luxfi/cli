// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package utils

import (
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	evmclient "github.com/luxfi/evm/plugin/evm/client"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/staking"
	"github.com/luxfi/node/vms/platformvm"
)

func NewBlsSecretKeyBytes() ([]byte, error) {
	blsSignerKey, err := localsigner.New()
	if err != nil {
		return nil, err
	}
	return blsSignerKey.ToBytes(), nil
}

func ToNodeID(certBytes []byte) (ids.NodeID, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return ids.EmptyNodeID, fmt.Errorf("failed to decode certificate")
	}
	cert, err := staking.ParseCertificate(block.Bytes)
	if err != nil {
		return ids.EmptyNodeID, err
	}
	idsCert := &ids.Certificate{
		Raw:       cert.Raw,
		PublicKey: cert.PublicKey,
	}
	return ids.NodeIDFromCert(idsCert), nil
}

func ToBLSPoP(keyBytes []byte) (
	[]byte, // bls public key
	[]byte, // bls proof of possession
	error,
) {
	localSigner, err := localsigner.FromBytes(keyBytes)
	if err != nil {
		return nil, nil, err
	}
	// LocalSigner has the secret key as a private field, but we can get the public key
	// and sign a proof of possession directly
	pk := localSigner.PublicKey()
	pkBytes := bls.PublicKeyToCompressedBytes(pk)
	sig, err := localSigner.SignProofOfPossession(pkBytes)
	if err != nil {
		return nil, nil, err
	}
	sigBytes := bls.SignatureToBytes(sig)
	return pkBytes, sigBytes, nil
}

// GetNodeParams returns node id, bls public key and bls proof of possession
func GetNodeParams(nodeDir string) (
	ids.NodeID,
	[]byte, // bls public key
	[]byte, // bls proof of possession
	error,
) {
	certBytes, err := os.ReadFile(filepath.Join(nodeDir, constants.StakerCertFileName))
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	nodeID, err := ToNodeID(certBytes)
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	blsKeyBytes, err := os.ReadFile(filepath.Join(nodeDir, constants.BLSKeyFileName))
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	blsPub, blsPoP, err := ToBLSPoP(blsKeyBytes)
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	return nodeID, blsPub, blsPoP, nil
}

func GetRemainingValidationTime(networkEndpoint string, nodeID ids.NodeID, subnetID ids.ID, startTime time.Time) (time.Duration, error) {
	ctx, cancel := GetAPIContext()
	defer cancel()
	platformCli := platformvm.NewClient(networkEndpoint)
	vs, err := platformCli.GetCurrentValidators(ctx, subnetID, nil)
	cancel()
	if err != nil {
		return 0, err
	}
	for _, v := range vs {
		if v.NodeID == nodeID {
			return time.Unix(int64(v.EndTime), 0).Sub(startTime), nil
		}
	}
	return 0, errors.New("nodeID not found in validator set: " + nodeID.String())
}

// GetL1ValidatorUptimeSeconds returns the uptime of the L1 validator
func GetL1ValidatorUptimeSeconds(rpcURL string, nodeID ids.NodeID) (uint64, error) {
	ctx, cancel := GetAPIContext()
	defer cancel()
	networkEndpoint, blockchainID, err := SplitLuxgoRPCURI(rpcURL)
	if err != nil {
		return 0, err
	}
	evmCli := evmclient.NewClient(networkEndpoint, blockchainID)
	validators, err := evmCli.GetCurrentValidators(ctx, []ids.NodeID{nodeID})
	if err != nil {
		return 0, err
	}
	if len(validators) > 0 {
		deductibleSeconds := uint64(constants.ValidatorUptimeDeductible.Seconds())
		if validators[0].UptimeSeconds > deductibleSeconds {
			return validators[0].UptimeSeconds - deductibleSeconds, nil
		}
		return 0, nil
	}

	return 0, errors.New("nodeID not found in validator set: " + nodeID.String())
}
