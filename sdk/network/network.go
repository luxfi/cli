// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"github.com/luxfi/node/v2/utils/constants"
)

type NetworkKind int64

const (
	Undefined NetworkKind = iota
	Mainnet
	Testnet
	Devnet
)

const (
	TestnetAPIEndpoint    = "https://api.lux-test.network"
	MainnetAPIEndpoint = "https://api.lux.network"
)

type Network struct {
	Kind     NetworkKind
	ID       uint32
	Endpoint string
}

var UndefinedNetwork = Network{}

func NetworkFromNetworkID(networkID uint32) Network {
	switch networkID {
	case constants.MainnetID:
		return MainnetNetwork()
	case constants.TestnetID:
		return TestnetNetwork()
	}
	return UndefinedNetwork
}

func NewNetwork(kind NetworkKind, id uint32, endpoint string) Network {
	return Network{
		Kind:     kind,
		ID:       id,
		Endpoint: endpoint,
	}
}

func TestnetNetwork() Network {
	return NewNetwork(Testnet, constants.TestnetID, TestnetAPIEndpoint)
}

func MainnetNetwork() Network {
	return NewNetwork(Mainnet, constants.MainnetID, MainnetAPIEndpoint)
}
