// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpccmd

import (
	"testing"

	"github.com/luxfi/ids"
	"github.com/luxfi/util/wrappers"
	"github.com/luxfi/utxo"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/stretchr/testify/require"
)

// packAtomicUTXO writes the blob exactly as the C-Chain's atomic codec does:
// version, then the UTXO header, then the secp256k1fx.TransferOutput body.
// Written independently of the decoder so a matching bug in both cannot pass.
func packAtomicUTXO(txID ids.ID, idx uint32, assetID ids.ID, amt, locktime uint64, threshold uint32, addrs []ids.ShortID) []byte {
	p := &wrappers.Packer{MaxSize: maxAtomicUTXOSize}
	p.PackShort(atomicCodecVersion)
	p.PackFixedBytes(txID[:])
	p.PackInt(idx)
	p.PackFixedBytes(assetID[:])
	p.PackInt(typeIDTransferOutput)
	p.PackLong(amt)
	p.PackLong(locktime)
	p.PackInt(threshold)
	p.PackInt(uint32(len(addrs)))
	for _, a := range addrs {
		p.PackFixedBytes(a[:])
	}
	return p.Bytes
}

func TestDecodeAtomicUTXO(t *testing.T) {
	txID := ids.ID{1, 2, 3}
	assetID := ids.ID{9, 8, 7}
	addrs := []ids.ShortID{{0xaa}, {0xbb}}
	blob := packAtomicUTXO(txID, 7, assetID, 1_000_000, 42, 2, addrs)

	var u utxo.UTXO
	require.NoError(t, decodeAtomicUTXO(blob, &u))

	require.Equal(t, txID, u.UTXOID.TxID)
	require.Equal(t, uint32(7), u.UTXOID.OutputIndex)
	require.Equal(t, assetID, u.Asset.ID)

	out, ok := u.Out.(*secp256k1fx.TransferOutput)
	require.True(t, ok, "Out should decode to *secp256k1fx.TransferOutput")
	require.Equal(t, uint64(1_000_000), out.Amt)
	require.Equal(t, uint64(42), out.OutputOwners.Locktime)
	require.Equal(t, uint32(2), out.OutputOwners.Threshold)
	require.Equal(t, addrs, out.OutputOwners.Addrs)
}

func TestDecodeAtomicUTXOZeroAddrs(t *testing.T) {
	blob := packAtomicUTXO(ids.Empty, 0, ids.Empty, 0, 0, 0, nil)

	var u utxo.UTXO
	require.NoError(t, decodeAtomicUTXO(blob, &u))
	require.Empty(t, u.Out.(*secp256k1fx.TransferOutput).OutputOwners.Addrs)
}

func TestDecodeAtomicUTXORejects(t *testing.T) {
	good := packAtomicUTXO(ids.ID{1}, 0, ids.ID{2}, 5, 0, 1, []ids.ShortID{{0xcc}})

	t.Run("truncated", func(t *testing.T) {
		for _, n := range []int{0, 1, 2, 33, len(good) - 1} {
			var u utxo.UTXO
			require.Error(t, decodeAtomicUTXO(good[:n], &u), "should reject %d-byte buffer", n)
		}
	})

	t.Run("trailing bytes", func(t *testing.T) {
		var u utxo.UTXO
		require.ErrorIs(t, decodeAtomicUTXO(append(good, 0x00), &u), errAtomicUTXOTrailing)
	})

	t.Run("wrong version", func(t *testing.T) {
		bad := make([]byte, len(good))
		copy(bad, good)
		bad[1] = 0x01 // version u16 -> 1
		var u utxo.UTXO
		require.ErrorIs(t, decodeAtomicUTXO(bad, &u), errAtomicUTXOVersion)
	})

	t.Run("unsupported output type", func(t *testing.T) {
		bad := make([]byte, len(good))
		copy(bad, good)
		bad[2+32+4+32+3] = 0x06 // typeID 7 -> 6
		var u utxo.UTXO
		require.Error(t, decodeAtomicUTXO(bad, &u))
	})

	// A header claiming more addresses than the buffer holds must fail rather
	// than allocate against the claimed count.
	t.Run("address count exceeds buffer", func(t *testing.T) {
		bad := make([]byte, len(good))
		copy(bad, good)
		nAddrsOff := 2 + 32 + 4 + 32 + 4 + 8 + 8 + 4
		bad[nAddrsOff], bad[nAddrsOff+1] = 0xff, 0xff
		var u utxo.UTXO
		require.ErrorIs(t, decodeAtomicUTXO(bad, &u), errAtomicUTXOTruncate)
	})
}
