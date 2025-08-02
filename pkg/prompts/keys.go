// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package prompts

import (
	"path/filepath"
	"strings"
)

func GetKeyOrLedger(prompter Prompter, prompt string, keysFilePaths []string, goalMsg string, useLedger bool, ledgerAddresses []string) (keyName string, useLedgerIndex int, err error) {
	// Simple implementation for now
	if useLedger && len(ledgerAddresses) > 0 {
		// Select from ledger addresses
		address, err := prompter.CaptureList(prompt, ledgerAddresses)
		if err != nil {
			return "", 0, err
		}
		for i, addr := range ledgerAddresses {
			if addr == address {
				return "", i, nil
			}
		}
	}
	
	// Select from key files
	keyNames := make([]string, len(keysFilePaths))
	for i, path := range keysFilePaths {
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		keyNames[i] = strings.TrimSuffix(base, ext)
	}
	
	if len(keyNames) == 0 {
		return "", 0, nil
	}
	
	keyName, err = prompter.CaptureList(prompt, keyNames)
	return keyName, -1, err
}