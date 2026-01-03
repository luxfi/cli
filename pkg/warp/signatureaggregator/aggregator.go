// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package signatureaggregator provides Warp signature aggregation functionality
package signatureaggregator

import (
	"os"
	"path/filepath"
)

// Cleanup cleans up signature aggregator state files.
func Cleanup(runPath, storageDir string) error {
	// Clean up run path
	if runPath != "" {
		pidFile := filepath.Join(runPath, "aggregator.pid")
		_ = os.Remove(pidFile)
	}

	// Clean up storage directory
	if storageDir != "" {
		_ = os.RemoveAll(storageDir)
	}

	return nil
}
