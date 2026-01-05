// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package configcmd

import (
	"strings"
	"testing"

	"github.com/luxfi/sdk/configspec"
)

func TestLintConfig_ValidConfig(t *testing.T) {
	config := map[string]interface{}{
		"http-host":          "127.0.0.1",
		"http-port":          float64(9630),
		"log-level":          "info",
		"network-timeout-halflife": "2s",
		"api-admin-enabled":  true,
	}

	result := lintConfig(config)
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestLintConfig_UnknownKey(t *testing.T) {
	config := map[string]interface{}{
		"unknown-key": "value",
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0] != `unknown key "unknown-key"` {
		t.Errorf("Unexpected error message: %s", result.Errors[0])
	}
}

func TestLintConfig_UnknownKeyWithSuggestion(t *testing.T) {
	config := map[string]interface{}{
		"inbound-throttler-node-max": "value",
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
	// Check that error contains the unknown key and a suggestion
	if !strings.Contains(result.Errors[0], `unknown key "inbound-throttler-node-max"`) {
		t.Errorf("Expected error to mention unknown key, got %q", result.Errors[0])
	}
	if !strings.Contains(result.Errors[0], "did you mean") {
		t.Errorf("Expected error to include suggestion, got %q", result.Errors[0])
	}
}

func TestLintConfig_DeprecatedKey(t *testing.T) {
	config := map[string]interface{}{
		"snow-sample-size": 20,
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
	// snow-sample-size may be deprecated or unknown, either should suggest consensus-sample-size
	if !strings.Contains(result.Errors[0], "snow-sample-size") {
		t.Errorf("Expected error to mention snow-sample-size, got %q", result.Errors[0])
	}
	if !strings.Contains(result.Errors[0], "consensus-sample-size") {
		t.Errorf("Expected error to suggest consensus-sample-size, got %q", result.Errors[0])
	}
}

func TestLintConfig_InvalidBool(t *testing.T) {
	config := map[string]interface{}{
		"api-admin-enabled": "yes", // should be bool
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestLintConfig_InvalidDuration(t *testing.T) {
	config := map[string]interface{}{
		"network-timeout-halflife": "abc", // invalid duration
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0] != `invalid value for "network-timeout-halflife": "abc" (expected duration like "2s", "500ms", "1h")` {
		t.Errorf("Unexpected error: %s", result.Errors[0])
	}
}

func TestLintConfig_ValidDuration(t *testing.T) {
	cases := []string{"2s", "500ms", "1h", "30m", "2h30m", "1h30m45s"}
	for _, dur := range cases {
		config := map[string]interface{}{
			"network-timeout-halflife": dur,
		}
		result := lintConfig(config)
		if len(result.Errors) != 0 {
			t.Errorf("Duration %q should be valid, got error: %v", dur, result.Errors)
		}
	}
}

func TestLintConfig_InvalidInteger(t *testing.T) {
	config := map[string]interface{}{
		"http-port": "not-a-number",
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestLintConfig_NegativeUint(t *testing.T) {
	config := map[string]interface{}{
		"http-port": float64(-1),
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestLintConfig_FloatWhenIntExpected(t *testing.T) {
	config := map[string]interface{}{
		"http-port": 9630.5, // should be integer
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestLintConfig_InvalidStringSlice(t *testing.T) {
	config := map[string]interface{}{
		"http-allowed-hosts": []interface{}{1, 2, 3}, // should be strings
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestLintConfig_ValidStringSlice(t *testing.T) {
	config := map[string]interface{}{
		"http-allowed-hosts": []interface{}{"localhost", "example.com"},
	}

	result := lintConfig(config)
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestLintConfig_InvalidIntSlice(t *testing.T) {
	config := map[string]interface{}{
		"lp-support": []interface{}{"a", "b"}, // should be ints
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestLintConfig_ValidIntSlice(t *testing.T) {
	config := map[string]interface{}{
		"lp-support": []interface{}{float64(1), float64(2), float64(3)},
	}

	result := lintConfig(config)
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestLintConfig_InvalidStringToString(t *testing.T) {
	config := map[string]interface{}{
		"tracing-headers": map[string]interface{}{
			"key1": 123, // should be string
		},
	}

	result := lintConfig(config)
	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestLintConfig_ValidStringToString(t *testing.T) {
	config := map[string]interface{}{
		"tracing-headers": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	result := lintConfig(config)
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestLintConfig_MultipleErrors(t *testing.T) {
	config := map[string]interface{}{
		"unknown-key":              "value",
		"http-port":                "invalid",
		"network-timeout-halflife": "xyz",
	}

	result := lintConfig(config)
	if len(result.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestSimilarity(t *testing.T) {
	cases := []struct {
		a, b     string
		minScore int
	}{
		{"http-host", "http-host", 100},
		{"http-host", "http", 40}, // partial match
		{"network-timeout-halflife", "network-timeout-halflife", 100},
		{"bootstrap-ip", "bootstrap-ips", 80},
	}

	for _, c := range cases {
		score := similarity(c.a, c.b)
		if score < c.minScore {
			t.Errorf("similarity(%q, %q) = %d, want >= %d", c.a, c.b, score, c.minScore)
		}
	}
}

func TestIsValidKey(t *testing.T) {
	if !IsValidKey("http-host") {
		t.Error("http-host should be valid")
	}
	if IsValidKey("not-a-real-key") {
		t.Error("not-a-real-key should not be valid")
	}
}

func TestGetFlagType(t *testing.T) {
	spec := configspec.MustSpec()
	flag := spec.GetFlag("http-port")
	if flag == nil {
		t.Fatal("http-port should have a spec")
	}
	if flag.Type != configspec.TypeUint {
		t.Errorf("http-port should be TypeUint, got %v", flag.Type)
	}

	flag = spec.GetFlag("not-a-key")
	if flag != nil {
		t.Error("not-a-key should not have a spec")
	}
}
