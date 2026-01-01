// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package prompts

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// Environment variable names for non-interactive mode.
const (
	// EnvNonInteractive forces non-interactive mode.
	// Set to "1", "true", "yes", or "on" to enable.
	EnvNonInteractive = "LUX_NON_INTERACTIVE"

	// EnvCI is a common CI environment variable.
	// When truthy, implies non-interactive.
	EnvCI = "CI"
)

// isTruthyEnv checks if an environment variable is set to a truthy value.
// Accepts: 1, true, t, yes, y, on (case-insensitive)
func isTruthyEnv(key string) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// stdinIsTTY returns true if stdin is a terminal (TTY).
// Uses golang.org/x/term for robust cross-platform detection.
func stdinIsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// IsInteractive returns true if prompting is allowed.
//
// Interactive mode is enabled when ALL of:
//   - stdin is a TTY (not piped/redirected)
//   - LUX_NON_INTERACTIVE is not truthy
//   - CI is not truthy
//
// This follows UNIX conventions:
//   - If stdin is not a TTY → never prompt (scripts, pipes)
//   - Explicit env override always wins
func IsInteractive() bool {
	// Explicit user override via env
	if isTruthyEnv(EnvNonInteractive) {
		return false
	}

	// CI convention (GitHub Actions, GitLab CI, etc.)
	if isTruthyEnv(EnvCI) {
		return false
	}

	// Piped/redirected stdin => never prompt
	if !stdinIsTTY() {
		return false
	}

	return true
}

// IsNonInteractive is the inverse of IsInteractive.
// Deprecated: Use !IsInteractive() or the Validator pattern instead.
func IsNonInteractive(flag bool) bool {
	if flag {
		return true
	}
	return !IsInteractive()
}

// NewPrompterForMode returns the appropriate prompter based on mode.
//
// If non-interactive, returns NonInteractivePrompter that fails fast.
// If interactive (TTY), returns the standard realPrompter that can prompt.
func NewPrompterForMode(nonInteractiveFlag bool) Prompter {
	if IsNonInteractive(nonInteractiveFlag) {
		return NewNonInteractivePrompter()
	}
	return NewPrompter()
}

// MustInteractive panics if in non-interactive mode.
// Use for operations that absolutely require user interaction
// and cannot be made non-interactive (e.g., ledger signing).
func MustInteractive(operation string) {
	if !IsInteractive() {
		panic("operation requires interactive mode: " + operation)
	}
}
