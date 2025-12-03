// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package doctorcmd

import (
	"io"
	"os"
	"testing"
)

// newTestDoctor creates a doctor instance with output redirected to discard
func newTestDoctor(fixMode bool) *Doctor {
	d := NewDoctor(nil, fixMode)
	d.output = io.Discard
	return d
}

func TestCheckStatus(t *testing.T) {
	tests := []struct {
		name   string
		status CheckStatus
		want   int
	}{
		{"StatusOK", StatusOK, 0},
		{"StatusWarn", StatusWarn, 1},
		{"StatusError", StatusError, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.status) != tt.want {
				t.Errorf("CheckStatus %s = %d, want %d", tt.name, int(tt.status), tt.want)
			}
		})
	}
}

func TestNewDoctor(t *testing.T) {
	d := newTestDoctor(false)

	if d == nil {
		t.Fatal("NewDoctor returned nil")
	}

	if d.fixMode != false {
		t.Errorf("fixMode = %v, want false", d.fixMode)
	}

	if d.results == nil {
		t.Error("results slice is nil")
	}

	if len(d.results) != 0 {
		t.Errorf("results length = %d, want 0", len(d.results))
	}

	// Test with fix mode enabled
	d2 := newTestDoctor(true)
	if d2.fixMode != true {
		t.Errorf("fixMode = %v, want true", d2.fixMode)
	}
}

func TestCheckGoVersion(t *testing.T) {
	d := newTestDoctor(false)
	d.checkGoVersion()

	if len(d.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(d.results))
	}

	result := d.results[0]
	if result.Name != "Go Version" {
		t.Errorf("result.Name = %q, want %q", result.Name, "Go Version")
	}

	// Go should be available in test environment
	if result.Status == StatusError {
		t.Logf("Go check returned error (may be expected if Go not in PATH): %s", result.Message)
	}
}

func TestCheckDockerAvailability(t *testing.T) {
	d := newTestDoctor(false)
	d.checkDockerAvailability()

	if len(d.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(d.results))
	}

	result := d.results[0]
	if result.Name != "Docker" {
		t.Errorf("result.Name = %q, want %q", result.Name, "Docker")
	}

	// Docker check should complete without panic
	t.Logf("Docker status: %d, message: %s", result.Status, result.Message)
}

func TestCheckLuxNodeBinary(t *testing.T) {
	d := newTestDoctor(false)
	d.checkLuxNodeBinary()

	if len(d.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(d.results))
	}

	result := d.results[0]
	if result.Name != "Lux Node" {
		t.Errorf("result.Name = %q, want %q", result.Name, "Lux Node")
	}

	// luxd may or may not be installed
	t.Logf("Lux Node status: %d, message: %s", result.Status, result.Message)
}

func TestCheckDiskSpace(t *testing.T) {
	d := newTestDoctor(false)
	d.checkDiskSpace()

	if len(d.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(d.results))
	}

	result := d.results[0]
	if result.Name != "Disk Space" {
		t.Errorf("result.Name = %q, want %q", result.Name, "Disk Space")
	}

	// Disk space check should succeed
	if result.Status == StatusError {
		t.Errorf("disk space check failed unexpectedly: %s", result.Message)
	}
}

func TestCheckCLIDirectories(t *testing.T) {
	d := newTestDoctor(false)
	d.checkCLIDirectories()

	if len(d.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(d.results))
	}

	result := d.results[0]
	if result.Name != "CLI Directories" {
		t.Errorf("result.Name = %q, want %q", result.Name, "CLI Directories")
	}

	// CLI directories may or may not exist
	t.Logf("CLI Directories status: %d, message: %s", result.Status, result.Message)
}

func TestCheckNetworkConnectivity(t *testing.T) {
	// Skip network tests by default to avoid flaky CI
	if os.Getenv("RUN_NETWORK_TESTS") == "" {
		t.Skip("Skipping network tests (set RUN_NETWORK_TESTS=1 to run)")
	}

	d := newTestDoctor(false)
	d.checkNetworkConnectivity()

	// Should have 2 results (mainnet and testnet)
	if len(d.results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(d.results))
	}

	for _, result := range d.results {
		t.Logf("%s status: %d, message: %s", result.Name, result.Status, result.Message)
	}
}

func TestPrintResults(t *testing.T) {
	d := newTestDoctor(false)

	// Add some test results
	d.results = []CheckResult{
		{Name: "Test OK", Status: StatusOK, Message: "All good"},
		{Name: "Test Warn", Status: StatusWarn, Message: "Warning", FixSuggestion: "Fix it"},
		{Name: "Test Error", Status: StatusError, Message: "Error", FixSuggestion: "Fix this too"},
	}

	// Should not panic
	d.printResults()
}

func TestAttemptFixes(t *testing.T) {
	d := newTestDoctor(true)

	fixCalled := false
	d.results = []CheckResult{
		{
			Name:       "Auto-fixable",
			Status:     StatusWarn,
			CanAutoFix: true,
			AutoFix: func() error {
				fixCalled = true
				return nil
			},
		},
		{
			Name:       "Not auto-fixable",
			Status:     StatusWarn,
			CanAutoFix: false,
		},
		{
			Name:   "Already OK",
			Status: StatusOK,
		},
	}

	err := d.attemptFixes()
	if err != nil {
		t.Errorf("attemptFixes returned error: %v", err)
	}

	if !fixCalled {
		t.Error("AutoFix function was not called")
	}
}

func TestVersionConstants(t *testing.T) {
	if MinGoVersion == "" {
		t.Error("MinGoVersion is empty")
	}

	if MinDockerVersion == "" {
		t.Error("MinDockerVersion is empty")
	}

	if MinDiskSpaceGB <= 0 {
		t.Errorf("MinDiskSpaceGB = %d, should be positive", MinDiskSpaceGB)
	}
}

func TestCheckResultStruct(t *testing.T) {
	result := CheckResult{
		Name:          "Test",
		Status:        StatusOK,
		Message:       "Test message",
		FixSuggestion: "Test fix",
		CanAutoFix:    true,
		AutoFix: func() error {
			return nil
		},
	}

	if result.Name != "Test" {
		t.Errorf("Name = %q, want %q", result.Name, "Test")
	}

	if result.Status != StatusOK {
		t.Errorf("Status = %d, want %d", result.Status, StatusOK)
	}

	if result.Message != "Test message" {
		t.Errorf("Message = %q, want %q", result.Message, "Test message")
	}

	if result.FixSuggestion != "Test fix" {
		t.Errorf("FixSuggestion = %q, want %q", result.FixSuggestion, "Test fix")
	}

	if !result.CanAutoFix {
		t.Error("CanAutoFix = false, want true")
	}

	if result.AutoFix == nil {
		t.Error("AutoFix is nil")
	}

	if err := result.AutoFix(); err != nil {
		t.Errorf("AutoFix returned error: %v", err)
	}
}
