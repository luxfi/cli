// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

import (
	"os"
	"testing"
)

func TestDetectEnvironmentDevelopment(t *testing.T) {
	// Clear CI-related env vars
	originalCI := os.Getenv("CI")
	originalGH := os.Getenv("GITHUB_ACTIONS")
	originalCS := os.Getenv("CODESPACES")
	originalProd := os.Getenv("PRODUCTION")

	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("JENKINS_URL")
	os.Unsetenv("CODESPACES")
	os.Unsetenv("CODESPACE_NAME")
	os.Unsetenv("PRODUCTION")
	os.Unsetenv("NODE_ENV")

	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		}
		if originalGH != "" {
			os.Setenv("GITHUB_ACTIONS", originalGH)
		}
		if originalCS != "" {
			os.Setenv("CODESPACES", originalCS)
		}
		if originalProd != "" {
			os.Setenv("PRODUCTION", originalProd)
		}
	}()

	env := DetectEnvironment()
	if env != EnvDevelopment {
		t.Errorf("expected %s, got %s", EnvDevelopment, env)
	}
}

func TestDetectEnvironmentCI(t *testing.T) {
	original := os.Getenv("CI")
	os.Setenv("CI", "true")
	defer func() {
		if original != "" {
			os.Setenv("CI", original)
		} else {
			os.Unsetenv("CI")
		}
	}()

	env := DetectEnvironment()
	if env != EnvCI {
		t.Errorf("expected %s, got %s", EnvCI, env)
	}
}

func TestDetectEnvironmentGitHubActions(t *testing.T) {
	original := os.Getenv("GITHUB_ACTIONS")
	os.Setenv("GITHUB_ACTIONS", "true")
	defer func() {
		if original != "" {
			os.Setenv("GITHUB_ACTIONS", original)
		} else {
			os.Unsetenv("GITHUB_ACTIONS")
		}
	}()

	env := DetectEnvironment()
	if env != EnvCI {
		t.Errorf("expected %s for GitHub Actions, got %s", EnvCI, env)
	}
}

func TestDetectEnvironmentCodespaces(t *testing.T) {
	originalCI := os.Getenv("CI")
	originalCS := os.Getenv("CODESPACES")
	os.Unsetenv("CI")
	os.Setenv("CODESPACES", "true")
	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		}
		if originalCS != "" {
			os.Setenv("CODESPACES", originalCS)
		} else {
			os.Unsetenv("CODESPACES")
		}
	}()

	env := DetectEnvironment()
	if env != EnvCodespace {
		t.Errorf("expected %s, got %s", EnvCodespace, env)
	}
}

func TestGetSmartDefaultsCI(t *testing.T) {
	original := os.Getenv("CI")
	os.Setenv("CI", "true")
	defer func() {
		if original != "" {
			os.Setenv("CI", original)
		} else {
			os.Unsetenv("CI")
		}
	}()

	defaults := GetSmartDefaults()

	if defaults.Environment != EnvCI {
		t.Errorf("expected env %s, got %s", EnvCI, defaults.Environment)
	}
	if defaults.SuggestedNumNodes != 3 {
		t.Errorf("expected 3 nodes for CI, got %d", defaults.SuggestedNumNodes)
	}
	if defaults.SuggestedInstance != "small" {
		t.Errorf("expected small instance for CI, got %s", defaults.SuggestedInstance)
	}
}

func TestSuggestTokenSupply(t *testing.T) {
	testnetSupply := SuggestTokenSupply(true)
	if testnetSupply != DefaultTokenSupply {
		t.Errorf("expected testnet supply %s, got %s", DefaultTokenSupply, testnetSupply)
	}

	prodSupply := SuggestTokenSupply(false)
	if prodSupply == DefaultTokenSupply {
		t.Error("expected different supply for production")
	}
}

func TestIsAutoTrackRecommended(t *testing.T) {
	// Clear environment
	originalCI := os.Getenv("CI")
	originalProd := os.Getenv("PRODUCTION")
	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("JENKINS_URL")
	os.Unsetenv("CODESPACES")
	os.Unsetenv("PRODUCTION")

	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		}
		if originalProd != "" {
			os.Setenv("PRODUCTION", originalProd)
		}
	}()

	// Development should recommend auto-track
	if !IsAutoTrackRecommended() {
		t.Error("expected auto-track recommended in development")
	}

	// CI should recommend auto-track
	os.Setenv("CI", "true")
	if !IsAutoTrackRecommended() {
		t.Error("expected auto-track recommended in CI")
	}
}
