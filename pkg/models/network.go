// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	lux_constants "github.com/luxfi/node/utils/constants"
)

type Network int64

const (
	Undefined Network = iota
	Mainnet
	Testnet
	Local
	Devnet
)

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
	}
	return 0, fmt.Errorf("unsupported network")
}

func (s Network) NetworkIDFlagValue() string {
	id, err := s.NetworkID()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", id)
}

func (s Network) ID() uint32 {
	id, _ := s.NetworkID()
	return id
}

func (s Network) Kind() Network {
	return s
}

// Name returns the network name as a string
func (s Network) Name() string {
	return s.String()
}

func (s Network) HandlePublicNetworkSimulation() bool {
	return s == Local
}

func NetworkFromString(s string) Network {
	switch s {
	case Mainnet.String():
		return Mainnet
	case Testnet.String():
		return Testnet
	case Local.String():
		return Local
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

// NewLocalNetwork creates a new Local network instance
func NewLocalNetwork() Network {
	return Local
}

// BootstrappingContext returns a context for bootstrapping operations
func (s Network) BootstrappingContext() (context.Context, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	return ctx, cancel
}
