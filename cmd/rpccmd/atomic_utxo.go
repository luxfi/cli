// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpccmd

import (
	"errors"
	"fmt"

	"github.com/luxfi/ids"
	"github.com/luxfi/utils/wrappers"
	"github.com/luxfi/utxo"
	"github.com/luxfi/utxo/secp256k1fx"
)

// The C-Chain returns atomic UTXOs in the atomic wire format:
//
//	[u16 version][32 txID][u32 outputIndex][32 assetID][u32 outTypeID][out body]
//
// and for the one output type that can appear on an atomic UTXO,
// secp256k1fx.TransferOutput:
//
//	[u64 amount][u64 locktime][u32 threshold][u32 nAddrs][20 addr]*
//
// Everything is big-endian, which is what wrappers.Packer does.
//
// This is a client: it only ever needs to READ this shape. Pulling in the
// C-Chain VM to get its codec would drag the whole VM — and its own
// dependency graph — behind one Unmarshal call, which is what previously
// pinned this module to a coreth that cannot build against current warp.
const (
	atomicCodecVersion   uint16 = 0
	typeIDTransferOutput uint32 = 7

	// A UTXO carrying a single-address TransferOutput is the smallest legal
	// encoding; anything shorter cannot be one.
	maxAtomicUTXOSize = 1 << 20
)

var (
	errAtomicUTXOVersion  = errors.New("atomic utxo: unknown wire version")
	errAtomicUTXOTruncate = errors.New("atomic utxo: buffer ended mid-record")
	errAtomicUTXOTrailing = errors.New("atomic utxo: trailing bytes after record")
)

// decodeAtomicUTXO parses one atomic UTXO blob into u.
func decodeAtomicUTXO(b []byte, u *utxo.UTXO) error {
	p := &wrappers.Packer{Bytes: b, MaxSize: maxAtomicUTXOSize}

	if version := p.UnpackShort(); p.Errored() || version != atomicCodecVersion {
		if p.Errored() {
			return errAtomicUTXOTruncate
		}
		return fmt.Errorf("%w: %d", errAtomicUTXOVersion, version)
	}

	copy(u.UTXOID.TxID[:], p.UnpackFixedBytes(ids.IDLen))
	u.UTXOID.OutputIndex = p.UnpackInt()
	copy(u.Asset.ID[:], p.UnpackFixedBytes(ids.IDLen))
	typeID := p.UnpackInt()
	if p.Errored() {
		return errAtomicUTXOTruncate
	}
	if typeID != typeIDTransferOutput {
		return fmt.Errorf("atomic utxo: unsupported output typeID %d", typeID)
	}

	out := &secp256k1fx.TransferOutput{}
	out.Amt = p.UnpackLong()
	out.OutputOwners.Locktime = p.UnpackLong()
	out.OutputOwners.Threshold = p.UnpackInt()
	nAddrs := p.UnpackInt()
	if p.Errored() {
		return errAtomicUTXOTruncate
	}
	// nAddrs is attacker-controlled, so grow the slice as addresses are read
	// rather than preallocating what the header claims.
	out.OutputOwners.Addrs = make([]ids.ShortID, 0, min(nAddrs, 16))
	for i := uint32(0); i < nAddrs; i++ {
		var addr ids.ShortID
		copy(addr[:], p.UnpackFixedBytes(ids.ShortIDLen))
		if p.Errored() {
			return errAtomicUTXOTruncate
		}
		out.OutputOwners.Addrs = append(out.OutputOwners.Addrs, addr)
	}
	u.Out = out

	if p.Offset != len(b) {
		return errAtomicUTXOTrailing
	}
	return nil
}
