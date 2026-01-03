// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package prompts

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"time"

	"github.com/luxfi/crypto/common"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
)

// ErrNonInteractive is returned when a prompt is attempted in non-interactive mode.
// Commands should catch this error and provide actionable guidance.
var ErrNonInteractive = errors.New("cannot prompt in non-interactive mode")

// NonInteractivePrompter implements Prompter but fails fast on any prompt attempt.
// Use this in CI/script environments to detect missing flags early.
type NonInteractivePrompter struct {
	// FailMessage provides context about what flag/env var to set.
	// If empty, a default message is used.
	FailMessage string
}

// NewNonInteractivePrompter creates a prompter that fails fast on any interaction.
func NewNonInteractivePrompter() *NonInteractivePrompter {
	return &NonInteractivePrompter{}
}

// NewNonInteractivePrompterWithMessage creates a prompter with a custom fail message.
func NewNonInteractivePrompterWithMessage(msg string) *NonInteractivePrompter {
	return &NonInteractivePrompter{FailMessage: msg}
}

func (p *NonInteractivePrompter) fail(operation string) error {
	msg := p.FailMessage
	if msg == "" {
		msg = "use flags to provide required values, or unset LUX_NON_INTERACTIVE"
	}
	return fmt.Errorf("%w: %s - %s", ErrNonInteractive, operation, msg)
}

func (p *NonInteractivePrompter) CapturePositiveBigInt(promptStr string) (*big.Int, error) {
	return nil, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureAddress(promptStr string) (common.Address, error) {
	return common.Address{}, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureNewFilepath(promptStr string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureExistingFilepath(promptStr string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureYesNo(promptStr string) (bool, error) {
	return false, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureNoYes(promptStr string) (bool, error) {
	return false, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureList(promptStr string, options []string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureString(promptStr string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureGitURL(promptStr string) (*url.URL, error) {
	return nil, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureURL(promptStr string, validateConnection bool) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureStringAllowEmpty(promptStr string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureEmail(promptStr string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureIndex(promptStr string, options []any) (int, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureVersion(promptStr string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureDuration(promptStr string) (time.Duration, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureDate(promptStr string) (time.Time, error) {
	return time.Time{}, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureNodeID(promptStr string) (ids.NodeID, error) {
	return ids.EmptyNodeID, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureID(promptStr string) (ids.ID, error) {
	return ids.Empty, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureWeight(promptStr string, validator func(uint64) error) (uint64, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CapturePositiveInt(promptStr string, comparators []Comparator) (int, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureUint64(promptStr string) (uint64, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureUint64Compare(promptStr string, comparators []Comparator) (uint64, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CapturePChainAddress(promptStr string, network models.Network) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureFutureDate(promptStr string, minDate time.Time) (time.Time, error) {
	return time.Time{}, p.fail(promptStr)
}

func (p *NonInteractivePrompter) ChooseKeyOrLedger(goal string) (bool, error) {
	return false, p.fail("choose key or ledger for " + goal)
}

func (p *NonInteractivePrompter) CaptureValidatorBalance(promptStr string, availableBalance float64, minBalance float64) (float64, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureListWithSize(prompt string, options []string, size int) ([]string, error) {
	return nil, p.fail(prompt)
}

func (p *NonInteractivePrompter) CaptureFloat(promptStr string, validator func(float64) error) (float64, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureAddresses(promptStr string) ([]common.Address, error) {
	return nil, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureXChainAddress(promptStr string, network models.Network) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureValidatedString(promptStr string, validator func(string) error) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureRepoBranch(promptStr string, repo string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureRepoFile(promptStr string, repo string, branch string) (string, error) {
	return "", p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureInt(promptStr string, validator func(int) error) (int, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureUint8(promptStr string) (uint8, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureFujiDuration(promptStr string) (time.Duration, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureMainnetDuration(promptStr string) (time.Duration, error) {
	return 0, p.fail(promptStr)
}

func (p *NonInteractivePrompter) CaptureMainnetL1StakingDuration(promptStr string) (time.Duration, error) {
	return 0, p.fail(promptStr)
}

// Verify NonInteractivePrompter implements Prompter at compile time.
var _ Prompter = (*NonInteractivePrompter)(nil)
