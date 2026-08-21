// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package primarycmd

import (
	"testing"

	"github.com/luxfi/address"
)

// An unstated --reward-address must stay nil rather than becoming some
// stand-in owner: nil is what tells the deployer to fall back to the paying
// wallet, and any address invented here would silently redirect the reward.
func TestRewardOwnerUnstatedIsNil(t *testing.T) {
	owner, err := rewardOwner("")
	if err != nil {
		t.Fatalf("empty address: %v", err)
	}
	if owner != nil {
		t.Fatalf("empty address produced an owner (%v); nil is what defers to the payer", owner)
	}
}

// A stated address must reach the transaction as exactly that address, held
// alone. Threshold 1 over a single key is the whole point — a reward nobody
// can claim, or one two parties must agree to claim, is a different promise
// than the one the flag makes.
func TestRewardOwnerStatedAddress(t *testing.T) {
	// The genesis idx0 P-Chain address, used here only as a well-formed
	// mainnet-HRP value with a known decoding.
	const addr = "P-lux1qsrd262r5w9dswv2wwzj0un79ncpwvdgkpqzqu"

	owner, err := rewardOwner(addr)
	if err != nil {
		t.Fatalf("parsing %s: %v", addr, err)
	}
	if owner == nil {
		t.Fatal("a stated address produced no owner")
	}
	if owner.Threshold != 1 {
		t.Errorf("threshold = %d, want 1", owner.Threshold)
	}
	if len(owner.Addrs) != 1 {
		t.Fatalf("got %d owners, want exactly the one named", len(owner.Addrs))
	}
	want, err := address.ParseToID(addr)
	if err != nil {
		t.Fatalf("parsing the expected value: %v", err)
	}
	if owner.Addrs[0] != want {
		t.Errorf("owner = %s, want %s", owner.Addrs[0], want)
	}
}

// Garbage must be refused rather than quietly ignored. Falling back to the
// payer on a typo would hand the reward to the wrong party while the command
// reported success.
func TestRewardOwnerRejectsMalformed(t *testing.T) {
	for _, addr := range []string{
		"not-a-p-address",
		"P-lux1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq", // bad checksum
		"0x55be906ece0552797752e2894d9d7ef952414def",     // an EVM address is not a P-Chain one
		"P-",
	} {
		if _, err := rewardOwner(addr); err == nil {
			t.Errorf("%q was accepted; it is not a P-Chain address", addr)
		}
	}
}
