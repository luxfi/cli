// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"regexp"
	"strings"
)

// RemoveLineCleanChars removes ANSI escape codes and other terminal control characters from a string
// This is useful for cleaning up command output before pattern matching
func RemoveLineCleanChars(s string) string {
	// Remove ANSI escape codes (color codes, cursor movements, etc.)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	s = ansiRegex.ReplaceAllString(s, "")

	// Remove carriage returns
	s = strings.ReplaceAll(s, "\r", "")

	// Remove other common control characters
	controlRegex := regexp.MustCompile(`[\x00-\x08\x0B-\x0C\x0E-\x1F]`)
	s = controlRegex.ReplaceAllString(s, "")

	return s
}
