// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"fmt"

	"github.com/luxfi/cli/pkg/constants"
	lux_constants "github.com/luxfi/node/utils/constants"
)

type Network int64

const (
	Undefined Network = iota
	Mainnet
	Testnet
	Local
)

func (s Network) String() string {
	switch s {
	case Mainnet:
		return "Mainnet"
	case Testnet:
		return "Testnet"
	case Local:
		return "Local Network"
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
