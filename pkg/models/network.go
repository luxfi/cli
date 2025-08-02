// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"fmt"

	"github.com/luxfi/cli/v2/v2/pkg/constants"
	lux_constants "github.com/luxfi/node/v2/v2/utils/constants"
)

type Network int64

const (
	Undefined Network = iota
	Mainnet
	Testnet
	Local
	Devnet
)

// Kind returns the network kind (same as the network itself for now)
func (n Network) Kind() Network {
	return n
}

func (s Network) String() string {
	switch s {
	case Mainnet:
		return "Mainnet"
	case Testnet:
		return "Testnet"
	case Local:
		return "Local Network"
	case Devnet:
		return "Devnet"
	}
	return "Unknown Network"
}

func (s Network) NetworkID() (uint32, error) {
	switch s {
	case Mainnet:
		return lux_constants.MainnetID, nil
	case Testnet:
		return lux_constants.TestnetID, nil
	case Local:
		return constants.LocalNetworkID, nil
	case Devnet:
		return constants.LocalNetworkID, nil // Devnet uses same ID as local for now
	}
	return 0, fmt.Errorf("unsupported network")
}

// ID returns the network ID as a uint32
func (s Network) ID() uint32 {
	id, _ := s.NetworkID()
	return id
}

func NetworkFromString(s string) Network {
	switch s {
	case Mainnet.String():
		return Mainnet
	case Testnet.String():
		return Testnet
	case Local.String():
		return Local
	case Devnet.String():
		return Devnet
	}
	return Undefined
}

func NetworkFromNetworkID(networkID uint32) Network {
	switch networkID {
	case lux_constants.MainnetID:
		return Mainnet
	case lux_constants.TestnetID:
		return Testnet
	case constants.LocalNetworkID:
		return Local
	}
	return Undefined
}

// HandlePublicNetworkSimulation handles public network simulation logic
func (n Network) HandlePublicNetworkSimulation() {
	// This is a placeholder - implement simulation logic if needed
}

// Endpoint returns the network endpoint URL
func (n Network) Endpoint() string {
	switch n {
	case Mainnet:
		return constants.MainnetAPIEndpoint
	case Testnet:
		return constants.TestnetAPIEndpoint
	case Local:
		return constants.LocalAPIEndpoint
	case Devnet:
		return constants.LocalAPIEndpoint // Devnet uses local endpoint
	}
	return ""
}

// Name returns the network name (alias for String)
func (n Network) Name() string {
	return n.String()
}

// NewLocalNetwork creates a new Local network
func NewLocalNetwork() Network {
	return Local
}
