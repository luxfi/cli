// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/application"
)

// RunManager manages network run directories with stable symlinks
type RunManager struct {
	app     *application.Lux
	profile string // "mainnet", "testnet", "local", etc.
}

// NewRunManager creates a new run manager for the given profile
func NewRunManager(app *application.Lux, profile string) *RunManager {
	return &RunManager{
		app:     app,
		profile: profile,
	}
}

// ProfileDir returns the base directory for this profile
func (r *RunManager) ProfileDir() string {
	return filepath.Join(r.app.GetBaseDir(), "runs", r.profile)
}

// CurrentLink returns the path to the "current" symlink
func (r *RunManager) CurrentLink() string {
	return filepath.Join(r.ProfileDir(), "current")
}

// CurrentRunDir returns the actual run directory that "current" points to
func (r *RunManager) CurrentRunDir() (string, error) {
	linkPath := r.CurrentLink()
	target, err := os.Readlink(linkPath)
	if err != nil {
		return "", fmt.Errorf("no current run: %w", err)
	}

	// If relative, resolve against profile dir
	if !filepath.IsAbs(target) {
		target = filepath.Join(r.ProfileDir(), target)
	}

	return target, nil
}

// EnsureRunDir ensures a run directory exists with the following behavior:
// - Default (fresh=false, newRun=false): Reuse current symlink target if it exists
// - fresh=true: Wipe current target and recreate empty
// - newRun=true: Create new timestamped directory and update current symlink
func (r *RunManager) EnsureRunDir(fresh, newRun bool) (string, error) {
	base := r.ProfileDir()
	currentLink := r.CurrentLink()

	// Create base profile directory if needed
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Reuse current target unless explicitly starting a new run
	if !newRun {
		if target, err := os.Readlink(currentLink); err == nil {
			absPath := target
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(base, target)
			}

			if fresh {
				// Wipe and recreate
				_ = os.RemoveAll(absPath)
				if err := os.MkdirAll(absPath, 0755); err != nil {
					return "", fmt.Errorf("failed to recreate run directory: %w", err)
				}
			}
			return absPath, nil
		}
	}

	// Create a new timestamped run directory and update current symlink
	runName := fmt.Sprintf("run_%s", time.Now().Format("20060102_150405"))
	runDir := filepath.Join(base, runName)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create run directory: %w", err)
	}

	// Atomically update current symlink
	if err := r.updateCurrentLink(runDir); err != nil {
		return "", err
	}

	return runDir, nil
}

// SetCurrent updates the "current" symlink to point to the specified run
func (r *RunManager) SetCurrent(runDir string) error {
	return r.updateCurrentLink(runDir)
}

// updateCurrentLink atomically updates the current symlink
func (r *RunManager) updateCurrentLink(runDir string) error {
	base := r.ProfileDir()
	currentLink := r.CurrentLink()

	// Get relative path for cleaner symlink
	relPath, err := filepath.Rel(base, runDir)
	if err != nil {
		relPath = runDir // Fall back to absolute
	}

	// Create temp symlink then rename for atomicity
	tmpLink := filepath.Join(base, ".current_tmp")
	_ = os.Remove(tmpLink)

	if err := os.Symlink(relPath, tmpLink); err != nil {
		return fmt.Errorf("failed to create temp symlink: %w", err)
	}

	// Remove existing and rename temp to current
	_ = os.Remove(currentLink)
	if err := os.Rename(tmpLink, currentLink); err != nil {
		return fmt.Errorf("failed to update current symlink: %w", err)
	}

	return nil
}

// NodeDir returns the directory for a specific node in the current run
func (r *RunManager) NodeDir(nodeNum int) (string, error) {
	runDir, err := r.CurrentRunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(runDir, fmt.Sprintf("node%d", nodeNum)), nil
}

// ChainConfigDir returns the chain config directory for a node
func (r *RunManager) ChainConfigDir(nodeNum int) (string, error) {
	nodeDir, err := r.NodeDir(nodeNum)
	if err != nil {
		return "", err
	}
	return filepath.Join(nodeDir, "chainConfigs"), nil
}

// ListRuns returns all run directories for this profile
func (r *RunManager) ListRuns() ([]string, error) {
	entries, err := os.ReadDir(r.ProfileDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var runs []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "current" {
			runs = append(runs, entry.Name())
		}
	}
	return runs, nil
}

// CleanOldRuns removes old run directories, keeping the N most recent
func (r *RunManager) CleanOldRuns(keep int) error {
	runs, err := r.ListRuns()
	if err != nil {
		return err
	}

	if len(runs) <= keep {
		return nil
	}

	// Runs are named with timestamps, so sorting gives chronological order
	// Remove oldest runs
	toRemove := runs[:len(runs)-keep]
	for _, run := range toRemove {
		runPath := filepath.Join(r.ProfileDir(), run)
		if err := os.RemoveAll(runPath); err != nil {
			return fmt.Errorf("failed to remove old run %s: %w", run, err)
		}
	}

	return nil
}

// EnsureNetworkRunDir ensures a network run directory exists.
// If "current" symlink exists, reuses it. Otherwise creates a new timestamped run.
// Use `lux network clean` to wipe and start fresh.
func EnsureNetworkRunDir(baseRunsDir, network string) (string, error) {
	base := filepath.Join(baseRunsDir, network)
	currentLink := filepath.Join(base, "current")

	// Create base directory
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", fmt.Errorf("failed to create network run directory: %w", err)
	}

	// Reuse current target if it exists
	if target, err := os.Readlink(currentLink); err == nil {
		absPath := target
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(base, target)
		}
		return absPath, nil
	}

	// Create a new timestamped run directory and update current symlink
	runName := fmt.Sprintf("run_%s", time.Now().Format("20060102_150405"))
	runDir := filepath.Join(base, runName)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create run directory: %w", err)
	}

	// Atomically update current symlink
	tmpLink := filepath.Join(base, ".current_tmp")
	_ = os.Remove(tmpLink)

	if err := os.Symlink(runName, tmpLink); err != nil {
		return "", fmt.Errorf("failed to create temp symlink: %w", err)
	}

	_ = os.Remove(currentLink)
	if err := os.Rename(tmpLink, currentLink); err != nil {
		return "", fmt.Errorf("failed to update current symlink: %w", err)
	}

	return runDir, nil
}
