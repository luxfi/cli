// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

const (
	baseDir      = ".cli"
	hardhatDir   = "./tests/e2e/hardhat"
	confFilePath = hardhatDir + "/dynamic_conf.json"
	greeterFile  = hardhatDir + "/greeter.json"

	// Test blockchain and node names
	BlockchainName    = "test-blockchain"
	TestLocalNodeName = "test-local-node"

	// LPM related constants
	LPMDir       = ".lpm"
	LPMPluginDir = "plugins"

	BaseTest          = "./test/index.ts"
	GreeterScript     = "./scripts/deploy.ts"
	GreeterCheck      = "./scripts/checkGreeting.ts"
	SoloEVMKey1       = "soloEVMVersion1"
	SoloEVMKey2       = "soloEVMVersion2"
	SoloLuxKey        = "soloLuxVersion"
	SoloSubnetEVMKey1 = "soloSubnetEVMVersion1"
	SoloSubnetEVMKey2 = "soloSubnetEVMVersion2"
	SoloLuxdKey       = "soloLuxdVersion"
	OnlyLuxKey        = "onlyLuxVersion"
	MultiLuxEVMKey    = "multiLuxEVMVersion"
	MultiLux1Key      = "multiLuxVersion1"
	MultiLux2Key      = "multiLuxVersion2"
	LatestEVM2LuxKey  = "latestEVM2Lux"
	LatestLux2EVMKey  = "latestLux2EVM"
	OnlyLuxValue      = "latest"

	SubnetEvmGenesisPath      = "tests/e2e/assets/test_subnet_evm_genesis.json"
	SubnetEvmGenesis2Path     = "tests/e2e/assets/test_subnet_evm_genesis_2.json"
	EwoqKeyPath               = "tests/e2e/assets/ewoq_key.pk"
	SubnetEvmAllowFeeRecpPath = "tests/e2e/assets/test_subnet_evm_allowFeeRecps_genesis.json"
	SubnetEvmGenesisBadPath   = "tests/e2e/assets/test_subnet_evm_genesis_bad.json"

	PluginDirExt = "plugins"
)
