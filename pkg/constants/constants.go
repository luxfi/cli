// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package constants

import (
	"time"
)

const (
	DefaultPerms755    = 0o755
	WriteReadReadPerms = 0o644

	BaseDirName = ".cli"
	LogDir      = "logs"

	ServerRunFile = "gRPCserver.run"
	LuxCliBinDir  = "bin"
	RunDir        = "runs"

	SuffixSeparator             = "_"
	SidecarFileName             = "sidecar.json"
	GenesisFileName             = "genesis.json"
	ElasticSubnetConfigFileName = "elastic_subnet_config.json"
	NodeConfigJSONFile          = "node-config.json"
	SidecarSuffix               = SuffixSeparator + SidecarFileName
	GenesisSuffix               = SuffixSeparator + GenesisFileName
	NodeFileName                = "node.json"

	SidecarVersion = "1.4.0"

	MaxLogFileSize   = 4
	MaxNumOfLogFiles = 5
	RetainOldFiles   = 0 // retain all old log files

	RequestTimeout         = 3 * time.Minute
	E2ERequestTimeout      = 30 * time.Second
	ANRRequestTimeout      = 3 * time.Minute
	APIRequestTimeout      = 30 * time.Second
	APIRequestLargeTimeout = 2 * time.Minute

	SimulatePublicNetwork = "SIMULATE_PUBLIC_NETWORK"
	TestnetAPIEndpoint    = "https://api.lux-test.network"
	MainnetAPIEndpoint    = "https://api.lux.network"

	// WebSocket endpoints
	MainnetWSEndpoint = "wss://api.lux.network/ext/bc/C/ws"
	TestnetWSEndpoint = "wss://api.lux-test.network/ext/bc/C/ws"
	LocalWSEndpoint   = "ws://127.0.0.1:9630/ext/bc/C/ws"
	DevnetWSEndpoint  = "wss://api.lux-dev.network/ext/bc/C/ws"

	// Cloud service constants
	GCPCloudService            = "gcp"
	AWSCloudService            = "aws"
	E2EDocker                  = "e2e-docker"
	GCPNodeAnsiblePrefix       = "gcp_node"
	AWSNodeAnsiblePrefix       = "aws_node"
	E2EDockerLoopbackHost      = "127.0.0.1"
	GCPDefaultImageProvider    = "canonical"
	GCPImageFilter             = "ubuntu-os-cloud"
	CloudNodeCLIConfigBasePath = "/home/ubuntu/.cli"
	CodespaceNameEnvVar        = "CODESPACE_NAME"
	AnsibleSSHShellParams      = "-o StrictHostKeyChecking=no"
	RemoteSSHUser              = "ubuntu"
	StakerCertFileName         = "staker.crt"
	StakerKeyFileName          = "staker.key"
	BLSKeyFileName             = "bls.key"
	ValidatorUptimeDeductible  = 5 * time.Minute

	// SSH constants
	SSHSleepBetweenChecks = 1 * time.Second
	SSHFileOpsTimeout     = 10 * time.Second
	SSHScriptTimeout      = 120 * time.Second
	SSHPOSTTimeout        = 30 * time.Second
	SSHDirOpsTimeout      = 30 * time.Second
	
	// Docker constants
	DockerNodeConfigPath   = "/data/.luxgo/configs"
	WriteReadUserOnlyPerms = 0o600
	
	// AWS constants  
	AWSCloudServerRunningState = "running"

	// this depends on bootstrap snapshot
	LocalAPIEndpoint = "http://127.0.0.1:9630"
	DevnetAPIEndpoint = "https://api.lux-dev.network"
	LocalNetworkID   = 1337

	DefaultTokenName = "TEST"
	
	// Default versions
	DefaultLuxdVersion = "v1.13.4"
	
	// Staking constants
	BootstrapValidatorBalanceNanoLUX = 1_000_000_000_000 // 1000 LUX
	PoSL1MinimumStakeDurationSeconds = 86400             // 24 hours
	
	// Logging
	DefaultAggregatorLogLevel = "INFO"
	
	// Git
	GitExtension = ".git"
	
	// Ansible
	AnsibleHostInventoryFileName = "hosts"
	AnsibleSSHUseAgentParams     = "-o ForwardAgent=yes"
	
	// Cloud node
	CloudNodeConfigPath = "/home/ubuntu/.luxgo/configs"
	CloudNodePrometheusConfigPath = "/home/ubuntu/.luxgo/configs/prometheus"
	CloudNodeStakingPath = "/home/ubuntu/.luxgo/staking"
	UpgradeFileName     = "upgrade.json"
	NodePrometheusConfigFileName = "prometheus.yml"
	ServicesDir = "services"
	WarpRelayerInstallDir = "warp-relayer"
	WarpRelayerConfigFilename = "warp-relayer.yml"
	
	// Config keys
	ConfigSnapshotsAutoSaveKey   = "SnapshotsAutoSaveEnabled"
	ConfigUpdatesDisabledKey     = "UpdatesDisabled"
	
	// Build environment
	BuildEnvGolangVersion = "1.24.5"
	
	// Docker images and repos
	LuxdDockerImage = "luxfi/luxd"
	LuxdGitRepo     = "https://github.com/luxfi/node"
	LuxdRepoName    = "luxfi/node"
	
	// Organizations
	LuxOrg = "luxfi"
	
	// Repo names
	LuxRepoName = "node"
	EVMRepoName = "evm"
	
	// Install directories
	LuxInstallDir   = "lux"
	LuxGoInstallDir = "luxgo"
	EVMInstallDir   = "evm"
	
	// Directories
	SubnetDir    = "subnets"
	ReposDir     = "repos"
	SnapshotsDirName = "snapshots"
	CustomVMDir  = "customvms"
	PluginDir    = "plugins"
	ConfigDir    = "config"
	KeyDir       = "keys"
	LPMPluginDir = "lpm-plugins"
	
	// Cloud node paths
	CloudNodeSubnetEvmBinaryPath = "/home/ubuntu/.cli/bin/subnet-evm"
	
	// File names
	UpgradeBytesFileName = "upgrade.json"
	LPMLogName          = "lpm.log"
	OldConfigFileName   = ".cli-config.json"
	OldMetricsConfigFileName = ".cli-metrics.json"
	ConfigLPMAdminAPIEndpointKey = "lpm-admin-api-endpoint"
	ConfigLPMCredentialsFileKey  = "lpm-credentials-file"
	
	// Devnet flags
	DevnetFlagsProposerVMUseCurrentHeight = true // This is a boolean flag
	
	// File names
	AliasesFileName = "aliases.json"
	
	// Directories
	DashboardsDir = "dashboards"
	
	// Grafana
	CustomGrafanaDashboardJSON = "custom_dashboard.json"
	
	// Config metrics keys
	ConfigMetricsUserIDKey         = "metrics-user-id"
	ConfigMetricsEnabledKey        = "metrics-enabled"
	ConfigAuthorizeCloudAccessKey  = "authorize-cloud-access"
	
	// Duplicate constants removed - these are already defined above
	
	// Environment variables
	MetricsAPITokenEnvVarName = "METRICS_API_TOKEN"

	HealthCheckInterval = 100 * time.Millisecond

	// it's unlikely anyone would want to name a snapshot `default`
	// but let's add some more entropy
	DefaultSnapshotName          = "default-1654102509"
	BootstrapSnapshotArchiveName = "bootstrapSnapshot.tar.gz"
	BootstrapSnapshotLocalPath   = "assets/" + BootstrapSnapshotArchiveName
	BootstrapSnapshotURL         = "https://github.com/luxfi/cli/raw/main/" + BootstrapSnapshotLocalPath
	BootstrapSnapshotSHA256URL   = "https://github.com/luxfi/cli/raw/main/assets/sha256sum.txt"

	CliInstallationURL    = "https://raw.githubusercontent.com/luxfi/cli/main/scripts/install.sh"
	ExpectedCliInstallErr = "resource temporarily unavailable"

	KeySuffix  = ".pk"
	YAMLSuffix = ".yml"

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

	CliRepoName = "cli"

	EVMBin = "evm"

	DefaultNodeRunURL = "http://127.0.0.1:9630"

	// Latest EVM version
	LatestEVMVersion = "v0.7.7"

	LPMDir = ".lpm"

	// Network ports
	SSHTCPPort      = 22
	LuxdAPIPort     = 9650
	LuxdGrafanaPort = 3000

	// Node roles
	APIRole         = "api"
	ValidatorRole   = "validator"
	
	// Cluster config
	ClustersConfigFileName = "clusters.json"
	MonitorRole     = "monitor"
	WarpRelayerRole = "warp-relayer"
	
	// Warp constants
	WarpDir     = "warp"
	WarpBranch  = "main"
	WarpURL     = "https://github.com/luxfi/warp.git"
	WarpKeyName = "warp"
	WarpVersion = "v1.0.0"
	
	// Relayer constants
	DefaultRelayerVersion = "v1.0.0"
	
	// Payment messages
	PayTxsFeesMsg = "pay transaction fees"
	
	// Units
	OneLux = 1_000_000_000 // 1 LUX = 1e9 nLUX
)

// HTTPAccess represents HTTP access configuration
type HTTPAccess string

const (
	// HTTPAccess values
	HTTPAccessPublic  HTTPAccess = "public"
	HTTPAccessPrivate HTTPAccess = "private"

	// SSH timeouts
	SSHLongRunningScriptTimeout = 10 * time.Minute
	DefaultLuxPackage           = "luxfi/plugins-core"

	// #nosec G101
	GithubAPITokenEnvVarName = "LUX_CLI_GITHUB_TOKEN"

	VMDir          = "vms"
	ChainConfigDir = "chains"

	SubnetType                 = "subnet type"
	SubnetConfigFileName       = "subnet.json"
	ChainConfigFileName        = "chain.json"
	PerNodeChainConfigFileName = "per-node-chain.json"

	GitRepoCommitName  = "Lux CLI"
	GitRepoCommitEmail = "info@lux.network"

	LuxMaintainers = "luxfi"

	UpgradeBytesLockExtension = ".lock"
	NotAvailableLabel         = "Not available"
	BackendCmd                = "cli-backend"

	LuxCompatibilityVersionAdded = "v1.9.2"
	LuxCompatibilityURL          = "https://raw.githubusercontent.com/luxfi/node/master/version/compatibility.json"
	LuxdCompatibilityURL         = LuxCompatibilityURL // Alias for backward compatibility
	EVMRPCCompatibilityURL       = "https://raw.githubusercontent.com/luxfi/evm/main/compatibility.json"
	CLIMinVersionURL             = "https://raw.githubusercontent.com/luxfi/cli/main/min-version.json"
	CLILatestDependencyURL       = CLIMinVersionURL // Alias for backward compatibility
	SubnetEVMRepoName            = EVMRepoName       // Alias for backward compatibility

	YesLabel = "Yes"
	NoLabel  = "No"

	// Default Warp Messenger Address
	DefaultWarpMessengerAddress = "0x0000000000000000000000000000000000000005"
	
	// C-Chain Warp Registry Addresses
	MainnetCChainWarpRegistryAddress = "0x0000000000000000000000000000000000000006"

	SubnetIDLabel     = "SubnetID: "
	BlockchainIDLabel = "BlockchainID: "

	Network        = "network"
	SkipUpdateFlag = "skip-update-check"
	LastFileName   = ".last_actions.json"

	DefaultWalletCreationTimeout = 5 * time.Second

	DefaultConfirmTxTimeout = 20 * time.Second

	// Cloud and network constants
	CloudOperationTimeout            = 5 * time.Minute
	LuxdP2PPort                      = 9651
	LuxdMonitoringPort               = 9090
	LuxdLokiPort                     = 23101
	GCPStaticIPPrefix                = "lux-"
	CloudServerStorageSize           = 100 // GB
	MonitoringCloudServerStorageSize = 200 // GB
	ErrReleasingGCPStaticIP          = "error releasing GCP static IP"
	IPAddressSuffix                  = "-ip"

	// Local network constants
	ExtraLocalNetworkDataFilename = "extra_local_network_data.json"
	LocalNetworkMetaFile           = "local_network_meta.json"
	FastGRPCDialTimeout            = 3 * time.Second
)
