// Copyright (C) 2025, Lux Industries Inc. All rights reserved
// See the file LICENSE for licensing terms.

package relayer

// Temporary config types until warp imports are fixed
// TODO: Replace with actual warp imports

type VM int

const (
	EVM VM = iota
)

func (v VM) String() string {
	return "evm"
}

type MessageFormat int

const (
	TELEPORTER MessageFormat = iota
	OFF_CHAIN_REGISTRY
)

func (m MessageFormat) String() string {
	switch m {
	case TELEPORTER:
		return "teleporter"
	case OFF_CHAIN_REGISTRY:
		return "off-chain-registry"
	default:
		return "unknown"
	}
}

type Config struct {
	LogLevel                        string
	PChainAPI                       *APIConfig
	InfoAPI                         *APIConfig
	StorageLocation                 string
	ProcessMissedBlocks             bool
	SourceBlockchains               []*SourceBlockchain
	DestinationBlockchains          []*DestinationBlockchain
	MetricsPort                     uint16
	DBWriteIntervalSeconds          int
	SignatureCacheSize              int
	AllowPrivateIPs                 bool
	InitialConnectionTimeoutSeconds int
}

type APIConfig struct {
	BaseURL     string
	QueryParams map[string]string
}

type SourceBlockchain struct {
	SubnetID         string
	BlockchainID     string
	VM               string
	RPCEndpoint      APIConfig
	WSEndpoint       APIConfig
	MessageContracts map[string]MessageProtocolConfig
}

type DestinationBlockchain struct {
	SubnetID          string
	BlockchainID      string
	VM                string
	RPCEndpoint       APIConfig
	AccountPrivateKey string
}

type MessageProtocolConfig struct {
	MessageFormat string
	Settings      map[string]interface{}
}

// Temporary offchain registry address
var OffChainRegistrySourceAddress = struct {
	Hex func() string
}{
	Hex: func() string {
		return "0x0000000000000000000000000000000000000000"
	},
}