// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package relayer provides Warp message relayer functionality
package relayer

import (
	"os"
	"path/filepath"
)

// Cleanup cleans up relayer state files.
func Cleanup(runPath, logPath, storageDir string) error {
	// Clean up run path
	if runPath != "" {
		pidFile := filepath.Join(runPath, "relayer.pid")
		_ = os.Remove(pidFile)
	}

	// Clean up log path
	if logPath != "" {
		_ = os.RemoveAll(logPath)
	}

	// Clean up storage directory
	if storageDir != "" {
		_ = os.RemoveAll(storageDir)
	}

	return nil
}
