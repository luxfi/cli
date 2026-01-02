// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package safety

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")
	policy := DefaultPolicy(baseDir)

	// Check allowed prefixes
	if len(policy.AllowPrefixes) == 0 {
		t.Error("expected allow prefixes")
	}

	// Check deny prefixes
	if len(policy.DenyPrefixes) == 0 {
		t.Error("expected deny prefixes")
	}
}

func TestRemoveAllAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")
	policy := DefaultPolicy(baseDir)

	// Create an allowed path
	runsDir := filepath.Join(baseDir, "runs", "test")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Should succeed
	if err := RemoveAll(policy, runsDir); err != nil {
		t.Errorf("expected RemoveAll to succeed for allowed path, got: %v", err)
	}
}

func TestRemoveAllDenied(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")
	policy := DefaultPolicy(baseDir)

	// Create a denied path
	chainsDir := filepath.Join(baseDir, "chains")
	if err := os.MkdirAll(chainsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Should fail
	if err := RemoveAll(policy, chainsDir); err == nil {
		t.Error("expected RemoveAll to fail for denied path")
	}
}

func TestRemoveAllNotInAllowList(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")
	policy := DefaultPolicy(baseDir)

	// Create a path not in allow list
	randomDir := filepath.Join(tmpDir, "random")
	if err := os.MkdirAll(randomDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Should fail
	if err := RemoveAll(policy, randomDir); err == nil {
		t.Error("expected RemoveAll to fail for path not in allow list")
	}
}

func TestRemoveChainConfig(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")

	// Create chains directory with a chain
	chainDir := filepath.Join(baseDir, "chains", "mychain")
	if err := os.MkdirAll(chainDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Should succeed
	if err := RemoveChainConfig(baseDir, "mychain"); err != nil {
		t.Errorf("expected RemoveChainConfig to succeed, got: %v", err)
	}

	// Verify deleted
	if _, err := os.Stat(chainDir); !os.IsNotExist(err) {
		t.Error("expected chain directory to be deleted")
	}
}

func TestRemoveChainConfigInvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")

	tests := []string{
		"",
		".",
		"..",
		"../escape",
		"chains/../escape",
		"/absolute",
	}

	for _, name := range tests {
		if err := RemoveChainConfig(baseDir, name); err == nil {
			t.Errorf("expected RemoveChainConfig to fail for invalid name %q", name)
		}
	}
}

func TestIsProtected(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")
	policy := DefaultPolicy(baseDir)

	// Create protected paths
	chainsDir := filepath.Join(baseDir, "chains")
	keysDir := filepath.Join(baseDir, "keys")
	_ = os.MkdirAll(chainsDir, 0o755)
	_ = os.MkdirAll(keysDir, 0o755)

	// Should be protected
	if !IsProtected(policy, chainsDir) {
		t.Error("expected chains to be protected")
	}
	if !IsProtected(policy, keysDir) {
		t.Error("expected keys to be protected")
	}

	// runs should not be protected
	runsDir := filepath.Join(baseDir, "runs")
	_ = os.MkdirAll(runsDir, 0o755)
	if IsProtected(policy, runsDir) {
		t.Error("expected runs to not be protected")
	}
}

func TestIsAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".lux")
	policy := DefaultPolicy(baseDir)

	runsDir := filepath.Join(baseDir, "runs")
	_ = os.MkdirAll(runsDir, 0o755)

	if !IsAllowed(policy, runsDir) {
		t.Error("expected runs to be allowed")
	}

	chainsDir := filepath.Join(baseDir, "chains")
	_ = os.MkdirAll(chainsDir, 0o755)

	if IsAllowed(policy, chainsDir) {
		t.Error("expected chains to not be in allow list")
	}
}
