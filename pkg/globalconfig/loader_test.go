// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Test loading non-existent config returns nil
	ClearCache()
	config, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error loading non-existent config: %v", err)
	}
	if config != nil {
		t.Fatal("expected nil config for non-existent file")
	}

	// Test loading valid config
	testConfig := DefaultGlobalConfig()
	err = SaveGlobalConfig(tmpDir, &testConfig)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	ClearCache()
	loaded, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil config")
	}
	if loaded.Version != ConfigVersion {
		t.Errorf("expected version %s, got %s", ConfigVersion, loaded.Version)
	}
}

func TestSaveGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultGlobalConfig()
	err := SaveGlobalConfig(tmpDir, &config)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, GlobalConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "dir")
	err := os.MkdirAll(subDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Test not found
	_, err = FindProjectRoot(subDir)
	if !os.IsNotExist(err) {
		t.Fatal("expected ErrNotExist for missing project config")
	}

	// Create project config at root
	configPath := filepath.Join(tmpDir, ProjectConfigFile)
	data := []byte(`{"projectName":"test-project","version":"1.0.0"}`)
	err = os.WriteFile(configPath, data, 0o644)
	if err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// Test finding from subdirectory
	foundRoot, err := FindProjectRoot(subDir)
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}
	if foundRoot != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, foundRoot)
	}
}

func TestCacheClearing(t *testing.T) {
	tmpDir := t.TempDir()

	// Save initial config
	config1 := DefaultGlobalConfig()
	numNodes := uint32(3)
	config1.Local.NumNodes = &numNodes
	err := SaveGlobalConfig(tmpDir, &config1)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load and verify
	ClearCache()
	loaded1, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if *loaded1.Local.NumNodes != 3 {
		t.Errorf("expected numNodes=3, got %d", *loaded1.Local.NumNodes)
	}

	// Update config
	numNodes2 := uint32(7)
	config1.Local.NumNodes = &numNodes2
	err = SaveGlobalConfig(tmpDir, &config1)
	if err != nil {
		t.Fatalf("failed to save updated config: %v", err)
	}

	// Load again (cache should be updated)
	loaded2, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load updated config: %v", err)
	}
	if *loaded2.Local.NumNodes != 7 {
		t.Errorf("expected numNodes=7 after update, got %d", *loaded2.Local.NumNodes)
	}
}
