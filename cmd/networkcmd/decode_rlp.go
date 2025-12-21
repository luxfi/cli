// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build ignore

package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run decode_rlp.go <file.rlp>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// RLP stream for reading blocks
	stream := rlp.NewStream(f, 0)

	// Read first block
	var block types.Block
	if err := stream.Decode(&block); err != nil {
		if err == io.EOF {
			fmt.Println("Empty RLP file")
			os.Exit(1)
		}
		fmt.Printf("Error decoding block: %v\n", err)
		os.Exit(1)
	}

	header := block.Header()
	fmt.Printf("=== Genesis Block from RLP ===\n")
	fmt.Printf("Hash:         %s\n", block.Hash().Hex())
	fmt.Printf("Number:       %d\n", header.Number.Uint64())
	fmt.Printf("ParentHash:   %s\n", header.ParentHash.Hex())
	fmt.Printf("StateRoot:    %s\n", header.Root.Hex())
	fmt.Printf("TxHash:       %s\n", header.TxHash.Hex())
	fmt.Printf("ReceiptHash:  %s\n", header.ReceiptHash.Hex())
	fmt.Printf("Coinbase:     %s\n", header.Coinbase.Hex())
	fmt.Printf("Difficulty:   %s\n", header.Difficulty.String())
	fmt.Printf("GasLimit:     %d (0x%x)\n", header.GasLimit, header.GasLimit)
	fmt.Printf("GasUsed:      %d\n", header.GasUsed)
	fmt.Printf("Timestamp:    %d (0x%x)\n", header.Time, header.Time)
	fmt.Printf("ExtraData:    0x%s\n", hex.EncodeToString(header.Extra))
	fmt.Printf("MixHash:      %s\n", header.MixDigest.Hex())
	fmt.Printf("Nonce:        %d\n", header.Nonce.Uint64())
	fmt.Printf("BaseFee:      %s\n", header.BaseFee.String())
	fmt.Printf("UncleHash:    %s\n", header.UncleHash.Hex())
	fmt.Printf("Bloom:        0x%s\n", hex.EncodeToString(header.Bloom[:]))

	// Check for post-merge fields
	if header.WithdrawalsHash != nil {
		fmt.Printf("WithdrawalsHash: %s\n", header.WithdrawalsHash.Hex())
	}
	if header.BlobGasUsed != nil {
		fmt.Printf("BlobGasUsed: %d\n", *header.BlobGasUsed)
	}
	if header.ExcessBlobGas != nil {
		fmt.Printf("ExcessBlobGas: %d\n", *header.ExcessBlobGas)
	}
	if header.ParentBeaconRoot != nil {
		fmt.Printf("ParentBeaconRoot: %s\n", header.ParentBeaconRoot.Hex())
	}

	fmt.Printf("\n=== Hash Comparisons ===\n")
	fmt.Printf("Standard Hash():  %s\n", header.Hash().Hex())
	fmt.Printf("Hash16() (16-field genesis format): %s\n", header.Hash16().Hex())

	fmt.Printf("\n=== Raw Header RLP ===\n")
	headerRLP, _ := rlp.EncodeToBytes(header)
	fmt.Printf("Header RLP (%d bytes): 0x%s\n", len(headerRLP), hex.EncodeToString(headerRLP))
}
