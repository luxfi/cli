// Code generated manually for testing. DO NOT EDIT.

package mocks

import (
	"math/big"
	"net/url"
	"time"

	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"github.com/stretchr/testify/mock"
)

// SDKPrompter is a mock implementation of sdk/prompts.Prompter
type SDKPrompter struct {
	mock.Mock
}

func (m *SDKPrompter) CapturePositiveBigInt(promptStr string) (*big.Int, error) {
	args := m.Called(promptStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *SDKPrompter) CaptureAddress(promptStr string) (crypto.Address, error) {
	args := m.Called(promptStr)
	return args.Get(0).(crypto.Address), args.Error(1)
}

func (m *SDKPrompter) CaptureNewFilepath(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureExistingFilepath(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureYesNo(promptStr string) (bool, error) {
	args := m.Called(promptStr)
	return args.Bool(0), args.Error(1)
}

func (m *SDKPrompter) CaptureNoYes(promptStr string) (bool, error) {
	args := m.Called(promptStr)
	return args.Bool(0), args.Error(1)
}

func (m *SDKPrompter) CaptureList(promptStr string, options []string) (string, error) {
	args := m.Called(promptStr, options)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureString(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureGitURL(promptStr string) (*url.URL, error) {
	args := m.Called(promptStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

func (m *SDKPrompter) CaptureURL(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureStringAllowEmpty(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureEmail(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureIndex(promptStr string, options []any) (int, error) {
	args := m.Called(promptStr, options)
	return args.Int(0), args.Error(1)
}

func (m *SDKPrompter) CaptureVersion(promptStr string) (string, error) {
	args := m.Called(promptStr)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureDuration(promptStr string) (time.Duration, error) {
	args := m.Called(promptStr)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *SDKPrompter) CaptureDate(promptStr string) (time.Time, error) {
	args := m.Called(promptStr)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *SDKPrompter) CaptureNodeID(promptStr string) (ids.NodeID, error) {
	args := m.Called(promptStr)
	return args.Get(0).(ids.NodeID), args.Error(1)
}

func (m *SDKPrompter) CaptureID(promptStr string) (ids.ID, error) {
	args := m.Called(promptStr)
	return args.Get(0).(ids.ID), args.Error(1)
}

func (m *SDKPrompter) CaptureWeight(promptStr string) (uint64, error) {
	args := m.Called(promptStr)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *SDKPrompter) CapturePositiveInt(promptStr string, comparators []prompts.Comparator) (int, error) {
	args := m.Called(promptStr, comparators)
	return args.Int(0), args.Error(1)
}

func (m *SDKPrompter) CaptureUint64(promptStr string) (uint64, error) {
	args := m.Called(promptStr)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *SDKPrompter) CaptureUint64Compare(promptStr string, comparators []prompts.Comparator) (uint64, error) {
	args := m.Called(promptStr, comparators)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *SDKPrompter) CapturePChainAddress(promptStr string, network models.Network) (string, error) {
	args := m.Called(promptStr, network)
	return args.String(0), args.Error(1)
}

func (m *SDKPrompter) CaptureFutureDate(promptStr string, minDate time.Time) (time.Time, error) {
	args := m.Called(promptStr, minDate)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *SDKPrompter) ChooseKeyOrLedger(goal string) (bool, error) {
	args := m.Called(goal)
	return args.Bool(0), args.Error(1)
}

func (m *SDKPrompter) CaptureValidatorBalance(promptStr string, availableBalance float64, minBalance float64) (float64, error) {
	args := m.Called(promptStr, availableBalance, minBalance)
	return args.Get(0).(float64), args.Error(1)
}

func (m *SDKPrompter) CaptureListWithSize(prompt string, options []string, size int) ([]string, error) {
	args := m.Called(prompt, options, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *SDKPrompter) CaptureFloat(promptStr string) (float64, error) {
	args := m.Called(promptStr)
	return args.Get(0).(float64), args.Error(1)
}