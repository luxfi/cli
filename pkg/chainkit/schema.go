// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chainkit provides the chain.yaml schema, parser, validator,
// and CRD generator for the `lux chain launch` command.
//
// A single chain.yaml file drives the entire ecosystem deployment:
// nodes, indexer, explorer, gateway, exchange, wallet, faucet.
// The generator produces Kubernetes CRDs that the lux-operator reconciles.
package chainkit

// ChainConfig is the top-level schema for chain.yaml.
// One file per chain project (e.g. ~/work/pars/chain.yaml).
type ChainConfig struct {
	// Version of the chain.yaml schema (currently "1")
	Version string `yaml:"version" json:"version"`

	// Chain identity and consensus
	Chain ChainSpec `yaml:"chain" json:"chain"`

	// Per-network configuration (mainnet, testnet, devnet)
	Networks map[string]NetworkSpec `yaml:"networks" json:"networks"`

	// Native token configuration
	Token TokenSpec `yaml:"token" json:"token"`

	// Genesis configuration
	Genesis GenesisSpec `yaml:"genesis" json:"genesis"`

	// Branding for explorer, exchange, wallet UIs
	Brand BrandSpec `yaml:"brand" json:"brand"`

	// Services to deploy (each maps to a CRD or K8s Deployment)
	Services ServicesSpec `yaml:"services" json:"services"`

	// Deployment target configuration
	Deploy DeploySpec `yaml:"deploy" json:"deploy"`

	// Precompile activation configuration
	Precompiles []PrecompileSpec `yaml:"precompiles,omitempty" json:"precompiles,omitempty"`
}

// ChainSpec defines the chain's identity and consensus parameters.
type ChainSpec struct {
	// Human-readable name (e.g. "Pars Network")
	Name string `yaml:"name" json:"name"`

	// Lowercase slug used in K8s namespaces, domains, labels (e.g. "pars")
	Slug string `yaml:"slug" json:"slug"`

	// Chain layer: l1, l2, l3
	Type string `yaml:"type" json:"type"`

	// Sequencer type: lux, ethereum, op, external
	Sequencer string `yaml:"sequencer" json:"sequencer"`

	// VM type: evm, pars, custom
	VM string `yaml:"vm" json:"vm"`

	// Additional VM plugins (e.g. sessionvm for Pars)
	VMPlugins []VMPluginSpec `yaml:"vmPlugins,omitempty" json:"vmPlugins,omitempty"`

	// Database type: zapdb, pebbledb, memdb
	DBType string `yaml:"dbType,omitempty" json:"dbType,omitempty"`

	// Network compression: zstd, none
	Compression string `yaml:"compression,omitempty" json:"compression,omitempty"`
}

// VMPluginSpec defines an additional VM plugin to deploy alongside the main VM.
type VMPluginSpec struct {
	Name  string `yaml:"name" json:"name"`   // Plugin name (e.g. "sessionvm")
	VMID  string `yaml:"vmId" json:"vmId"`   // VM ID on the P-chain
	Image string `yaml:"image" json:"image"` // Container image with the plugin binary
}

// NetworkSpec defines per-network (mainnet/testnet/devnet) configuration.
type NetworkSpec struct {
	// Lux network ID (1=mainnet, 2=testnet, 3=devnet, or custom)
	NetworkID uint32 `yaml:"networkId" json:"networkId"`

	// EVM chain ID
	ChainID uint64 `yaml:"chainId" json:"chainId"`

	// P-chain blockchain ID (assigned after deployment, or preconfigured)
	BlockchainID string `yaml:"blockchainId,omitempty" json:"blockchainId,omitempty"`

	// RPC endpoint
	RPCUrl string `yaml:"rpcUrl,omitempty" json:"rpcUrl,omitempty"`

	// WebSocket endpoint
	WSUrl string `yaml:"wsUrl,omitempty" json:"wsUrl,omitempty"`

	// Number of validator nodes
	Validators uint32 `yaml:"validators" json:"validators"`

	// Node image tag override
	ImageTag string `yaml:"imageTag,omitempty" json:"imageTag,omitempty"`

	// Bootstrap nodes (host:port)
	BootstrapNodes []string `yaml:"bootstrapNodes,omitempty" json:"bootstrapNodes,omitempty"`

	// Seed restore URL (S3 snapshot tarball)
	SeedRestoreURL string `yaml:"seedRestoreUrl,omitempty" json:"seedRestoreUrl,omitempty"`

	// Snapshot schedule interval (seconds, 0=disabled)
	SnapshotInterval uint64 `yaml:"snapshotInterval,omitempty" json:"snapshotInterval,omitempty"`

	// Sybil protection (disable for devnet)
	SybilProtection *bool `yaml:"sybilProtection,omitempty" json:"sybilProtection,omitempty"`
}

// TokenSpec defines the native token.
type TokenSpec struct {
	Name     string `yaml:"name" json:"name"`
	Symbol   string `yaml:"symbol" json:"symbol"`
	Decimals uint8  `yaml:"decimals" json:"decimals"`
	Supply   string `yaml:"supply,omitempty" json:"supply,omitempty"`
}

// GenesisSpec defines genesis configuration.
type GenesisSpec struct {
	// Path to genesis JSON file (relative to chain.yaml)
	File string `yaml:"file,omitempty" json:"file,omitempty"`

	// Inline genesis JSON (alternative to file)
	Inline map[string]interface{} `yaml:"inline,omitempty" json:"inline,omitempty"`

	// Airdrop configuration
	AirdropAddress string `yaml:"airdropAddress,omitempty" json:"airdropAddress,omitempty"`
	AirdropAmount  string `yaml:"airdropAmount,omitempty" json:"airdropAmount,omitempty"`

	// Gas configuration
	GasLimit   uint64 `yaml:"gasLimit,omitempty" json:"gasLimit,omitempty"`
	MinBaseFee uint64 `yaml:"minBaseFee,omitempty" json:"minBaseFee,omitempty"`
	BlockRate  uint64 `yaml:"blockRate,omitempty" json:"blockRate,omitempty"` // seconds
}

// BrandSpec defines UI branding across all services.
type BrandSpec struct {
	DisplayName  string     `yaml:"displayName" json:"displayName"`
	LegalEntity  string     `yaml:"legalEntity,omitempty" json:"legalEntity,omitempty"`
	PrimaryColor string     `yaml:"primaryColor,omitempty" json:"primaryColor,omitempty"`
	Logo         string     `yaml:"logo,omitempty" json:"logo,omitempty"`
	Favicon      string     `yaml:"favicon,omitempty" json:"favicon,omitempty"`
	Domains      DomainSpec `yaml:"domains" json:"domains"`
	Social       SocialSpec `yaml:"social,omitempty" json:"social,omitempty"`
}

// DomainSpec defines per-service domain names.
type DomainSpec struct {
	Explorer string `yaml:"explorer,omitempty" json:"explorer,omitempty"`
	Exchange string `yaml:"exchange,omitempty" json:"exchange,omitempty"`
	Wallet   string `yaml:"wallet,omitempty" json:"wallet,omitempty"`
	Faucet   string `yaml:"faucet,omitempty" json:"faucet,omitempty"`
	RPC      string `yaml:"rpc,omitempty" json:"rpc,omitempty"`
	Docs     string `yaml:"docs,omitempty" json:"docs,omitempty"`
}

// SocialSpec defines social media links.
type SocialSpec struct {
	Twitter string `yaml:"twitter,omitempty" json:"twitter,omitempty"`
	Discord string `yaml:"discord,omitempty" json:"discord,omitempty"`
	GitHub  string `yaml:"github,omitempty" json:"github,omitempty"`
}

// ServicesSpec controls which services to deploy.
type ServicesSpec struct {
	Node     NodeServiceSpec     `yaml:"node" json:"node"`
	Indexer  IndexerServiceSpec  `yaml:"indexer" json:"indexer"`
	Explorer ExplorerServiceSpec `yaml:"explorer" json:"explorer"`
	Gateway  GatewayServiceSpec  `yaml:"gateway" json:"gateway"`
	Exchange ExchangeServiceSpec `yaml:"exchange" json:"exchange"`
	Wallet   WalletServiceSpec   `yaml:"wallet" json:"wallet"`
	Faucet   FaucetServiceSpec   `yaml:"faucet" json:"faucet"`
}

// NodeServiceSpec configures the validator node fleet.
type NodeServiceSpec struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Image    string `yaml:"image,omitempty" json:"image,omitempty"` // default: ghcr.io/luxfi/node
	ImageTag string `yaml:"imageTag,omitempty" json:"imageTag,omitempty"`

	// Storage
	StorageSize  string `yaml:"storageSize,omitempty" json:"storageSize,omitempty"`   // default: 100Gi
	StorageClass string `yaml:"storageClass,omitempty" json:"storageClass,omitempty"` // default: do-block-storage

	// Resources
	CPURequest    string `yaml:"cpuRequest,omitempty" json:"cpuRequest,omitempty"`
	CPULimit      string `yaml:"cpuLimit,omitempty" json:"cpuLimit,omitempty"`
	MemoryRequest string `yaml:"memoryRequest,omitempty" json:"memoryRequest,omitempty"`
	MemoryLimit   string `yaml:"memoryLimit,omitempty" json:"memoryLimit,omitempty"`

	// Staking key source (KMS reference)
	StakingKMS *StakingKMSSpec `yaml:"stakingKms,omitempty" json:"stakingKms,omitempty"`

	// MPC configuration
	MPC *MPCSpec `yaml:"mpc,omitempty" json:"mpc,omitempty"`
}

// StakingKMSSpec configures staking key retrieval from Hanzo KMS.
type StakingKMSSpec struct {
	HostAPI     string `yaml:"hostApi" json:"hostApi"`
	ProjectSlug string `yaml:"projectSlug" json:"projectSlug"`
	EnvSlug     string `yaml:"envSlug" json:"envSlug"`
	SecretsPath string `yaml:"secretsPath" json:"secretsPath"`
}

// MPCSpec configures MPC key management.
type MPCSpec struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`
}

// IndexerServiceSpec configures the Blockscout indexer.
type IndexerServiceSpec struct {
	Enabled             bool   `yaml:"enabled" json:"enabled"`
	Image               string `yaml:"image,omitempty" json:"image,omitempty"`
	ImageTag            string `yaml:"imageTag,omitempty" json:"imageTag,omitempty"`
	Replicas            int32  `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	DBStorageSize       string `yaml:"dbStorageSize,omitempty" json:"dbStorageSize,omitempty"` // default: 20Gi
	TraceEnabled        bool   `yaml:"traceEnabled,omitempty" json:"traceEnabled,omitempty"`
	ContractVerification bool  `yaml:"contractVerification,omitempty" json:"contractVerification,omitempty"`
	PollInterval        int    `yaml:"pollInterval,omitempty" json:"pollInterval,omitempty"` // seconds
}

// ExplorerServiceSpec configures the Blockscout frontend.
type ExplorerServiceSpec struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	Image        string `yaml:"image,omitempty" json:"image,omitempty"` // default: ghcr.io/luxfi/explore
	Replicas     int32  `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	IngressClass string `yaml:"ingressClass,omitempty" json:"ingressClass,omitempty"` // default: hanzo
}

// GatewayServiceSpec configures the API gateway.
type GatewayServiceSpec struct {
	Enabled          bool   `yaml:"enabled" json:"enabled"`
	Replicas         int32  `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	RateLimitRPS     int    `yaml:"rateLimitRps,omitempty" json:"rateLimitRps,omitempty"`
	RateLimitBurst   int    `yaml:"rateLimitBurst,omitempty" json:"rateLimitBurst,omitempty"`
	CORSAllowOrigins string `yaml:"corsAllowOrigins,omitempty" json:"corsAllowOrigins,omitempty"` // comma-separated
}

// ExchangeServiceSpec configures the DEX frontend.
type ExchangeServiceSpec struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	Image        string `yaml:"image,omitempty" json:"image,omitempty"`
	BrandPackage string `yaml:"brandPackage,omitempty" json:"brandPackage,omitempty"` // e.g. "@parsdao/brand"
}

// WalletServiceSpec configures the wallet deployment.
type WalletServiceSpec struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`
	Image     string   `yaml:"image,omitempty" json:"image,omitempty"`
	Platforms []string `yaml:"platforms,omitempty" json:"platforms,omitempty"` // web, extension, ios, android
}

// FaucetServiceSpec configures the testnet/devnet faucet.
type FaucetServiceSpec struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	DripAmount string `yaml:"dripAmount,omitempty" json:"dripAmount,omitempty"` // in wei
	RateLimit  string `yaml:"rateLimit,omitempty" json:"rateLimit,omitempty"`   // e.g. "1/hour"
}

// DeploySpec defines the deployment target.
type DeploySpec struct {
	// Platform: hanzo, k8s, docker
	Platform string `yaml:"platform" json:"platform"`

	// K8s namespace template (e.g. "pars-{network}")
	Namespace string `yaml:"namespace" json:"namespace"`

	// Container registry (e.g. "ghcr.io/luxfi")
	Registry string `yaml:"registry" json:"registry"`

	// Secrets provider: kms.hanzo.ai
	SecretsProvider string `yaml:"secretsProvider" json:"secretsProvider"`

	// Ingress class: hanzo (NEVER nginx/caddy)
	IngressClass string `yaml:"ingressClass" json:"ingressClass"`
}

// PrecompileSpec defines a precompile activation.
type PrecompileSpec struct {
	Name           string `yaml:"name" json:"name"`
	BlockTimestamp int64  `yaml:"blockTimestamp" json:"blockTimestamp"` // 0 = genesis
}
