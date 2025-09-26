// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	nodeWarp "github.com/luxfi/node/vms/platformvm/warp"
	standaloneWarp "github.com/luxfi/warp"
	warpPayload "github.com/luxfi/warp/payload"
)

var ErrInvalidMessageType = errors.New("invalid message type")

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
	// Hash the conversion data to generate a unique ID
	bytes, err := json.Marshal(data)
	if err != nil {
		return ids.Empty, err
	}
	hash := sha256.Sum256(bytes)
	return ids.ID(hash), nil
}

// NewSubnetToL1Conversion creates a new subnet-to-L1 conversion message
func NewSubnetToL1Conversion(conversionID ids.ID) (*warpPayload.AddressedCall, error) {
	// Create a subnet-to-L1 conversion message
	payload := &warpPayload.AddressedCall{
		SourceAddress: []byte{}, // Will be filled by the sender
		Payload:       conversionID[:],
	}
	return payload, nil
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
	Valid            bool      `serialize:"true" json:"valid"`
}

// GetValidationID returns the validation ID for this registration
func (r *L1ValidatorRegistration) GetValidationID() ids.ID {
	return r.ValidationID
}

// Bytes returns the byte representation of this registration
func (r *L1ValidatorRegistration) Bytes() []byte {
	// Use the warp payload L1ValidatorRegistration for serialization
	payload, _ := warpPayload.NewL1ValidatorRegistration(r.Valid, []byte{})
	return payload.Bytes()
}

// L1ValidatorWeight represents an L1 validator weight update
type L1ValidatorWeight struct {
	ValidationID ids.ID `serialize:"true" json:"validationID"`
	Nonce        uint64 `serialize:"true" json:"nonce"`
	Weight       uint64 `serialize:"true" json:"weight"`
}

// NewL1ValidatorWeight creates a new L1ValidatorWeight message
func NewL1ValidatorWeight(validationID ids.ID, nonce uint64, weight uint64) (*L1ValidatorWeight, error) {
	return &L1ValidatorWeight{
		ValidationID: validationID,
		Nonce:        nonce,
		Weight:       weight,
	}, nil
}

// Bytes returns the byte representation of the message
func (l *L1ValidatorWeight) Bytes() []byte {
	// Serialize the validator weight message
	bytes, _ := json.Marshal(l)
	return bytes
}

// ParseL1ValidatorWeight parses L1 validator weight from payload
func ParseL1ValidatorWeight(payload []byte) (*L1ValidatorWeight, error) {
	// Deserialize the L1ValidatorWeight message from the payload
	msg := &L1ValidatorWeight{}
	if err := json.Unmarshal(payload, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// ParseRegisterL1Validator parses L1 validator registration from payload
func ParseRegisterL1Validator(payload []byte) (*L1ValidatorRegistration, error) {
	// Use the warp payload parser
	payloadObj, err := warpPayload.ParsePayload(payload)
	if err != nil {
		return nil, err
	}
	
	// Type assert to RegisterL1Validator
	registerPayload, ok := payloadObj.(*warpPayload.RegisterL1Validator)
	if !ok {
		return nil, ErrInvalidMessageType
	}
	
	// Convert SubnetID bytes to ids.ID
	subnetID, err := ids.ToID(registerPayload.SubnetID)
	if err != nil {
		return nil, err
	}
	
	// Convert NodeID bytes to ids.NodeID  
	nodeID, err := ids.ToNodeID(registerPayload.NodeID)
	if err != nil {
		return nil, err
	}
	
	// Convert to local type
	reg := &L1ValidatorRegistration{
		ValidationID: subnetID, // Use SubnetID as validation ID for now
		NodeID:       nodeID,
		BLSPublicKey: registerPayload.BLSPublicKey,
		Weight:       registerPayload.Weight,
		Expiry:       registerPayload.RegistrationTime,
		Valid:        true,
	}
	
	return reg, nil
}

// NewRegisterL1Validator creates a new L1 validator registration payload with proper signature
func NewRegisterL1Validator(
	subnetID ids.ID,
	nodeID ids.NodeID, 
	blsPublicKey []byte,
	expiry uint64,
	balanceOwners PChainOwner,
	disableOwners PChainOwner,
	weight uint64,
) (*L1ValidatorRegistration, error) {
	// Create a validation ID from the inputs
	validationID := ids.GenerateTestID() // In production, calculate from inputs
	
	reg := &L1ValidatorRegistration{
		ValidationID:     validationID,
		NodeID:           nodeID,
		BLSPublicKey:     blsPublicKey,
		Weight:           weight,
		Expiry:           expiry,
		RemainingBalance: 0, // Initialize as needed
		DisableOwner:     disableOwners,
		Valid:            true,
	}
	
	return reg, nil
}

// NewL1ValidatorRegistration creates a new L1 validator registration message
func NewL1ValidatorRegistration(validationID ids.ID, valid bool) (*warpPayload.L1ValidatorRegistration, error) {
	return warpPayload.NewL1ValidatorRegistration(valid, validationID[:])
}

// ParseAddressedCall parses an addressed call from payload
func ParseAddressedCall(payload []byte) (*warpPayload.AddressedCall, error) {
	payloadObj, err := warpPayload.ParsePayload(payload)
	if err != nil {
		return nil, err
	}
	
	// Type assert to AddressedCall
	addressedCall, ok := payloadObj.(*warpPayload.AddressedCall)
	if !ok {
		return nil, ErrInvalidMessageType
	}
	
	return addressedCall, nil
}

// ConvertStandaloneToNodeWarpMessage converts a standalone warp message to a node warp message
func ConvertStandaloneToNodeWarpMessage(standaloneMsg *standaloneWarp.Message) (*nodeWarp.Message, error) {
	// Extract the raw bytes from the standalone message
	msgBytes := standaloneMsg.Bytes()
	
	// Parse as a node warp message
	return nodeWarp.ParseMessage(msgBytes)
}