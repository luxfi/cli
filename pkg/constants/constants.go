// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package constants

import (
	"time"
)

const (
	DefaultPerms755 = 0o755

	BaseDirName = ".cli"
	LogDir      = "logs"

	ServerRunFile      = "gRPCserver.run"
	LuxCliBinDir = "bin"
	RunDir             = "runs"

	SuffixSeparator             = "_"
	SidecarFileName             = "sidecar.json"
	GenesisFileName             = "genesis.json"
	ElasticSubnetConfigFileName = "elastic_subnet_config.json"
	SidecarSuffix               = SuffixSeparator + SidecarFileName
	GenesisSuffix               = SuffixSeparator + GenesisFileName
	NodeFileName                = "node.json"

	SidecarVersion = "1.4.0"

	MaxLogFileSize   = 4
	MaxNumOfLogFiles = 5
	RetainOldFiles   = 0 // retain all old log files

	RequestTimeout    = 3 * time.Minute
	E2ERequestTimeout = 30 * time.Second

	SimulatePublicNetwork = "SIMULATE_PUBLIC_NETWORK"
	FujiAPIEndpoint       = "https://api.lux-test.network"
	MainnetAPIEndpoint    = "https://api.lux.network"

	// this depends on bootstrap snapshot
	LocalAPIEndpoint = "http://127.0.0.1:9630"
	LocalNetworkID   = 1337

	DefaultTokenName = "TEST"

	HealthCheckInterval = 100 * time.Millisecond

	// it's unlikely anyone would want to name a snapshot `default`
	// but let's add some more entropy
	SnapshotsDirName             = "snapshots"
	DefaultSnapshotName          = "default-1654102509"
	BootstrapSnapshotArchiveName = "bootstrapSnapshot.tar.gz"
	BootstrapSnapshotLocalPath   = "assets/" + BootstrapSnapshotArchiveName
	BootstrapSnapshotURL         = "https://github.com/luxfi/cli/raw/main/" + BootstrapSnapshotLocalPath
	BootstrapSnapshotSHA256URL   = "https://github.com/luxfi/cli/raw/main/assets/sha256sum.txt"

	CliInstallationURL    = "https://raw.githubusercontent.com/luxfi/cli/main/scripts/install.sh"
	ExpectedCliInstallErr = "resource temporarily unavailable"

	KeyDir     = "key"
	KeySuffix  = ".pk"
	YAMLSuffix = ".yml"
	ConfigDir  = "config"

	Enable = "enable"

	Disable = "disable"

	TimeParseLayout    = "2006-01-02 15:04:05"
	MinStakeDuration   = 24 * 14 * time.Hour
	MaxStakeDuration   = 24 * 365 * time.Hour
	MaxStakeWeight     = 100
	MinStakeWeight     = 1
	DefaultStakeWeight = 20
	// The absolute minimum is 25 seconds, but set to 1 minute to allow for
	// time to go through the command
	StakingStartLeadTime   = 1 * time.Minute
	StakingMinimumLeadTime = 25 * time.Second

	DefaultConfigFileName = ".cli"
	DefaultConfigFileType = "json"

	CustomVMDir = "vms"

	AvaLabsOrg          = "luxfi"
	LuxRepoName = "node"
	SubnetEVMRepoName   = "subnet-evm"
	CliRepoName         = "cli"

	LuxInstallDir = "node"
	SubnetEVMInstallDir   = "subnet-evm"

	SubnetEVMBin = "subnet-evm"

	DefaultNodeRunURL = "http://127.0.0.1:9630"

	APMDir                = ".apm"
	APMLogName            = "apm.log"
	DefaultAvaLabsPackage = "luxfi/plugins-core"
	APMPluginDir          = "apm_plugins"

	// #nosec G101
	GithubAPITokenEnvVarName = "LUX_CLI_GITHUB_TOKEN"

	ReposDir       = "repos"
	SubnetDir      = "subnets"
	VMDir          = "vms"
	ChainConfigDir = "chains"

	SubnetType                 = "subnet type"
	SubnetConfigFileName       = "subnet.json"
	ChainConfigFileName        = "chain.json"
	PerNodeChainConfigFileName = "per-node-chain.json"

	GitRepoCommitName  = "Lux CLI"
	GitRepoCommitEmail = "info@lux.network"

	AvaLabsMaintainers = "luxfi"

	UpgradeBytesFileName      = "upgrade.json"
	UpgradeBytesLockExtension = ".lock"
	NotAvailableLabel         = "Not available"
	BackendCmd                = "cli-backend"

	LuxCompatibilityVersionAdded = "v1.9.2"
	LuxCompatibilityURL          = "https://raw.githubusercontent.com/luxfi/node/master/version/compatibility.json"
	SubnetEVMRPCCompatibilityURL         = "https://raw.githubusercontent.com/luxfi/subnet-evm/master/compatibility.json"

	YesLabel = "Yes"
	NoLabel  = "No"

	SubnetIDLabel     = "SubnetID: "
	BlockchainIDLabel = "BlockchainID: "

	PluginDir = "plugins"

	Network        = "network"
	SkipUpdateFlag = "skip-update-check"
	LastFileName   = ".last_actions.json"

	DefaultWalletCreationTimeout = 5 * time.Second

	DefaultConfirmTxTimeout = 20 * time.Second
)
