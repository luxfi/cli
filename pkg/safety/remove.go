// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package safety provides safe deletion operations that protect user configuration.
package safety

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Policy defines which paths are allowed or denied for deletion.
type Policy struct {
	BaseDir       string   // The Lux base directory (e.g., ~/.lux)
	AllowPrefixes []string // absolute paths allowed to delete under
	DenyPrefixes  []string // absolute paths never deletable
}

// DefaultPolicy returns the standard safety policy for the CLI.
// It allows deletion of ephemeral runtime state but protects user configuration.
// baseDir is the Lux base directory (e.g., ~/.lux)
func DefaultPolicy(baseDir string) Policy {
	homeDir := filepath.Dir(baseDir) // ~/.lux -> ~
	return Policy{
		BaseDir: baseDir,
		AllowPrefixes: []string{
			filepath.Join(baseDir, "runs"),      // Runtime state
			filepath.Join(baseDir, "snapshots"), // User-managed snapshots (optional)
			filepath.Join(baseDir, "logs"),      // Log files
			filepath.Join(baseDir, "db"),        // Database (ephemeral for local)
			filepath.Join(baseDir, "dev"),       // Dev mode state
			filepath.Join(baseDir, "devnet"),    // Devnet state
		},
		DenyPrefixes: []string{
			filepath.Join(baseDir, "chains"),   // Chain configurations - NEVER delete automatically
			filepath.Join(baseDir, "plugins"),  // VM plugins - NEVER delete
			filepath.Join(baseDir, "keys"),     // User keys - NEVER delete
			filepath.Join(baseDir, "cli.json"), // CLI config - NEVER delete
			filepath.Join(baseDir, "sdk.json"), // SDK config - NEVER delete
			filepath.Join(homeDir, ".cli.json"), // Legacy config - NEVER delete
		},
	}
}

// RemoveAll safely removes a directory or file, respecting the policy.
// It returns an error if the target is protected or not in an allowed path.
func RemoveAll(policy Policy, target string) error {
	abs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// First check deny list - these paths are NEVER deletable
	for _, d := range policy.DenyPrefixes {
		if isUnderOrEqual(abs, d) {
			return fmt.Errorf("refusing to delete protected path: %s (protected by policy)", abs)
		}
	}

	// Then check allow list - path must be under an allowed prefix
	allowed := false
	for _, a := range policy.AllowPrefixes {
		if isUnderOrEqual(abs, a) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("refusing to delete non-ephemeral path: %s (not in allowed list)", abs)
	}

	return os.RemoveAll(abs)
}

// isUnderOrEqual returns true if path is equal to or under prefix.
func isUnderOrEqual(path, prefix string) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+string(filepath.Separator))
}

// MustRemoveAll is like RemoveAll but panics on policy violation.
// Use only in tests or when you're absolutely sure the path is safe.
func MustRemoveAll(policy Policy, target string) {
	if err := RemoveAll(policy, target); err != nil {
		panic(err)
	}
}

// RemoveAllUnsafe removes a path without policy checks.
// This should ONLY be used in very specific cases with explicit user confirmation.
// The caller is responsible for ensuring safety.
func RemoveAllUnsafe(target string) error {
	return os.RemoveAll(target)
}

// IsProtected checks if a path is protected by the given policy.
func IsProtected(policy Policy, target string) bool {
	abs, err := filepath.Abs(target)
	if err != nil {
		return true // If we can't resolve, assume protected
	}

	for _, d := range policy.DenyPrefixes {
		if isUnderOrEqual(abs, d) {
			return true
		}
	}
	return false
}

// IsAllowed checks if a path is in the allowed deletion list.
func IsAllowed(policy Policy, target string) bool {
	abs, err := filepath.Abs(target)
	if err != nil {
		return false
	}

	for _, a := range policy.AllowPrefixes {
		if isUnderOrEqual(abs, a) {
			return true
		}
	}
	return false
}

// RemoveChainConfig removes a specific chain configuration directory.
// This is a special-case function for user-confirmed chain deletion.
// It requires that the target is a direct subdirectory of the chains directory,
// NOT the chains directory itself. The caller must ensure user confirmation.
// baseDir is the Lux base directory (e.g., ~/.lux)
func RemoveChainConfig(baseDir, chainName string) error {
	if chainName == "" || chainName == "." || chainName == ".." {
		return fmt.Errorf("invalid chain name: %s", chainName)
	}
	if filepath.Base(chainName) != chainName {
		return fmt.Errorf("chain name cannot contain path separators: %s", chainName)
	}

	chainsDir := filepath.Join(baseDir, "chains")
	targetDir := filepath.Join(chainsDir, chainName)

	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve chain directory: %w", err)
	}

	absChains, err := filepath.Abs(chainsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve chains directory: %w", err)
	}

	// Safety: must be a direct subdirectory
	if absTarget == absChains {
		return fmt.Errorf("SAFETY: refusing to delete the entire chains directory")
	}
	if filepath.Dir(absTarget) != absChains {
		return fmt.Errorf("SAFETY: chain config must be directly inside chains directory")
	}

	return os.RemoveAll(absTarget)
}
