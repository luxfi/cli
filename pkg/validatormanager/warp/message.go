// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	warpPayload "github.com/luxfi/warp/payload"
)

// PChainOwner represents an owner on the P-Chain
type PChainOwner struct {
	Threshold uint32        `serialize:"true" json:"threshold"`
	Addresses []ids.ShortID `serialize:"true" json:"addresses"`
}

// SubnetToL1ConversionValidatorData contains validator information for subnet-to-L1 conversion
type SubnetToL1ConversionValidatorData struct {
	NodeID       []byte                 `serialize:"true" json:"nodeID"`
	BLSPublicKey [bls.PublicKeyLen]byte `serialize:"true" json:"blsPublicKey"`
	Weight       uint64                 `serialize:"true" json:"weight"`
}

// SubnetToL1ConversionData contains the full subnet-to-L1 conversion payload
type SubnetToL1ConversionData struct {
	SubnetID       ids.ID                               `serialize:"true" json:"subnetID"`
	ManagerChainID ids.ID                               `serialize:"true" json:"managerChainID"`
	ManagerAddress []byte                               `serialize:"true" json:"managerAddress"`
	Validators     []SubnetToL1ConversionValidatorData `serialize:"true" json:"validators"`
}

// SubnetToL1ConversionID calculates the ID for a subnet-to-L1 conversion
func SubnetToL1ConversionID(data SubnetToL1ConversionData) (ids.ID, error) {
	// TODO: Implement proper hashing of the conversion data
	return ids.GenerateTestID(), nil
}

// NewSubnetToL1Conversion creates a new subnet-to-L1 conversion message
func NewSubnetToL1Conversion(conversionID ids.ID) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper message creation
	return &warpPayload.AddressedCall{}, nil
}

// L1ValidatorRegistration represents an L1 validator registration
type L1ValidatorRegistration struct {
	ValidationID     ids.ID    `serialize:"true" json:"validationID"`
	NodeID           ids.NodeID `serialize:"true" json:"nodeID"`
	BLSPublicKey     []byte    `serialize:"true" json:"blsPublicKey"`
	Weight           uint64    `serialize:"true" json:"weight"`
	Expiry           uint64    `serialize:"true" json:"expiry"`
	RemainingBalance uint64    `serialize:"true" json:"remainingBalance"`
	DisableOwner     PChainOwner `serialize:"true" json:"disableOwner"`
}

// L1ValidatorWeight represents an L1 validator weight update
type L1ValidatorWeight struct {
	ValidationID ids.ID `serialize:"true" json:"validationID"`
	Nonce        uint64 `serialize:"true" json:"nonce"`
	Weight       uint64 `serialize:"true" json:"weight"`
}

// ParseL1ValidatorWeight parses L1 validator weight from payload
func ParseL1ValidatorWeight(payload []byte) (*L1ValidatorWeight, error) {
	// TODO: Implement proper parsing
	return &L1ValidatorWeight{}, nil
}

// ParseRegisterL1Validator parses L1 validator registration from payload
func ParseRegisterL1Validator(payload []byte) (*L1ValidatorRegistration, error) {
	// TODO: Implement proper parsing
	return &L1ValidatorRegistration{}, nil
}

// NewRegisterL1Validator creates a new L1 validator registration payload
func NewRegisterL1Validator(subnetID ids.ID, nodeID ids.NodeID, weight uint64, blsPublicKey []byte, expiry uint64) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper message creation
	return &warpPayload.AddressedCall{}, nil
}

// NewL1ValidatorRegistration creates a new L1 validator registration message
func NewL1ValidatorRegistration(validationID ids.ID, valid bool) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper message creation
	return &warpPayload.AddressedCall{}, nil
}

// ParseAddressedCall parses an addressed call from payload
func ParseAddressedCall(payload []byte) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper parsing
	return &warpPayload.AddressedCall{}, nil
}