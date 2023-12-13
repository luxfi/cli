// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package utils

const (
	baseDir      = ".cli"
	hardhatDir   = "./tests/e2e/hardhat"
	confFilePath = hardhatDir + "/dynamic_conf.json"
	greeterFile  = hardhatDir + "/greeter.json"

	BaseTest               = "./test/index.ts"
	GreeterScript          = "./scripts/deploy.ts"
	GreeterCheck           = "./scripts/checkGreeting.ts"
	SoloSubnetEVMKey1      = "soloSubnetEVMVersion1"
	SoloSubnetEVMKey2      = "soloSubnetEVMVersion2"
	SoloLuxdKey           = "soloLuxdVersion"
	OnlyLuxdKey           = "onlyLuxdVersion"
	MultiLuxdSubnetEVMKey = "multiLuxdSubnetEVMVersion"
	MultiLuxd1Key         = "multiLuxdVersion1"
	MultiLuxd2Key         = "multiLuxdVersion2"
	LatestEVM2LuxdKey     = "latestEVM2Luxd"
	LatestLuxd2EVMKey     = "latestLuxd2EVM"
	OnlyLuxdValue         = "latest"

	SubnetEvmGenesisPath      = "tests/e2e/assets/test_subnet_evm_genesis.json"
	SubnetEvmGenesis2Path     = "tests/e2e/assets/test_subnet_evm_genesis_2.json"
	EwoqKeyPath               = "tests/e2e/assets/ewoq_key.pk"
	SubnetEvmAllowFeeRecpPath = "tests/e2e/assets/test_subnet_evm_allowFeeRecps_genesis.json"
	SubnetEvmGenesisBadPath   = "tests/e2e/assets/test_subnet_evm_genesis_bad.json"

	PluginDirExt = "plugins"

	ledgerSimDir         = "./tests/e2e/ledgerSim"
	basicLedgerSimScript = "./launchAndApproveTxs.ts"
)
