// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/ids"
	nodecfg "github.com/luxfi/node/config/node"
	"github.com/luxfi/node/staking"
	protosigner "github.com/luxfi/proto/p/signer"
)

// The NodeID this command prints is what an operator registers on chain, and
// the files it writes are what luxd boots with. If the two ever disagree the
// node joins under an identity nobody staked, so re-derive both from disk.
func TestStakerFilesCarryThePrintedIdentity(t *testing.T) {
	dir := t.TempDir()

	nodeID, pop, err := newStaker(dir)
	if err != nil {
		t.Fatal(err)
	}
	if nodeID == ids.EmptyNodeID {
		t.Fatal("empty NodeID")
	}
	if err := pop.Verify(); err != nil {
		t.Fatalf("printed proof of possession does not verify: %v", err)
	}

	cert, err := staking.LoadTLSCertFromFiles(
		filepath.Join(dir, stakerKey),
		filepath.Join(dir, stakerCert),
	)
	if err != nil {
		t.Fatal(err)
	}
	fromDisk, err := (&nodecfg.StakingConfig{StakingTLSCert: *cert}).DeriveNodeID(ids.Empty)
	if err != nil {
		t.Fatal(err)
	}
	if fromDisk != nodeID {
		t.Fatalf("cert on disk derives %s, printed %s", fromDisk, nodeID)
	}

	signerBytes, err := os.ReadFile(filepath.Join(dir, stakerSigner))
	if err != nil {
		t.Fatal(err)
	}
	sk, err := localsigner.FromBytes(signerBytes)
	if err != nil {
		t.Fatal(err)
	}
	fromKey, err := protosigner.NewProofOfPossession(sk)
	if err != nil {
		t.Fatal(err)
	}
	if fromKey.PublicKey != pop.PublicKey {
		t.Fatal("signer.key holds a different BLS key than the one printed")
	}

	// A second run must refuse rather than replace a staked identity.
	if _, _, err := newStaker(dir); err == nil {
		t.Fatal("overwrote an existing node identity")
	}
}
