// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package netspec

// NetworkSpec defines a declarative network specification for IaC.
// This is version-controllable and idempotent.
type NetworkSpec struct {
	// APIVersion for schema compatibility
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`

	// Kind identifies this as a network specification
	Kind string `yaml:"kind" json:"kind"`

	// Network configuration
	Network NetworkConfig `yaml:"network" json:"network"`
}

// NetworkConfig defines the network-level configuration.
type NetworkConfig struct {
	// Name is the unique identifier for this network
	Name string `yaml:"name" json:"name"`

	// Nodes is the number of validator nodes to run
	Nodes uint32 `yaml:"nodes" json:"nodes"`

	// LuxdVersion specifies the node version (e.g., "v1.20.3", "latest")
	LuxdVersion string `yaml:"luxdVersion,omitempty" json:"luxdVersion,omitempty"`

	// Subnets defines the blockchains to deploy
	Subnets []SubnetSpec `yaml:"subnets,omitempty" json:"subnets,omitempty"`
}

// SubnetSpec defines a subnet/blockchain configuration.
type SubnetSpec struct {
	// Name is the blockchain name
	Name string `yaml:"name" json:"name"`

	// VM specifies the virtual machine type (subnet-evm, custom)
	VM string `yaml:"vm" json:"vm"`

	// VMVersion specifies the VM version (optional, defaults to latest)
	VMVersion string `yaml:"vmVersion,omitempty" json:"vmVersion,omitempty"`

	// Genesis path to genesis file (optional)
	Genesis string `yaml:"genesis,omitempty" json:"genesis,omitempty"`

	// Validators is the number of validators for this subnet
	Validators uint32 `yaml:"validators,omitempty" json:"validators,omitempty"`

	// ChainID for EVM-based chains (optional)
	ChainID uint64 `yaml:"chainId,omitempty" json:"chainId,omitempty"`

	// TokenSymbol for the native token (optional)
	TokenSymbol string `yaml:"tokenSymbol,omitempty" json:"tokenSymbol,omitempty"`

	// Sovereign indicates if this is an L1 (true) or L2/subnet (false)
	Sovereign bool `yaml:"sovereign,omitempty" json:"sovereign,omitempty"`

	// ValidatorManagement specifies proof-of-authority or proof-of-stake
	ValidatorManagement string `yaml:"validatorManagement,omitempty" json:"validatorManagement,omitempty"`

	// TestDefaults uses test-optimized settings
	TestDefaults bool `yaml:"testDefaults,omitempty" json:"testDefaults,omitempty"`

	// ProductionDefaults uses production-optimized settings
	ProductionDefaults bool `yaml:"productionDefaults,omitempty" json:"productionDefaults,omitempty"`
}

// NetworkState represents the current deployed state of a network.
// Used for diffing against desired state.
type NetworkState struct {
	// Name of the network
	Name string `yaml:"name" json:"name"`

	// Running indicates if the network is currently running
	Running bool `yaml:"running" json:"running"`

	// Nodes is the count of running nodes
	Nodes uint32 `yaml:"nodes" json:"nodes"`

	// LuxdVersion is the current node version
	LuxdVersion string `yaml:"luxdVersion,omitempty" json:"luxdVersion,omitempty"`

	// Subnets lists deployed blockchains
	Subnets []SubnetState `yaml:"subnets,omitempty" json:"subnets,omitempty"`
}

// SubnetState represents the deployed state of a subnet.
type SubnetState struct {
	// Name of the blockchain
	Name string `yaml:"name" json:"name"`

	// SubnetID is the deployed subnet ID
	SubnetID string `yaml:"subnetId,omitempty" json:"subnetId,omitempty"`

	// BlockchainID is the deployed blockchain ID
	BlockchainID string `yaml:"blockchainId,omitempty" json:"blockchainId,omitempty"`

	// VM type
	VM string `yaml:"vm" json:"vm"`

	// VMVersion currently running
	VMVersion string `yaml:"vmVersion,omitempty" json:"vmVersion,omitempty"`

	// ChainID for EVM chains
	ChainID uint64 `yaml:"chainId,omitempty" json:"chainId,omitempty"`

	// Deployed indicates if this subnet is deployed
	Deployed bool `yaml:"deployed" json:"deployed"`

	// RPCEndpoint for the blockchain
	RPCEndpoint string `yaml:"rpcEndpoint,omitempty" json:"rpcEndpoint,omitempty"`
}

// DiffResult represents differences between desired and current state.
type DiffResult struct {
	// NetworkChanges indicates if network-level changes are needed
	NetworkChanges bool `yaml:"networkChanges" json:"networkChanges"`

	// SubnetsToCreate lists subnets that need to be created
	SubnetsToCreate []SubnetSpec `yaml:"subnetsToCreate,omitempty" json:"subnetsToCreate,omitempty"`

	// SubnetsToUpdate lists subnets that need configuration updates
	SubnetsToUpdate []SubnetSpec `yaml:"subnetsToUpdate,omitempty" json:"subnetsToUpdate,omitempty"`

	// SubnetsToDelete lists subnets that should be removed
	SubnetsToDelete []string `yaml:"subnetsToDelete,omitempty" json:"subnetsToDelete,omitempty"`

	// NeedsRestart indicates if the network needs restart
	NeedsRestart bool `yaml:"needsRestart" json:"needsRestart"`

	// Summary provides a human-readable description
	Summary string `yaml:"summary" json:"summary"`
}

// ValidVMs lists supported VM types.
var ValidVMs = []string{"subnet-evm", "custom"}

// ValidValidatorManagement lists supported validator management types.
var ValidValidatorManagement = []string{"proof-of-authority", "proof-of-stake"}
