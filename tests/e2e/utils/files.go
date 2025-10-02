// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"os"
	"strings"

	"github.com/luxfi/cli/pkg/constants"
)

// CreateTmpFile creates a temporary file with the given name prefix and content
func CreateTmpFile(namePrefix string, content []byte) (string, error) {
	file, err := os.CreateTemp("", namePrefix+"*")
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := os.WriteFile(file.Name(), content, constants.DefaultPerms755); err != nil {
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}

// CreateTmpDir creates a temporary directory with the given prefix
func CreateTmpDir(prefix string) (string, error) {
	dir, err := os.MkdirTemp("", prefix+"*")
	if err != nil {
		return "", err
	}
	return dir, nil
}

// CleanupTmpFile removes a temporary file if it exists
func CleanupTmpFile(path string) error {
	if path == "" {
		return nil
	}
	// Only remove files in temp directory
	if strings.HasPrefix(path, os.TempDir()) {
		return os.RemoveAll(path)
	}
	return nil
}
