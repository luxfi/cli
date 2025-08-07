// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package prompts

import (
	"fmt"
	"strings"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
)

// EVMFormat represents the EVM address format
const EVMFormat = "evm"

// PromptAddress prompts the user for an address
func PromptAddress(prompter Prompter, prompt string) (string, error) {
	addr, err := prompter.CaptureString(prompt)
	if err != nil {
		return "", err
	}
	// Basic validation
	if !strings.HasPrefix(addr, "0x") {
		return "", fmt.Errorf("invalid address format: must start with 0x")
	}
	if len(addr) != 42 { // 0x + 40 hex chars
		return "", fmt.Errorf("invalid address length: expected 42 characters")
	}
	return addr, nil
}

// ValidateAddress validates an Ethereum address
func ValidateAddress(addr string) error {
	if !strings.HasPrefix(addr, "0x") {
		return fmt.Errorf("invalid address format: must start with 0x")
	}
	if len(addr) != 42 { // 0x + 40 hex chars
		return fmt.Errorf("invalid address length: expected 42 characters")
	}
	// Try to parse as common.Address
	if !common.IsHexAddress(addr) {
		return fmt.Errorf("invalid hex address")
	}
	return nil
}

// PromptPrivateKey prompts the user for a private key
func PromptPrivateKey(prompter Prompter, prompt string) (string, error) {
	key, err := prompter.CaptureString(prompt)
	if err != nil {
		return "", err
	}
	// Remove 0x prefix if present
	key = strings.TrimPrefix(key, "0x")
	// Basic validation - should be 64 hex chars
	if len(key) != 64 {
		return "", fmt.Errorf("invalid private key length: expected 64 hex characters")
	}
	// Validate hex
	for _, c := range key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("invalid private key: contains non-hex characters")
		}
	}
	return key, nil
}

// ConvertToAddress converts a string to a crypto.Address
func ConvertToAddress(addr string) (crypto.Address, error) {
	if err := ValidateAddress(addr); err != nil {
		return crypto.Address{}, err
	}
	return crypto.HexToAddress(addr), nil
}