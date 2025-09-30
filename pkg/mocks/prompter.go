// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mocks

import (
	"math/big"
	"time"

	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
)

// Prompter is a mock implementation of the Prompter interface for testing
type Prompter struct {
	CaptureStringVal        string
	CaptureStringErr        error
	CaptureYesNoVal         bool
	CaptureYesNoErr         error
	CaptureListVal          string
	CaptureListErr          error
	CaptureIndexVal         int
	CaptureIndexErr         error
	CaptureUintVal          uint64
	CaptureUintErr          error
	CaptureUint16Val        uint16
	CaptureUint16Err        error
	CaptureUint32Val        uint32
	CaptureUint32Err        error
	CaptureFloatVal         float64
	CaptureFloatErr         error
	CaptureAddressVal       crypto.Address
	CaptureAddressErr       error
	CapturePChainAddressVal ids.ShortID
	CapturePChainAddressErr error
	CaptureIDVal            ids.ID
	CaptureIDErr            error
	CaptureDurationVal      time.Duration
	CaptureDurationErr      error
	CaptureEmailVal         string
	CaptureEmailErr         error
	CaptureURLVal           string
	CaptureURLErr           error
	CaptureListEditVal      []string
	CaptureListEditErr      error
}

func (m *Prompter) CaptureString(prompt string) (string, error) {
	return m.CaptureStringVal, m.CaptureStringErr
}

func (m *Prompter) CaptureStringAllowEmpty(prompt string) (string, error) {
	return m.CaptureStringVal, m.CaptureStringErr
}

func (m *Prompter) CaptureYesNo(prompt string) (bool, error) {
	return m.CaptureYesNoVal, m.CaptureYesNoErr
}

func (m *Prompter) CaptureNoYes(prompt string) (bool, error) {
	return m.CaptureYesNoVal, m.CaptureYesNoErr
}

func (m *Prompter) CaptureList(prompt string, options []string) (string, error) {
	return m.CaptureListVal, m.CaptureListErr
}

func (m *Prompter) CaptureIndex(prompt string, options []any) (int, error) {
	return m.CaptureIndexVal, m.CaptureIndexErr
}

func (m *Prompter) CaptureUint64(prompt string) (uint64, error) {
	return m.CaptureUintVal, m.CaptureUintErr
}

func (m *Prompter) CaptureUint16(prompt string) (uint16, error) {
	return m.CaptureUint16Val, m.CaptureUint16Err
}

func (m *Prompter) CaptureUint32(prompt string) (uint32, error) {
	return m.CaptureUint32Val, m.CaptureUint32Err
}

func (m *Prompter) CaptureFloat(prompt string) (float64, error) {
	return m.CaptureFloatVal, m.CaptureFloatErr
}

func (m *Prompter) CaptureAddress(prompt string) (crypto.Address, error) {
	return m.CaptureAddressVal, m.CaptureAddressErr
}

func (m *Prompter) CapturePChainAddress(prompt string, network string) (ids.ShortID, error) {
	return m.CapturePChainAddressVal, m.CapturePChainAddressErr
}

func (m *Prompter) CaptureID(prompt string) (ids.ID, error) {
	return m.CaptureIDVal, m.CaptureIDErr
}

func (m *Prompter) CaptureDuration(prompt string) (time.Duration, error) {
	return m.CaptureDurationVal, m.CaptureDurationErr
}

func (m *Prompter) CaptureDate(prompt string) (time.Time, error) {
	return time.Time{}, nil
}

func (m *Prompter) CaptureEmail(prompt string) (string, error) {
	return m.CaptureEmailVal, m.CaptureEmailErr
}

func (m *Prompter) CaptureURL(prompt string) (string, error) {
	return m.CaptureURLVal, m.CaptureURLErr
}

func (m *Prompter) CaptureListEdit(prompt string, initialList []string, info string) ([]string, bool, error) {
	return m.CaptureListEditVal, false, m.CaptureListEditErr
}

func (m *Prompter) CaptureExistingFilepath(prompt string) (string, error) {
	return m.CaptureStringVal, m.CaptureStringErr
}

func (m *Prompter) CaptureNewFilepath(prompt string) (string, error) {
	return m.CaptureStringVal, m.CaptureStringErr
}

func (m *Prompter) CaptureValidatedString(prompt string, validator func(string) error) (string, error) {
	return m.CaptureStringVal, m.CaptureStringErr
}

func (m *Prompter) CaptureNodeID(prompt string) (ids.NodeID, error) {
	return ids.NodeID{}, nil
}

func (m *Prompter) CaptureUint64Compare(prompt string, comparators []interface{}) (uint64, error) {
	return m.CaptureUintVal, m.CaptureUintErr
}

func (m *Prompter) CaptureVersion(prompt string) (string, error) {
	return m.CaptureStringVal, m.CaptureStringErr
}

func (m *Prompter) CaptureUint8(prompt string) (uint8, error) {
	return 0, nil
}

func (m *Prompter) CapturePositiveBigInt(prompt string) (*big.Int, error) {
	return new(big.Int), nil
}

func (m *Prompter) ChooseKeyOrLedger(goal string) (bool, error) {
	return false, nil
}

func (m *Prompter) CaptureStringSlice(prompt string) ([]string, error) {
	return []string{}, nil
}

func (m *Prompter) SelectFromListWithSize(prompt string, options []string, size int) (string, error) {
	return m.CaptureListVal, m.CaptureListErr
}
