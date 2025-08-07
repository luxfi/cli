// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package signatureaggregator

import "os"

// SignatureAggregatorCleanup cleans up signature aggregator files
func SignatureAggregatorCleanup(runPath string, storagePath string) error {
	// Clean up run file
	if runPath != "" {
		_ = os.Remove(runPath)
	}
	// Clean up storage directory
	if storagePath != "" {
		_ = os.RemoveAll(storagePath)
	}
	return nil
}