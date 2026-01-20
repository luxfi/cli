// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/ux"
)

// CleanupConfig configures cleanup behavior
type CleanupConfig struct {
	// MaxLogSize is the maximum size in bytes for netrunner-server.log files
	// Default: 100MB
	MaxLogSize int64

	// MaxLogAge is the maximum age for log files before rotation
	// Default: 7 days
	MaxLogAge time.Duration

	// MaxBackupAge is the maximum age for .backup.* directories
	// Default: 7 days
	MaxBackupAge time.Duration

	// MaxStaleRunAge is the maximum age for stale run directories
	// Default: 24 hours
	MaxStaleRunAge time.Duration

	// DryRun if true, only report what would be deleted
	DryRun bool

	// Verbose enables verbose output
	Verbose bool
}

// DefaultCleanupConfig returns sensible defaults
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		MaxLogSize:     100 * 1024 * 1024, // 100MB
		MaxLogAge:      7 * 24 * time.Hour,
		MaxBackupAge:   7 * 24 * time.Hour,
		MaxStaleRunAge: 24 * time.Hour,
		DryRun:         false,
		Verbose:        false,
	}
}

// CleanupResult contains statistics from cleanup operation
type CleanupResult struct {
	LogsDeleted       int
	LogBytesFreed     int64
	BackupsDeleted    int
	BackupBytesFreed  int64
	StaleRunsDeleted  int
	StaleRunBytesFreed int64
	Errors            []error
}

// TotalBytesFreed returns total bytes freed
func (r CleanupResult) TotalBytesFreed() int64 {
	return r.LogBytesFreed + r.BackupBytesFreed + r.StaleRunBytesFreed
}

// Cleanup performs cleanup of logs, backups, and stale runs
func (sm *SnapshotManager) Cleanup(cfg CleanupConfig) CleanupResult {
	result := CleanupResult{}

	// 1. Clean netrunner-server.log files in ~/.lux/runs/server/
	sm.cleanupLogs(cfg, &result)

	// 2. Clean old .backup.* directories in ~/.lux/runs/
	sm.cleanupBackups(cfg, &result)

	// 3. Clean stale run directories
	sm.cleanupStaleRuns(cfg, &result)

	return result
}

// cleanupLogs handles netrunner-server.log cleanup and rotation
func (sm *SnapshotManager) cleanupLogs(cfg CleanupConfig, result *CleanupResult) {
	serverDir := filepath.Join(sm.baseDir, "runs", "server")

	// Walk all subdirectories looking for log files
	err := filepath.Walk(serverDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		if info.IsDir() {
			return nil
		}

		// Only process netrunner-server.log files
		if info.Name() != "netrunner-server.log" {
			return nil
		}

		shouldDelete := false
		reason := ""

		// Check size
		if info.Size() > cfg.MaxLogSize {
			shouldDelete = true
			reason = fmt.Sprintf("size %d > %d bytes", info.Size(), cfg.MaxLogSize)
		}

		// Check age
		if time.Since(info.ModTime()) > cfg.MaxLogAge {
			shouldDelete = true
			reason = fmt.Sprintf("age %v > %v", time.Since(info.ModTime()).Round(time.Hour), cfg.MaxLogAge)
		}

		if shouldDelete {
			if cfg.Verbose {
				ux.Logger.PrintToUser("  Log: %s (%s)", path, reason)
			}

			if !cfg.DryRun {
				if err := os.Remove(path); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("failed to remove log %s: %w", path, err))
				} else {
					result.LogsDeleted++
					result.LogBytesFreed += info.Size()
				}
			} else {
				result.LogsDeleted++
				result.LogBytesFreed += info.Size()
			}
		}

		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to walk server dir: %w", err))
	}
}

// cleanupBackups removes old .backup.* directories
func (sm *SnapshotManager) cleanupBackups(cfg CleanupConfig, result *CleanupResult) {
	runsDir := filepath.Join(sm.baseDir, "runs")

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Errorf("failed to read runs dir: %w", err))
		}
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Match .backup.* pattern
		if !strings.Contains(entry.Name(), ".backup.") {
			continue
		}

		backupPath := filepath.Join(runsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > cfg.MaxBackupAge {
			size := dirSize(backupPath)

			if cfg.Verbose {
				ux.Logger.PrintToUser("  Backup: %s (age %v)", backupPath, time.Since(info.ModTime()).Round(time.Hour))
			}

			if !cfg.DryRun {
				if err := os.RemoveAll(backupPath); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("failed to remove backup %s: %w", backupPath, err))
				} else {
					result.BackupsDeleted++
					result.BackupBytesFreed += size
				}
			} else {
				result.BackupsDeleted++
				result.BackupBytesFreed += size
			}
		}
	}
}

// cleanupStaleRuns removes run directories that are no longer associated with a running process
func (sm *SnapshotManager) cleanupStaleRuns(cfg CleanupConfig, result *CleanupResult) {
	serverDir := filepath.Join(sm.baseDir, "runs", "server")

	// Iterate network types
	networkTypes := []string{"mainnet", "testnet", "devnet", "custom"}
	for _, netType := range networkTypes {
		netDir := filepath.Join(serverDir, netType)

		entries, err := os.ReadDir(netDir)
		if err != nil {
			continue // Directory may not exist
		}

		// Sort entries by name (timestamp-based naming means older first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		// Keep only the most recent run directory, clean up older ones
		for i := 0; i < len(entries)-1; i++ {
			entry := entries[i]
			if !entry.IsDir() {
				continue
			}

			runPath := filepath.Join(netDir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if time.Since(info.ModTime()) > cfg.MaxStaleRunAge {
				size := dirSize(runPath)

				if cfg.Verbose {
					ux.Logger.PrintToUser("  Stale run: %s (age %v)", runPath, time.Since(info.ModTime()).Round(time.Hour))
				}

				if !cfg.DryRun {
					if err := os.RemoveAll(runPath); err != nil {
						result.Errors = append(result.Errors, fmt.Errorf("failed to remove stale run %s: %w", runPath, err))
					} else {
						result.StaleRunsDeleted++
						result.StaleRunBytesFreed += size
					}
				} else {
					result.StaleRunsDeleted++
					result.StaleRunBytesFreed += size
				}
			}
		}
	}
}

// RotateLog rotates a log file if it exceeds the size limit
// Returns the path to the rotated file, or empty string if no rotation needed
func RotateLog(logPath string, maxSize int64) (string, error) {
	info, err := os.Stat(logPath)
	if err != nil {
		return "", nil // File doesn't exist, nothing to rotate
	}

	if info.Size() <= maxSize {
		return "", nil // No rotation needed
	}

	// Create rotated filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", logPath, timestamp)

	// Rename current log to rotated
	if err := os.Rename(logPath, rotatedPath); err != nil {
		return "", fmt.Errorf("failed to rotate log: %w", err)
	}

	return rotatedPath, nil
}

// TruncateLog truncates a log file to the last N bytes, preserving recent entries
func TruncateLog(logPath string, keepBytes int64) error {
	info, err := os.Stat(logPath)
	if err != nil {
		return nil // File doesn't exist
	}

	if info.Size() <= keepBytes {
		return nil // No truncation needed
	}

	// Read the last keepBytes from the file
	f, err := os.Open(logPath)
	if err != nil {
		return err
	}

	// Seek to position where we want to start keeping
	offset := info.Size() - keepBytes
	if _, err := f.Seek(offset, 0); err != nil {
		f.Close()
		return err
	}

	// Read remaining content
	content := make([]byte, keepBytes)
	n, err := f.Read(content)
	f.Close()
	if err != nil {
		return err
	}

	// Find first newline to avoid partial line at start
	for i := 0; i < n && i < 1024; i++ {
		if content[i] == '\n' {
			content = content[i+1:]
			break
		}
	}

	// Write truncated content back
	return os.WriteFile(logPath, content, 0o644)
}

// dirSize calculates the total size of a directory
func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// FormatBytes formats bytes in human-readable form
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
