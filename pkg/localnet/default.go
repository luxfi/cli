// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/tests/fixture/tmpnet"
	"github.com/luxfi/node/utils/units"

	"golang.org/x/exp/maps"
)

type nodeConfig struct {
	Flags map[string]interface{} `json:"flags"`
}

type networkConfig struct {
	NodeConfigs []nodeConfig           `json:"nodeConfigs"`
	CommonFlags map[string]interface{} `json:"commonFlags"`
	Upgrade     string                 `json:"upgrade"`
}

//go:embed default.json
var defaultNetworkData []byte

// GetDefaultNetworkConf creates a default network configuration of [numNodes]
// compatible with TmpNet usage, where the first len(networkConf.NodeConfigs) /== 5/
// will have default local network NodeID/BLSInfo/Ports, and the remaining
// ones will be dynamically generated.
// It returns the local network's:
// - genesis
// - upgrade
// - common flags
// - node confs
func GetDefaultNetworkConf(numNodes uint32) (
	uint32,
	*genesis.UnparsedConfig,
	[]byte,
	map[string]interface{},
	[]*tmpnet.Node,
	error,
) {
	networkConf := networkConfig{}
	if err := json.Unmarshal(defaultNetworkData, &networkConf); err != nil {
		return 0, nil, nil, nil, nil, fmt.Errorf("failure unmarshaling default local network config: %w", err)
	}
	nodes := []*tmpnet.Node{}
	for i := range numNodes {
		node := tmpnet.NewNode()
		if int(i) < len(networkConf.NodeConfigs) {
			maps.Copy(node.Flags, networkConf.NodeConfigs[i].Flags)
		}
		if err := node.EnsureKeys(); err != nil {
			return 0, nil, nil, nil, nil, err
		}
		nodes = append(nodes, node)
	}
	// Use the CLI's secure local key system - generates on first use
	localKey, err := key.GetLocalPrivateKey()
	if err != nil {
		return 0, nil, nil, nil, nil, fmt.Errorf("failed to get local key: %w", err)
	}

	// Create genesis config directly using genesis package types
	unparsedGenesis, err := createTestGenesis(constants.LocalNetworkID, nodes, localKey)
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}
	return constants.LocalNetworkID, unparsedGenesis, []byte(networkConf.Upgrade), networkConf.CommonFlags, nodes, nil
}

// createTestGenesis creates a test genesis configuration
func createTestGenesis(networkID uint32, nodes []*tmpnet.Node, fundedKey *secp256k1.PrivateKey) (*genesis.UnparsedConfig, error) {
	startTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	config := &genesis.UnparsedConfig{
		NetworkID:                  networkID,
		StartTime:                  uint64(startTime.Unix()),
		InitialStakeDuration:       uint64((365 * 24 * time.Hour).Seconds()),
		InitialStakeDurationOffset: 0,
		Message:                    "LUX Test Genesis",
	}

	// Add allocations for funded key
	addr := fundedKey.Address()
	addrStr := addr.String()
	config.Allocations = append(config.Allocations, genesis.UnparsedAllocation{
		LUXAddr:       addrStr,
		InitialAmount: 300 * units.MegaLux,
	})
	config.InitialStakedFunds = append(config.InitialStakedFunds, addrStr)

	// Add initial stakers from nodes
	for _, node := range nodes {
		if node.NodeID != ids.EmptyNodeID {
			config.InitialStakers = append(config.InitialStakers, genesis.UnparsedStaker{
				NodeID:        node.NodeID,
				RewardAddress: addrStr,
				DelegationFee: 20000, // 2%
			})
		}
	}

	// Add basic C-Chain genesis
	config.CChainGenesis = getBasicCChainGenesis(networkID)

	return config, nil
}

// getBasicCChainGenesis returns a basic C-Chain genesis configuration
func getBasicCChainGenesis(networkID uint32) string {
	chainID := int64(networkID)

	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId":             chainID,
			"homesteadBlock":      0,
			"eip150Block":         0,
			"eip155Block":         0,
			"eip158Block":         0,
			"byzantiumBlock":      0,
			"constantinopleBlock": 0,
			"petersburgBlock":     0,
			"istanbulBlock":       0,
			"muirGlacierBlock":    0,
		},
		"nonce":      "0x0",
		"timestamp":  "0x0",
		"extraData":  "0x00",
		"gasLimit":   fmt.Sprintf("0x%x", 8000000),
		"difficulty": "0x1",
		"mixHash":    "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase":   "0x0000000000000000000000000000000000000000",
		"alloc":      map[string]interface{}{},
		"number":     "0x0",
		"gasUsed":    "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	}

	data, _ := json.Marshal(genesis)
	return string(data)
}
