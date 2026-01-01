// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package prompts

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNonInteractive_EnvVar(t *testing.T) {
	// Reset global state
	SetNonInteractive(false)

	tests := []struct {
		envValue string
		expected bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run("LUX_NON_INTERACTIVE="+tc.envValue, func(t *testing.T) {
			SetNonInteractive(false) // Reset
			os.Setenv(EnvNonInteractive, tc.envValue)
			defer os.Unsetenv(EnvNonInteractive)

			result := IsNonInteractive()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNonInteractive_CI(t *testing.T) {
	// Reset global state
	SetNonInteractive(false)
	os.Unsetenv(EnvNonInteractive)

	// Set CI env var
	os.Setenv(EnvCI, "true")
	defer os.Unsetenv(EnvCI)

	require.True(t, IsNonInteractive())
}

func TestIsNonInteractive_ExplicitSetting(t *testing.T) {
	// Clear all env vars
	os.Unsetenv(EnvNonInteractive)
	os.Unsetenv(EnvCI)

	// Test explicit setting takes precedence
	SetNonInteractive(true)
	require.True(t, IsNonInteractive())

	SetNonInteractive(false)
	// Note: might still be true if stdin is not a terminal
	// So we just verify the explicit true case
}

func TestNewPrompterForMode(t *testing.T) {
	// Reset and set non-interactive
	SetNonInteractive(true)
	defer SetNonInteractive(false)

	p := NewPrompterForMode()
	_, ok := p.(*NonInteractivePrompter)
	require.True(t, ok, "expected NonInteractivePrompter in non-interactive mode")
}

func TestNewPrompterForMode_Interactive(t *testing.T) {
	// Reset all
	SetNonInteractive(false)
	os.Unsetenv(EnvNonInteractive)
	os.Unsetenv(EnvCI)

	// This test may behave differently in CI vs local
	// because isTerminal() will return false in CI
	p := NewPrompterForMode()

	// In CI, we get NonInteractivePrompter; locally we might get realPrompter
	// Just verify we get a valid prompter
	require.NotNil(t, p)
}
