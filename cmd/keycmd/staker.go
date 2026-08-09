// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/ids"
	nodecfg "github.com/luxfi/node/config/node"
	"github.com/luxfi/node/staking"
	protosigner "github.com/luxfi/proto/p/signer"
	"github.com/spf13/cobra"
)

// A node's identity is two keys under one directory, named as luxd expects
// them: the TLS pair (staker.crt/staker.key) the NodeID hashes from, and the
// BLS key (signer.key) whose proof of possession the P-Chain records.
const (
	stakerCert   = "staker.crt"
	stakerKey    = "staker.key"
	stakerSigner = "signer.key"
)

func newStakerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "staker <dir>",
		Short: "Create a node identity: staking cert plus BLS key",
		Long: `Writes ` + stakerCert + `, ` + stakerKey + ` and ` + stakerSigner + ` into <dir> and prints the
NodeID, BLS public key and proof of possession.

Point luxd at the directory, then register the printed values:

  lux key staker ~/.luxd/staking
  lux primary addValidator --mainnet \
      --node-id <NodeID> --public-key <pub> --proof-of-possession <pop> \
      --stake 2 --duration 336h`,
		Args: cobrautils.ExactArgs(1),
		RunE: runStaker,
	}
}

func runStaker(_ *cobra.Command, args []string) error {
	nodeID, pop, err := newStaker(args[0])
	if err != nil {
		return err
	}
	popJSON, err := json.Marshal(pop)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("Node identity written to %s", args[0])
	ux.Logger.PrintToUser("  NodeID: %s", nodeID)
	ux.Logger.PrintToUser("  BLS:    %s", popJSON)
	return nil
}

// newStaker writes a fresh identity into dir and returns the two values that
// identify it on chain: the NodeID the staking cert hashes to, and the BLS
// proof of possession.
func newStaker(dir string) (ids.NodeID, *protosigner.ProofOfPossession, error) {
	certPath := filepath.Join(dir, stakerCert)
	keyPath := filepath.Join(dir, stakerKey)
	signerPath := filepath.Join(dir, stakerSigner)

	// An identity is worth more than the command that made it: a second run
	// must not silently replace a NodeID that is already staked.
	for _, p := range []string{certPath, keyPath, signerPath} {
		if _, err := os.Stat(p); err == nil {
			return ids.EmptyNodeID, nil, fmt.Errorf("%s already exists: refusing to replace a node identity", p)
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ids.EmptyNodeID, nil, err
	}

	certBytes, keyBytes, err := staking.NewCertAndKeyBytes()
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}
	cert, err := staking.LoadTLSCertFromBytes(keyBytes, certBytes)
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}
	nodeID, err := (&nodecfg.StakingConfig{StakingTLSCert: *cert}).DeriveNodeID(ids.Empty)
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}

	sk, err := localsigner.New()
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}
	pop, err := protosigner.NewProofOfPossession(sk)
	if err != nil {
		return ids.EmptyNodeID, nil, err
	}

	for _, f := range []struct {
		path  string
		bytes []byte
	}{
		{certPath, certBytes},
		{keyPath, keyBytes},
		{signerPath, sk.ToBytes()},
	} {
		if err := os.WriteFile(f.path, f.bytes, 0o600); err != nil {
			return ids.EmptyNodeID, nil, err
		}
	}
	return nodeID, pop, nil
}
