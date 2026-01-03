// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package prompts

import (
	"errors"
	"testing"
	"time"

	"github.com/luxfi/sdk/models"
	"github.com/stretchr/testify/require"
)

func TestNonInteractivePrompter_FailsWithError(t *testing.T) {
	p := NewNonInteractivePrompter()

	// Test CaptureYesNo
	_, err := p.CaptureYesNo("Confirm?")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNonInteractive))
	require.Contains(t, err.Error(), "Confirm?")

	// Test CaptureString
	_, err = p.CaptureString("Enter name")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNonInteractive))
	require.Contains(t, err.Error(), "Enter name")

	// Test CaptureList
	_, err = p.CaptureList("Choose", []string{"a", "b"})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNonInteractive))

	// Test CaptureUint64
	_, err = p.CaptureUint64("Enter number")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNonInteractive))
}

func TestNonInteractivePrompter_CustomMessage(t *testing.T) {
	p := NewNonInteractivePrompterWithMessage("use --chain-id flag")

	_, err := p.CaptureString("Chain ID")
	require.Error(t, err)
	require.Contains(t, err.Error(), "use --chain-id flag")
}

func TestNonInteractivePrompter_AllMethods(t *testing.T) {
	p := NewNonInteractivePrompter()

	// Test all methods return ErrNonInteractive
	tests := []struct {
		name string
		fn   func() error
	}{
		{"CapturePositiveBigInt", func() error { _, err := p.CapturePositiveBigInt(""); return err }},
		{"CaptureAddress", func() error { _, err := p.CaptureAddress(""); return err }},
		{"CaptureNewFilepath", func() error { _, err := p.CaptureNewFilepath(""); return err }},
		{"CaptureExistingFilepath", func() error { _, err := p.CaptureExistingFilepath(""); return err }},
		{"CaptureYesNo", func() error { _, err := p.CaptureYesNo(""); return err }},
		{"CaptureNoYes", func() error { _, err := p.CaptureNoYes(""); return err }},
		{"CaptureList", func() error { _, err := p.CaptureList("", nil); return err }},
		{"CaptureString", func() error { _, err := p.CaptureString(""); return err }},
		{"CaptureGitURL", func() error { _, err := p.CaptureGitURL(""); return err }},
		{"CaptureURL", func() error { _, err := p.CaptureURL("", false); return err }},
		{"CaptureStringAllowEmpty", func() error { _, err := p.CaptureStringAllowEmpty(""); return err }},
		{"CaptureEmail", func() error { _, err := p.CaptureEmail(""); return err }},
		{"CaptureIndex", func() error { _, err := p.CaptureIndex("", nil); return err }},
		{"CaptureVersion", func() error { _, err := p.CaptureVersion(""); return err }},
		{"CaptureDuration", func() error { _, err := p.CaptureDuration(""); return err }},
		{"CaptureDate", func() error { _, err := p.CaptureDate(""); return err }},
		{"CaptureNodeID", func() error { _, err := p.CaptureNodeID(""); return err }},
		{"CaptureID", func() error { _, err := p.CaptureID(""); return err }},
		{"CaptureWeight", func() error { _, err := p.CaptureWeight("", nil); return err }},
		{"CapturePositiveInt", func() error { _, err := p.CapturePositiveInt("", nil); return err }},
		{"CaptureUint64", func() error { _, err := p.CaptureUint64(""); return err }},
		{"CaptureUint64Compare", func() error { _, err := p.CaptureUint64Compare("", nil); return err }},
		{"CapturePChainAddress", func() error { _, err := p.CapturePChainAddress("", models.UndefinedNetwork); return err }},
		{"CaptureFutureDate", func() error { _, err := p.CaptureFutureDate("", time.Time{}); return err }},
		{"ChooseKeyOrLedger", func() error { _, err := p.ChooseKeyOrLedger(""); return err }},
		{"CaptureValidatorBalance", func() error { _, err := p.CaptureValidatorBalance("", 0, 0); return err }},
		{"CaptureListWithSize", func() error { _, err := p.CaptureListWithSize("", nil, 0); return err }},
		{"CaptureFloat", func() error { _, err := p.CaptureFloat("", nil); return err }},
		{"CaptureAddresses", func() error { _, err := p.CaptureAddresses(""); return err }},
		{"CaptureXChainAddress", func() error { _, err := p.CaptureXChainAddress("", models.UndefinedNetwork); return err }},
		{"CaptureValidatedString", func() error { _, err := p.CaptureValidatedString("", nil); return err }},
		{"CaptureRepoBranch", func() error { _, err := p.CaptureRepoBranch("", ""); return err }},
		{"CaptureRepoFile", func() error { _, err := p.CaptureRepoFile("", "", ""); return err }},
		{"CaptureInt", func() error { _, err := p.CaptureInt("", nil); return err }},
		{"CaptureUint8", func() error { _, err := p.CaptureUint8(""); return err }},
		{"CaptureFujiDuration", func() error { _, err := p.CaptureFujiDuration(""); return err }},
		{"CaptureMainnetDuration", func() error { _, err := p.CaptureMainnetDuration(""); return err }},
		{"CaptureMainnetL1StakingDuration", func() error { _, err := p.CaptureMainnetL1StakingDuration(""); return err }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			require.Error(t, err)
			require.True(t, errors.Is(err, ErrNonInteractive), "expected ErrNonInteractive for %s", tc.name)
		})
	}
}
