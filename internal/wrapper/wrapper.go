// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package wrapper implements argv[0]-based subcommand routing.
// When the lux binary is invoked via a symlink (e.g. lux-zk, zk),
// the corresponding subcommand is automatically prepended to os.Args.
package wrapper

import (
	"os"
	"path/filepath"
	"strings"
)

// Domains lists subcommands that can be invoked via symlink.
var Domains = map[string]bool{
	"ai":       true,
	"tui":      true,
	"zk":       true,
	"fhe":      true,
	"mpc":      true,
	"kms":      true,
	"rt":       true,
	"ringtail": true,
	"explore":  true,
}

// platformSuffixes are stripped from the executable name before matching.
var platformSuffixes = []string{
	"-linux-amd64",
	"-linux-arm64",
	"-darwin-amd64",
	"-darwin-arm64",
}

// RewriteArgs detects when the binary is invoked via a symlink
// (e.g. "lux-zk", "zk") and prepends the corresponding subcommand to os.Args.
func RewriteArgs() {
	exe := filepath.Base(os.Args[0])

	// Strip platform suffixes from development builds
	for _, suffix := range platformSuffixes {
		exe = strings.TrimSuffix(exe, suffix)
	}

	// If invoked as "lux", nothing to rewrite
	if exe == "lux" {
		return
	}

	// Try stripping "lux-" prefix: "lux-zk" -> "zk"
	domain := strings.TrimPrefix(exe, "lux-")

	if Domains[domain] {
		os.Args = append([]string{os.Args[0], domain}, os.Args[1:]...)
	}
}
