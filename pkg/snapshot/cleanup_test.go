// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanup_LogFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "runs", "server", "mainnet", "12345")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a large log file
	logPath := filepath.Join(serverDir, "netrunner-server.log")
	largeContent := make([]byte, 200*1024*1024) // 200MB
	if err := os.WriteFile(logPath, largeContent, 0o644); err != nil {
		t.Fatal(err)
	}

	sm := NewSnapshotManager(tmpDir)
	cfg := CleanupConfig{
		MaxLogSize:     100 * 1024 * 1024, // 100MB threshold
		MaxLogAge:      7 * 24 * time.Hour,
		MaxBackupAge:   7 * 24 * time.Hour,
		MaxStaleRunAge: 24 * time.Hour,
		DryRun:         false,
		Verbose:        false,
	}

	result := sm.Cleanup(cfg)

	if result.LogsDeleted != 1 {
		t.Errorf("expected 1 log deleted, got %d", result.LogsDeleted)
	}
	if result.LogBytesFreed != int64(len(largeContent)) {
		t.Errorf("expected %d bytes freed, got %d", len(largeContent), result.LogBytesFreed)
	}

	// Verify file was actually deleted
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("log file should have been deleted")
	}
}

func TestCleanup_DryRun(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "runs", "server", "testnet", "12345")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a large log file
	logPath := filepath.Join(serverDir, "netrunner-server.log")
	largeContent := make([]byte, 200*1024*1024)
	if err := os.WriteFile(logPath, largeContent, 0o644); err != nil {
		t.Fatal(err)
	}

	sm := NewSnapshotManager(tmpDir)
	cfg := CleanupConfig{
		MaxLogSize:     100 * 1024 * 1024,
		MaxLogAge:      7 * 24 * time.Hour,
		MaxBackupAge:   7 * 24 * time.Hour,
		MaxStaleRunAge: 24 * time.Hour,
		DryRun:         true, // Dry run mode
		Verbose:        false,
	}

	result := sm.Cleanup(cfg)

	if result.LogsDeleted != 1 {
		t.Errorf("expected 1 log identified for deletion, got %d", result.LogsDeleted)
	}

	// In dry run mode, file should NOT be deleted
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file should NOT have been deleted in dry run mode")
	}
}

func TestCleanup_BackupDirectories(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	runsDir := filepath.Join(tmpDir, "runs")

	// Create old backup directory
	oldBackupDir := filepath.Join(runsDir, "custom.backup.20240101-120000")
	if err := os.MkdirAll(oldBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file inside to give it some size
	testFile := filepath.Join(oldBackupDir, "test.db")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set modification time to old date
	oldTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	os.Chtimes(oldBackupDir, oldTime, oldTime)

	sm := NewSnapshotManager(tmpDir)
	cfg := CleanupConfig{
		MaxLogSize:     100 * 1024 * 1024,
		MaxLogAge:      7 * 24 * time.Hour,
		MaxBackupAge:   7 * 24 * time.Hour, // 7 day threshold
		MaxStaleRunAge: 24 * time.Hour,
		DryRun:         false,
		Verbose:        false,
	}

	result := sm.Cleanup(cfg)

	if result.BackupsDeleted != 1 {
		t.Errorf("expected 1 backup deleted, got %d", result.BackupsDeleted)
	}

	// Verify directory was actually deleted
	if _, err := os.Stat(oldBackupDir); !os.IsNotExist(err) {
		t.Error("backup directory should have been deleted")
	}
}

func TestCleanup_StaleRunDirectories(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	netDir := filepath.Join(tmpDir, "runs", "server", "devnet")
	if err := os.MkdirAll(netDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create two run directories - one old, one current
	oldRunDir := filepath.Join(netDir, "1000") // Old PID
	newRunDir := filepath.Join(netDir, "2000") // Current PID
	if err := os.MkdirAll(oldRunDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newRunDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create files inside directories
	if err := os.WriteFile(filepath.Join(oldRunDir, "log.txt"), make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newRunDir, "log.txt"), make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set old run directory modification time
	oldTime := time.Now().Add(-48 * time.Hour) // 2 days ago
	os.Chtimes(oldRunDir, oldTime, oldTime)

	sm := NewSnapshotManager(tmpDir)
	cfg := CleanupConfig{
		MaxLogSize:     100 * 1024 * 1024,
		MaxLogAge:      7 * 24 * time.Hour,
		MaxBackupAge:   7 * 24 * time.Hour,
		MaxStaleRunAge: 24 * time.Hour, // 24 hour threshold
		DryRun:         false,
		Verbose:        false,
	}

	result := sm.Cleanup(cfg)

	if result.StaleRunsDeleted != 1 {
		t.Errorf("expected 1 stale run deleted, got %d", result.StaleRunsDeleted)
	}

	// Verify old directory was deleted
	if _, err := os.Stat(oldRunDir); !os.IsNotExist(err) {
		t.Error("old run directory should have been deleted")
	}

	// Verify new directory was NOT deleted
	if _, err := os.Stat(newRunDir); os.IsNotExist(err) {
		t.Error("new run directory should NOT have been deleted")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 100, "100.0 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestRotateLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Create a log file larger than threshold
	content := make([]byte, 200)
	if err := os.WriteFile(logPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Rotate with threshold below file size
	rotatedPath, err := RotateLog(logPath, 100)
	if err != nil {
		t.Fatal(err)
	}

	if rotatedPath == "" {
		t.Error("expected rotation to occur")
	}

	// Original file should not exist
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("original log file should have been renamed")
	}

	// Rotated file should exist
	if _, err := os.Stat(rotatedPath); os.IsNotExist(err) {
		t.Error("rotated log file should exist")
	}
}

func TestRotateLog_NoRotationNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Create a small log file
	content := make([]byte, 50)
	if err := os.WriteFile(logPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Rotate with threshold above file size
	rotatedPath, err := RotateLog(logPath, 100)
	if err != nil {
		t.Fatal(err)
	}

	if rotatedPath != "" {
		t.Error("no rotation should have occurred")
	}

	// Original file should still exist
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("original log file should still exist")
	}
}

func TestTruncateLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Create a log file with multiple lines
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Truncate to keep only last 15 bytes
	if err := TruncateLog(logPath, 15); err != nil {
		t.Fatal(err)
	}

	// Read result
	result, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Should have kept only the last portion
	if len(result) > 15 {
		t.Errorf("truncated file should be at most 15 bytes, got %d", len(result))
	}
}

func TestDefaultCleanupConfig(t *testing.T) {
	cfg := DefaultCleanupConfig()

	if cfg.MaxLogSize != 100*1024*1024 {
		t.Errorf("expected MaxLogSize 100MB, got %d", cfg.MaxLogSize)
	}
	if cfg.MaxLogAge != 7*24*time.Hour {
		t.Errorf("expected MaxLogAge 7 days, got %v", cfg.MaxLogAge)
	}
	if cfg.MaxBackupAge != 7*24*time.Hour {
		t.Errorf("expected MaxBackupAge 7 days, got %v", cfg.MaxBackupAge)
	}
	if cfg.MaxStaleRunAge != 24*time.Hour {
		t.Errorf("expected MaxStaleRunAge 24 hours, got %v", cfg.MaxStaleRunAge)
	}
	if cfg.DryRun {
		t.Error("DryRun should default to false")
	}
	if cfg.Verbose {
		t.Error("Verbose should default to false")
	}
}

func TestCleanupResult_TotalBytesFreed(t *testing.T) {
	result := CleanupResult{
		LogBytesFreed:      1000,
		BackupBytesFreed:   2000,
		StaleRunBytesFreed: 3000,
	}

	total := result.TotalBytesFreed()
	if total != 6000 {
		t.Errorf("expected total 6000, got %d", total)
	}
}
