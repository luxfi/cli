// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build !unix

package key

// mlock is a no-op on non-Unix platforms.
func mlock(b []byte) error {
	return nil
}

// munlock is a no-op on non-Unix platforms.
func munlock(b []byte) error {
	return nil
}

// mlockSupported returns false on non-Unix platforms.
func mlockSupported() bool {
	return false
}
