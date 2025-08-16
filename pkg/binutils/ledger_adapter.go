// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package binutils

import (
	"fmt"

	"github.com/luxfi/ids"
	ledger "github.com/luxfi/ledger-lux-go"
	"github.com/luxfi/crypto/keychain"
	"github.com/luxfi/node/version"
)

// LedgerAdapter wraps ledger.LedgerLux to implement keychain.Ledger interface
type LedgerAdapter struct {
	device *ledger.LedgerLux
}

// NewLedgerAdapter creates a new ledger adapter
func NewLedgerAdapter() (keychain.Ledger, error) {
	device, err := ledger.FindLedgerLuxApp()
	if err != nil {
		return nil, err
	}
	return &LedgerAdapter{device: device}, nil
}

// Version returns the app version
func (l *LedgerAdapter) Version() (*version.Semantic, error) {
	ver, err := l.device.GetVersion()
	if err != nil {
		return nil, err
	}
	return &version.Semantic{
		Major: int(ver.Major),
		Minor: int(ver.Minor),
		Patch: int(ver.Patch),
	}, nil
}

// Address returns a single address at the given index
func (l *LedgerAdapter) Address(displayHRP string, addressIndex uint32) (ids.ShortID, error) {
	path := fmt.Sprintf("44'/9000'/0'/0/%d", addressIndex)
	resp, err := l.device.GetPubKey(path, false, displayHRP, "")
	if err != nil {
		return ids.ShortID{}, err
	}
	if len(resp.Hash) != 20 {
		return ids.ShortID{}, fmt.Errorf("invalid hash length: %d", len(resp.Hash))
	}
	return ids.ToShortID(resp.Hash)
}

// Addresses returns multiple addresses at the given indices
func (l *LedgerAdapter) Addresses(addressIndices []uint32) ([]ids.ShortID, error) {
	addresses := make([]ids.ShortID, len(addressIndices))
	for i, index := range addressIndices {
		addr, err := l.Address("P", index)
		if err != nil {
			return nil, err
		}
		addresses[i] = addr
	}
	return addresses, nil
}

// SignHash signs a hash with the given address indices
func (l *LedgerAdapter) SignHash(hash []byte, addressIndices []uint32) ([][]byte, error) {
	// Build signing paths from address indices
	signingPaths := make([]string, len(addressIndices))
	for i, index := range addressIndices {
		signingPaths[i] = fmt.Sprintf("0/%d", index)
	}
	
	// Sign with all paths at once
	resp, err := l.device.SignHash("44'/9000'/0'", signingPaths, hash)
	if err != nil {
		return nil, err
	}
	
	// Extract signatures for each path
	signatures := make([][]byte, len(addressIndices))
	for i, path := range signingPaths {
		if sigBytes, ok := resp.Signature[path]; ok {
			signatures[i] = sigBytes
		} else {
			return nil, fmt.Errorf("signature not found for path %s", path)
		}
	}
	return signatures, nil
}

// Sign signs transaction bytes with the given address indices
func (l *LedgerAdapter) Sign(unsignedTxBytes []byte, addressIndices []uint32) ([][]byte, error) {
	// Build signing paths from address indices
	signingPaths := make([]string, len(addressIndices))
	for i, index := range addressIndices {
		signingPaths[i] = fmt.Sprintf("0/%d", index)
	}
	
	// Sign with all paths at once (no change paths for now)
	changePaths := []string{}
	resp, err := l.device.Sign("44'/9000'/0'", signingPaths, unsignedTxBytes, changePaths)
	if err != nil {
		return nil, err
	}
	
	// Extract signatures for each path
	signatures := make([][]byte, len(addressIndices))
	for i, path := range signingPaths {
		if sigBytes, ok := resp.Signature[path]; ok {
			signatures[i] = sigBytes
		} else {
			return nil, fmt.Errorf("signature not found for path %s", path)
		}
	}
	return signatures, nil
}

// Disconnect closes the connection to the ledger device
func (l *LedgerAdapter) Disconnect() error {
	return l.device.Close()
}