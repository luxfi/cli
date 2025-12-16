// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ammcmd

import "github.com/luxfi/geth/common"

// NetworkConfig holds AMM contract addresses for a specific network
type NetworkConfig struct {
	ChainID     int64
	RPC         string
	Name        string
	V2Factory   common.Address
	V2Router    common.Address
	V3Factory   common.Address
	V3Router    common.Address
	Multicall   common.Address
	Quoter      common.Address
	WETH        common.Address
	NFTPosition common.Address
	TickLens    common.Address
}

// Predefined network configurations
var (
	// Lux Mainnet (C-Chain)
	LuxMainnet = NetworkConfig{
		ChainID:     96369,
		RPC:         "http://localhost:8546",
		Name:        "Lux Mainnet",
		V2Factory:   common.HexToAddress("0xD173926A10A0C4eCd3A51B1422270b65Df0551c1"),
		V2Router:    common.HexToAddress("0xAe2cf1E403aAFE6C05A5b8Ef63EB19ba591d8511"),
		V3Factory:   common.HexToAddress("0x80bBc7C4C7a59C899D1B37BC14539A22D5830a84"),
		V3Router:    common.HexToAddress("0x939bC0Bca6F9B9c52E6e3AD8A3C590b5d9B9D10E"),
		Multicall:   common.HexToAddress("0xd25F88CBdAe3c2CCA3Bb75FC4E723b44C0Ea362F"),
		Quoter:      common.HexToAddress("0x12e2B76FaF4dDA5a173a4532916bb6Bfa3645275"),
		WETH:        common.HexToAddress("0x4888E4a2Ee0F03051c72D2BD3ACf755eD3498B3E"), // WLUX
		NFTPosition: common.HexToAddress("0x7a4C48B9dae0b7c396569b34042fcA604150Ee28"),
		TickLens:    common.HexToAddress("0x57A22965AdA0e52D785A9Aa155beF423D573b879"),
	}

	// Zoo Mainnet
	ZooMainnet = NetworkConfig{
		ChainID:     200200,
		RPC:         "http://localhost:8545",
		Name:        "Zoo Mainnet",
		V2Factory:   common.HexToAddress("0xD173926A10A0C4eCd3A51B1422270b65Df0551c1"),
		V2Router:    common.HexToAddress("0xAe2cf1E403aAFE6C05A5b8Ef63EB19ba591d8511"),
		V3Factory:   common.HexToAddress("0x80bBc7C4C7a59C899D1B37BC14539A22D5830a84"),
		V3Router:    common.HexToAddress("0x939bC0Bca6F9B9c52E6e3AD8A3C590b5d9B9D10E"),
		Multicall:   common.HexToAddress("0xd25F88CBdAe3c2CCA3Bb75FC4E723b44C0Ea362F"),
		Quoter:      common.HexToAddress("0x12e2B76FaF4dDA5a173a4532916bb6Bfa3645275"),
		WETH:        common.HexToAddress("0x4888E4a2Ee0F03051c72D2BD3ACf755eD3498B3E"), // WZOO
		NFTPosition: common.HexToAddress("0x7a4C48B9dae0b7c396569b34042fcA604150Ee28"),
		TickLens:    common.HexToAddress("0x57A22965AdA0e52D785A9Aa155beF423D573b879"),
	}

	// Lux Testnet
	LuxTestnet = NetworkConfig{
		ChainID:     96368,
		RPC:         "http://localhost:8547",
		Name:        "Lux Testnet",
		V2Factory:   common.HexToAddress("0xD173926A10A0C4eCd3A51B1422270b65Df0551c1"),
		V2Router:    common.HexToAddress("0xAe2cf1E403aAFE6C05A5b8Ef63EB19ba591d8511"),
		V3Factory:   common.HexToAddress("0x80bBc7C4C7a59C899D1B37BC14539A22D5830a84"),
		V3Router:    common.HexToAddress("0x939bC0Bca6F9B9c52E6e3AD8A3C590b5d9B9D10E"),
		Multicall:   common.HexToAddress("0xd25F88CBdAe3c2CCA3Bb75FC4E723b44C0Ea362F"),
		Quoter:      common.HexToAddress("0x12e2B76FaF4dDA5a173a4532916bb6Bfa3645275"),
		WETH:        common.HexToAddress("0x4888E4a2Ee0F03051c72D2BD3ACf755eD3498B3E"), // WLUX
		NFTPosition: common.HexToAddress("0x7a4C48B9dae0b7c396569b34042fcA604150Ee28"),
		TickLens:    common.HexToAddress("0x57A22965AdA0e52D785A9Aa155beF423D573b879"),
	}

	// Zoo chain tokens (chainId 200200)
	ZooTokens = map[string]common.Address{
		"WZOO":  common.HexToAddress("0x4888E4a2Ee0F03051c72D2BD3ACf755eD3498B3E"),
		"ZETH":  common.HexToAddress("0x60E0a8167FC13dE89348978860466C9ceC24B9ba"),
		"ZBTC":  common.HexToAddress("0x1E48D32a4F5e9f08DB9aE4959163300FaF8A6C8e"),
		"ZUSD":  common.HexToAddress("0x848Cff46eb323f323b6Bbe1Df274E40793d7f2c2"),
		"ZLUX":  common.HexToAddress("0x5E5290f350352768bD2bfC59c2DA15DD04A7cB88"),
		"ZSOL":  common.HexToAddress("0x26B40f650156C7EbF9e087Dd0dca181Fe87625B7"),
		"ZBNB":  common.HexToAddress("0x6EdcF3645DeF09DB45050638c41157D8B9FEa1cf"),
		"ZPOL":  common.HexToAddress("0x28BfC5DD4B7E15659e41190983e5fE3df1132bB9"),
		"ZCELO": common.HexToAddress("0x3078847F879A33994cDa2Ec1540ca52b5E0eE2e5"),
		"ZFTM":  common.HexToAddress("0x8B982132d639527E8a0eAAD385f97719af8f5e04"),
		"ZTON":  common.HexToAddress("0x3141b94b89691009b950c96e97Bff48e0C543E3C"),
	}

	// Lux chain tokens (chainId 96369)
	LuxTokens = map[string]common.Address{
		"WLUX": common.HexToAddress("0x4888E4a2Ee0F03051c72D2BD3ACf755eD3498B3E"),
		"LETH": common.HexToAddress("0x60E0a8167FC13dE89348978860466C9ceC24B9ba"),
		"LBTC": common.HexToAddress("0x1E48D32a4F5e9f08DB9aE4959163300FaF8A6C8e"),
		"LUSD": common.HexToAddress("0x848Cff46eb323f323b6Bbe1Df274E40793d7f2c2"),
		"LZOO": common.HexToAddress("0x5E5290f350352768bD2bfC59c2DA15DD04A7cB88"),
		"LSOL": common.HexToAddress("0x26B40f650156C7EbF9e087Dd0dca181Fe87625B7"),
		"LBNB": common.HexToAddress("0x6EdcF3645DeF09DB45050638c41157D8B9FEa1cf"),
		"LPOL": common.HexToAddress("0x28BfC5DD4B7E15659e41190983e5fE3df1132bB9"),
	}

	// Network lookup by chain ID
	Networks = map[int64]*NetworkConfig{
		96369:  &LuxMainnet,
		200200: &ZooMainnet,
		96368:  &LuxTestnet,
	}

	// Network lookup by name
	NetworksByName = map[string]*NetworkConfig{
		"lux":         &LuxMainnet,
		"lux-mainnet": &LuxMainnet,
		"zoo":         &ZooMainnet,
		"zoo-mainnet": &ZooMainnet,
		"lux-testnet": &LuxTestnet,
		"testnet":     &LuxTestnet,
	}
)

// GetNetwork returns network config by name or chain ID
func GetNetwork(nameOrID string) *NetworkConfig {
	if cfg, ok := NetworksByName[nameOrID]; ok {
		return cfg
	}
	return nil
}

// GetNetworkByChainID returns network config by chain ID
func GetNetworkByChainID(chainID int64) *NetworkConfig {
	return Networks[chainID]
}
