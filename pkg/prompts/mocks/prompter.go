// Code generated manually for testing. Update as needed.

package mocks

import (
	"math/big"
	"net/url"
	"time"

	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
	"github.com/stretchr/testify/mock"
)

// Prompter is a mock implementation of prompts.Prompter
type Prompter struct {
	mock.Mock
}

func (m *Prompter) CapturePositiveBigInt(promptStr string) (*big.Int, error) {
	args := m.Called(promptStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *Prompter) CaptureAddress(promptStr string) (crypto.Address, error) {
	args := m.Called(promptStr)
	return args.Get(0).(crypto.Address), args.Error(1)
}

func (m *Prompter) CaptureNewFilepath(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureExistingFilepath(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureYesNo(promptStr string) (bool, error) {
	args := m.Called(promptStr)
	return args.Bool(0), args.Error(1)
}

func (m *Prompter) CaptureNoYes(promptStr string) (bool, error) {
	args := m.Called(promptStr)
	return args.Bool(0), args.Error(1)
}

func (m *Prompter) CaptureList(promptStr string, options []string) (string, error) {
	args := m.Called(promptStr, options)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureString(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureGitURL(promptStr string) (*url.URL, error) {
	args := m.Called(promptStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

func (m *Prompter) CaptureURL(promptStr string, validateConnection bool) (string, error) {
	args := m.Called(promptStr, validateConnection)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureStringAllowEmpty(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureEmail(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureIndex(promptStr string, options []any) (int, error) {
	args := m.Called(promptStr, options)
	return args.Int(0), args.Error(1)
}

func (m *Prompter) CaptureVersion(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureDuration(promptStr string) (time.Duration, error) {
	args := m.Called(promptStr)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *Prompter) CaptureDate(promptStr string) (time.Time, error) {
	args := m.Called(promptStr)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *Prompter) CaptureNodeID(promptStr string) (ids.NodeID, error) {
	args := m.Called(promptStr)
	return args.Get(0).(ids.NodeID), args.Error(1)
}

func (m *Prompter) CaptureID(promptStr string) (ids.ID, error) {
	args := m.Called(promptStr)
	return args.Get(0).(ids.ID), args.Error(1)
}

func (m *Prompter) CaptureWeight(promptStr string, validator func(uint64) error) (uint64, error) {
	args := m.Called(promptStr, validator)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *Prompter) CapturePositiveInt(promptStr string, comparators []prompts.Comparator) (int, error) {
	args := m.Called(promptStr, comparators)
	return args.Int(0), args.Error(1)
}

func (m *Prompter) CaptureUint64(promptStr string) (uint64, error) {
	args := m.Called(promptStr)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *Prompter) CaptureUint64Compare(promptStr string, comparators []prompts.Comparator) (uint64, error) {
	args := m.Called(promptStr, comparators)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *Prompter) CapturePChainAddress(promptStr string, network models.Network) (string, error) {
	args := m.Called(promptStr, network)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureFutureDate(promptStr string, minDate time.Time) (time.Time, error) {
	args := m.Called(promptStr, minDate)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *Prompter) ChooseKeyOrLedger(goal string) (bool, error) {
	args := m.Called(goal)
	return args.Bool(0), args.Error(1)
}

func (m *Prompter) CaptureValidatorBalance(promptStr string, availableBalance float64, minBalance float64) (float64, error) {
	args := m.Called(promptStr, availableBalance, minBalance)
	return args.Get(0).(float64), args.Error(1)
}

func (m *Prompter) CaptureListWithSize(prompt string, options []string, size int) ([]string, error) {
	args := m.Called(prompt, options, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *Prompter) CaptureFloat(promptStr string) (float64, error) {
	args := m.Called(promptStr)
	return args.Get(0).(float64), args.Error(1)
}

func (m *Prompter) CaptureAddresses(promptStr string) ([]crypto.Address, error) {
	args := m.Called(promptStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]crypto.Address), args.Error(1)
}

func (m *Prompter) CaptureXChainAddress(promptStr string, network models.Network) (string, error) {
	args := m.Called(promptStr, network)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureValidatedString(promptStr string, validator func(string) error) (string, error) {
	args := m.Called(promptStr, validator)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureRepoBranch(promptStr string, repo string) (string, error) {
	args := m.Called(promptStr, repo)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureRepoFile(promptStr string, repo string, branch string) (string, error) {
	args := m.Called(promptStr, repo, branch)
	return args.String(0), args.Error(1)
}

func (m *Prompter) CaptureInt(promptStr string, validator func(int) error) (int, error) {
	args := m.Called(promptStr, validator)
	return args.Int(0), args.Error(1)
}

func (m *Prompter) CaptureUint8(promptStr string) (uint8, error) {
	args := m.Called(promptStr)
	return args.Get(0).(uint8), args.Error(1)
}

func (m *Prompter) CaptureUint16(promptStr string) (uint16, error) {
	args := m.Called(promptStr)
	return args.Get(0).(uint16), args.Error(1)
}

func (m *Prompter) CaptureUint32(promptStr string) (uint32, error) {
	args := m.Called(promptStr)
	return args.Get(0).(uint32), args.Error(1)
}

func (m *Prompter) CaptureFujiDuration(promptStr string) (time.Duration, error) {
	args := m.Called(promptStr)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *Prompter) CaptureMainnetDuration(promptStr string) (time.Duration, error) {
	args := m.Called(promptStr)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *Prompter) CaptureMainnetL1StakingDuration(promptStr string) (time.Duration, error) {
	args := m.Called(promptStr)
	return args.Get(0).(time.Duration), args.Error(1)
}
