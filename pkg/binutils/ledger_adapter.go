// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package binutils

import (
	"fmt"

	"github.com/luxfi/ids"
	ledger "github.com/luxfi/ledger-lux-go"
	"github.com/luxfi/node/utils/crypto/keychain"
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

// GetAddresses is an alias for Addresses to satisfy the interface
func (l *LedgerAdapter) GetAddresses(addressIndices []uint32) ([]ids.ShortID, error) {
	return l.Addresses(addressIndices)
}

// SignHash signs a hash with a single address index
func (l *LedgerAdapter) SignHash(hash []byte, addressIndex uint32) ([]byte, error) {
	// Build signing path from address index
	signingPath := fmt.Sprintf("0/%d", addressIndex)
	signingPaths := []string{signingPath}
	
	// Sign with the path
	resp, err := l.device.SignHash("44'/9000'/0'", signingPaths, hash)
	if err != nil {
		return nil, err
	}
	
	// Extract signature for the path
	if sigBytes, ok := resp.Signature[signingPath]; ok {
		return sigBytes, nil
	}
	return nil, fmt.Errorf("signature not found for path %s", signingPath)
}

// Sign signs transaction bytes with a single address index
func (l *LedgerAdapter) Sign(unsignedTxBytes []byte, addressIndex uint32) ([]byte, error) {
	// Build signing path from address index
	signingPath := fmt.Sprintf("0/%d", addressIndex)
	signingPaths := []string{signingPath}
	
	// Sign with the path (no change paths for now)
	changePaths := []string{}
	resp, err := l.device.Sign("44'/9000'/0'", signingPaths, unsignedTxBytes, changePaths)
	if err != nil {
		return nil, err
	}
	
	// Extract signature for the path
	if sigBytes, ok := resp.Signature[signingPath]; ok {
		return sigBytes, nil
	}
	return nil, fmt.Errorf("signature not found for path %s", signingPath)
}

// SignTransaction signs transaction bytes with multiple address indices
func (l *LedgerAdapter) SignTransaction(unsignedTxBytes []byte, addressIndices []uint32) ([][]byte, error) {
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