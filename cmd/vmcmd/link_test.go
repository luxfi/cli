// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vmcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/pkg/utils"
)

func TestVMID(t *testing.T) {
	tests := []struct {
		name     string
		vmName   string
		wantErr  bool
		wantVMID string
	}{
		{
			name:     "Lux EVM",
			vmName:   "Lux EVM",
			wantErr:  false,
			wantVMID: "ag3GReYPNuSR17rUP8acMdZipQBikdXNRKDyFszAysmy3vDXE",
		},
		{
			name:     "lux-evm",
			vmName:   "lux-evm",
			wantErr:  false,
			wantVMID: "pmSJB3vLVKGEGEPULpcsBwfYyd8dBHnCgbUNrMniLq6izCjKq",
		},
		{
			name:    "too long name",
			vmName:  "this-is-a-very-long-vm-name-that-exceeds-32-bytes",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vmID, err := utils.VMID(tt.vmName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("VMID() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("VMID() unexpected error: %v", err)
				return
			}
			if tt.wantVMID != "" && vmID.String() != tt.wantVMID {
				t.Errorf("VMID() = %s, want %s", vmID.String(), tt.wantVMID)
			}
		})
	}
}

func TestSymlinkOperations(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "vmcmd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a fake binary
	binaryPath := filepath.Join(tmpDir, "fake-vm")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0o755); err != nil { //nolint:gosec // G306: Test script needs to be executable
		t.Fatalf("failed to create fake binary: %v", err)
	}

	// Create plugins directory
	pluginDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginDir, 0o750); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	// Calculate VMID
	vmID, err := utils.VMID("test-vm")
	if err != nil {
		t.Fatalf("failed to calculate VMID: %v", err)
	}

	symlinkPath := filepath.Join(pluginDir, vmID.String())

	// Test creating symlink
	if err := os.Symlink(binaryPath, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Verify symlink exists and points to correct target
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if target != binaryPath {
		t.Errorf("symlink target = %s, want %s", target, binaryPath)
	}

	// Test updating symlink (atomic update)
	newBinaryPath := filepath.Join(tmpDir, "new-fake-vm")
	if err := os.WriteFile(newBinaryPath, []byte("#!/bin/sh\necho new"), 0o755); err != nil { //nolint:gosec // G306: Test script needs to be executable
		t.Fatalf("failed to create new binary: %v", err)
	}

	// Remove old symlink and create new one
	if err := os.Remove(symlinkPath); err != nil {
		t.Fatalf("failed to remove old symlink: %v", err)
	}
	if err := os.Symlink(newBinaryPath, symlinkPath); err != nil {
		t.Fatalf("failed to create new symlink: %v", err)
	}

	// Verify updated symlink
	target, err = os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("failed to read updated symlink: %v", err)
	}
	if target != newBinaryPath {
		t.Errorf("updated symlink target = %s, want %s", target, newBinaryPath)
	}

	// Test removing symlink
	if err := os.Remove(symlinkPath); err != nil {
		t.Fatalf("failed to remove symlink: %v", err)
	}

	// Verify symlink no longer exists
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Errorf("symlink should not exist after removal")
	}
}
