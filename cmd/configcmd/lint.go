// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package configcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/luxfi/sdk/configspec"
	"github.com/spf13/cobra"
)

// LintResult contains the result of linting a configuration file.
type LintResult struct {
	Errors   []string
	Warnings []string
}

func newLintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint <config-file.json>",
		Short: "Validate luxd configuration file",
		Long: `Validate a luxd configuration file for errors.

Reports:
  - Unknown configuration keys (with typo suggestions)
  - Invalid value types (e.g., "abc" for a duration)
  - Deprecated keys (with replacement hints)

Uses the authoritative flag spec from github.com/luxfi/sdk/configspec,
which is generated from the node's source of truth.

Example:
  lux config lint myconfig.json`,
		Args: cobra.ExactArgs(1),
		RunE: runLint,
	}
}

func runLint(_ *cobra.Command, args []string) error {
	configPath := args[0]

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := lintConfig(config)

	// Print results
	for _, e := range result.Errors {
		fmt.Printf("ERROR: %s\n", e)
	}
	for _, w := range result.Warnings {
		fmt.Printf("WARN: %s\n", w)
	}

	// Summary
	fmt.Printf("%d errors, %d warnings\n", len(result.Errors), len(result.Warnings))

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
	return nil
}

func lintConfig(config map[string]interface{}) *LintResult {
	result := &LintResult{}
	spec := configspec.MustSpec()

	// Sort keys for deterministic output
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := config[key]

		// Look up flag in spec
		flagSpec := spec.GetFlag(key)
		if flagSpec == nil {
			suggestion := suggestKey(key, spec)
			if suggestion != "" {
				result.Errors = append(result.Errors,
					fmt.Sprintf("unknown key %q (did you mean %q?)", key, suggestion))
			} else {
				result.Errors = append(result.Errors,
					fmt.Sprintf("unknown key %q", key))
			}
			continue
		}

		// Check for deprecated keys
		if flagSpec.Deprecated {
			msg := fmt.Sprintf("deprecated key %q", key)
			if flagSpec.DeprecatedMessage != "" {
				msg += fmt.Sprintf(" (%s)", flagSpec.DeprecatedMessage)
			}
			if flagSpec.ReplacedBy != "" {
				msg += fmt.Sprintf(" - use %q instead", flagSpec.ReplacedBy)
			}
			result.Warnings = append(result.Warnings, msg)
		}

		// Validate value type
		if err := validateValue(key, value, flagSpec.Type); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result
}

func suggestKey(unknown string, spec *configspec.ConfigSpec) string {
	// Find closest match by similarity
	bestMatch := ""
	bestScore := 0

	for _, flag := range spec.Flags {
		score := similarity(unknown, flag.Key)
		if score > bestScore && score >= 50 { // Require >=50% similarity
			bestScore = score
			bestMatch = flag.Key
		}
	}

	return bestMatch
}

// similarity returns a percentage (0-100) of how similar two strings are.
func similarity(a, b string) int {
	if a == b {
		return 100
	}

	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)

	// Simple substring matching
	if strings.Contains(aLower, bLower) || strings.Contains(bLower, aLower) {
		shorter := len(a)
		if len(b) < shorter {
			shorter = len(b)
		}
		longer := len(a)
		if len(b) > longer {
			longer = len(b)
		}
		return (shorter * 100) / longer
	}

	// Token-based matching
	aTokens := strings.Split(strings.ReplaceAll(aLower, "_", "-"), "-")
	bTokens := strings.Split(strings.ReplaceAll(bLower, "_", "-"), "-")

	matches := 0
	charMatches := 0
	for _, at := range aTokens {
		for _, bt := range bTokens {
			if at == bt {
				matches++
				charMatches += len(at)
				break
			}
			// Count character-level similarity for partial matches
			if len(at) >= 3 && len(bt) >= 3 {
				commonPrefix := 0
				for i := 0; i < len(at) && i < len(bt); i++ {
					if at[i] == bt[i] {
						commonPrefix++
					} else {
						break
					}
				}
				if commonPrefix >= 3 {
					charMatches += commonPrefix
				}
			}
		}
	}

	totalTokens := len(aTokens)
	if len(bTokens) > totalTokens {
		totalTokens = len(bTokens)
	}

	if totalTokens == 0 {
		return 0
	}

	// Combine token matching with character-level similarity
	tokenScore := (matches * 100) / totalTokens
	// Add bonus for character-level matches (up to 30 points)
	charBonus := 0
	if charMatches > 0 {
		maxLen := len(a)
		if len(b) > maxLen {
			maxLen = len(b)
		}
		charBonus = (charMatches * 30) / maxLen
	}

	return tokenScore + charBonus
}

func validateValue(key string, value interface{}, expectedType configspec.FlagType) error {
	switch expectedType {
	case configspec.TypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("invalid value for %q: %v (expected boolean true/false)", key, value)
		}

	case configspec.TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("invalid value for %q: %v (expected string)", key, value)
		}

	case configspec.TypeInt, configspec.TypeUint, configspec.TypeUint64:
		switch v := value.(type) {
		case float64:
			if v != float64(int64(v)) {
				return fmt.Errorf("invalid value for %q: %v (expected integer, got float)", key, value)
			}
			if expectedType == configspec.TypeUint || expectedType == configspec.TypeUint64 {
				if v < 0 {
					return fmt.Errorf("invalid value for %q: %v (expected non-negative integer)", key, value)
				}
			}
		case int, int64, uint, uint64:
			// OK
		default:
			return fmt.Errorf("invalid value for %q: %v (expected integer)", key, value)
		}

	case configspec.TypeFloat64:
		switch value.(type) {
		case float64, int, int64, uint, uint64:
			// OK
		default:
			return fmt.Errorf("invalid value for %q: %v (expected number)", key, value)
		}

	case configspec.TypeDuration:
		switch v := value.(type) {
		case string:
			if _, err := time.ParseDuration(v); err != nil {
				return fmt.Errorf("invalid value for %q: %q (expected duration like \"2s\", \"500ms\", \"1h\")", key, v)
			}
		case float64:
			// Numeric durations interpreted as nanoseconds - this is valid
		default:
			return fmt.Errorf("invalid value for %q: %v (expected duration string like \"2s\")", key, value)
		}

	case configspec.TypeStringSlice:
		switch v := value.(type) {
		case []interface{}:
			for i, item := range v {
				if _, ok := item.(string); !ok {
					return fmt.Errorf("invalid value for %q[%d]: %v (expected string)", key, i, item)
				}
			}
		case string:
			// Single string is OK, will be converted to slice
		default:
			return fmt.Errorf("invalid value for %q: %v (expected string array)", key, value)
		}

	case configspec.TypeIntSlice:
		switch v := value.(type) {
		case []interface{}:
			for i, item := range v {
				if num, ok := item.(float64); !ok || num != float64(int(num)) {
					return fmt.Errorf("invalid value for %q[%d]: %v (expected integer)", key, i, item)
				}
			}
		default:
			return fmt.Errorf("invalid value for %q: %v (expected integer array)", key, value)
		}

	case configspec.TypeStringToString:
		switch v := value.(type) {
		case map[string]interface{}:
			for k, val := range v {
				if _, ok := val.(string); !ok {
					return fmt.Errorf("invalid value for %q[%q]: %v (expected string)", key, k, val)
				}
			}
		default:
			return fmt.Errorf("invalid value for %q: %v (expected object with string values)", key, value)
		}
	}

	return nil
}

// ValidateConfigFile is exported for programmatic use.
func ValidateConfigFile(path string) (*LintResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return lintConfig(config), nil
}

// ValidateConfigJSON validates a JSON config string.
func ValidateConfigJSON(jsonStr string) (*LintResult, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return lintConfig(config), nil
}

// IsValidKey returns true if the key is a valid luxd configuration key.
func IsValidKey(key string) bool {
	return configspec.KnownKey(key)
}

// FormatDuration returns a string suitable for duration config values.
func FormatDuration(d time.Duration) string {
	return d.String()
}

// ParseDuration parses a duration string for config values.
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// ParseInt parses an integer string for config values.
func ParseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ParseUint parses an unsigned integer string for config values.
func ParseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// ParseBool parses a boolean string for config values.
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
