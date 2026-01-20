// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build unix

package key

import (
	"golang.org/x/sys/unix"
)

// mlock locks memory to prevent swapping to disk.
// This is a security measure for sensitive data like encryption keys.
func mlock(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return unix.Mlock(b)
}

// munlock unlocks previously locked memory.
func munlock(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return unix.Munlock(b)
}

// mlockSupported returns true if mlock is available on this platform.
func mlockSupported() bool {
	return true
}
