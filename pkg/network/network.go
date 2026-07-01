// Package network resolves a (name, env) tuple to a runtime profile for
// booting, deploying to, snapshotting, and stopping a sovereign-L1
// node.
//
// The source of truth is each network's `chain.yaml` discovered under
// $LUX_NETWORK_PATH (default ~/work/lux/universe:~/work/zoo/universe:…).
// One typed schema, one parse path, one resolution rule — every CLI
// verb consumes the same Profile.
//
// Identity at runtime: a network instance is uniquely identified by
// (name, env). At the wire that maps to (NetworkID, HTTPPort). Stop
// probes HTTPPort and verifies NetworkID matches before signaling:
// identity by what-the-node-IS, not by what-PID-it-was.
package network

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const SchemaVersion = "1"

// Spec is the parsed chain.yaml for one network universe.
type Spec struct {
	Version     string             `yaml:"version"`
	Network     Meta               `yaml:"network"`
	Networks    map[string]Env     `yaml:"networks"`
	Chains      []Chain            `yaml:"chains"`
	Token       Token              `yaml:"token"`
	Brand       Branding           `yaml:"brand"`
	Runtime     Runtime            `yaml:"runtime"`
	Precompiles []map[string]any   `yaml:"precompiles"`

	// SourcePath is the absolute path to the chain.yaml file. All
	// relative paths in the file resolve against its directory.
	SourcePath string `yaml:"-"`
}

// Meta is the top-level `network:` block.
type Meta struct {
	Name        string `yaml:"name"`
	Slug        string `yaml:"slug"`
	Type        string `yaml:"type"`   // l1 | l2 | l3
	Parent      string `yaml:"parent"` // null for sovereign L1
	Validators  int    `yaml:"validators"`
	DBType      string `yaml:"dbType"`
	Compression string `yaml:"compression"`
}

// Env is one `networks.<env>:` block.
//
// NetworkID and PrimaryEvmChainID are DISTINCT identifiers:
//   - NetworkID (uint32) is the validator-wire ID — --network-id flag,
//     P-chain handshake, info.getNetworkID.
//   - PrimaryEvmChainID (uint64) is the C-chain EIP-155 ID — eth_chainId,
//     MetaMask, EIP-155 tx signing.
//
// Lux brand keeps them DELIBERATELY DISTINCT (NID 1 / EVM 96369).
// Sovereign-L1 brand forks (Zoo, Hanzo, Pars, Osage, Liquidity)
// collapse them to one ID per env per L1 by convention.
type Env struct {
	NetworkID         uint32 `yaml:"networkID"`
	PrimaryEvmChainID uint64 `yaml:"primaryEvmChainID"`
	HTTPPort          int    `yaml:"httpPort"`
	StakingPort       int    `yaml:"stakingPort"`
	GenesisFile       string `yaml:"genesisFile"` // relative to chain.yaml dir
	RPCUrl            string `yaml:"rpcUrl"`
	WSUrl             string `yaml:"wsUrl"`
	Explorer          string `yaml:"explorer"`
	Cluster           string `yaml:"cluster"`
	Namespace         string `yaml:"namespace"`
	ImageTag          string `yaml:"imageTag"`
}

// Chain is one entry in `chains:`.
type Chain struct {
	Name        string `yaml:"name"`
	Slug        string `yaml:"slug"`
	VMType      string `yaml:"vmType"`
	Description string `yaml:"description"`
	RPCPath     string `yaml:"rpcPath"`
	WSPath      string `yaml:"wsPath"`
	GenesisFile string `yaml:"genesisFile"`
}

type Token struct {
	Name     string `yaml:"name"`
	Symbol   string `yaml:"symbol"`
	Decimals int    `yaml:"decimals"`
}

// Branding is the UI-facing `brand:` block (display name, colors,
// domains). Distinct from package name — kept because the YAML key is
// historical.
type Branding struct {
	DisplayName string         `yaml:"displayName"`
	LegalEntity string         `yaml:"legalEntity"`
	Domains     map[string]any `yaml:"domains"`
}

// Runtime declares the templates the CLI expands. Path values support
// {name}, {env}, {networkID}, {primaryEvmChainID}, {httpPort},
// {stakingPort} substitution.
type Runtime struct {
	DataDirTemplate      string `yaml:"dataDirTemplate"`
	SnapshotDir          string `yaml:"snapshotDir"`
	SnapshotNameTemplate string `yaml:"snapshotNameTemplate"`
	ServiceLabelTemplate string `yaml:"serviceLabelTemplate"`
	LogLevel             string `yaml:"logLevel"`
}

// Profile is what every verb consumes. Resolved from a Spec + env
// string by Resolve(); all paths absolute, all defaults applied.
type Profile struct {
	Name              string // network name slug (zoo, lux, hanzo…)
	Env               string // mainnet | testnet | devnet | localnet
	NetworkID         uint32 // validator wire / --network-id
	PrimaryEvmChainID uint64 // C-chain EIP-155 / eth_chainId
	HTTPPort          int
	StakingPort       int
	DataDir           string // absolute
	LogDir            string // absolute (always <dataDir>/logs)
	GenesisFile       string // absolute, may be empty (use luxd embedded)
	RPCUrl            string // remote (production) RPC if applicable
	LocalRPCUrl       string // http://127.0.0.1:<httpPort>/v1/bc/C/rpc
	SnapshotDir       string // absolute
	SnapshotName      string // expanded
	ServiceLabel      string // expanded, e.g. ai.lux.zoo.devnet
	LogLevel          string
}

// String prints "name/env" — the canonical identity used in CLI args.
func (p Profile) String() string { return p.Name + "/" + p.Env }

// Lock is the on-disk invariant manifest in <DataDir>/chain.lock.
// First-boot writes it; subsequent boots reject mismatches.
type Lock struct {
	Version           string `json:"version"`
	Name              string `json:"name"`
	Env               string `json:"env"`
	NetworkID         uint32 `json:"networkID"`
	PrimaryEvmChainID uint64 `json:"primaryEvmChainID"`
	HTTPPort          int    `json:"httpPort"`
	StakingPort       int    `json:"stakingPort"`
	GenesisHash       string `json:"genesisHash"`
	CreatedAt         string `json:"createdAt"`
}

// ParseRef splits "name/env" into its parts. Empty env or extra
// slashes are errors.
func ParseRef(ref string) (string, string, error) {
	parts := strings.Split(ref, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("ref %q must be <name>/<env> (e.g. zoo/devnet)", ref)
	}
	return parts[0], parts[1], nil
}

// HashGenesis returns the lowercased hex sha256 of the genesis blob —
// the value that goes in chain.lock to bind a data-dir to its genesis.
func HashGenesis(data []byte) string {
	sum := sha256.Sum256(data)
	return "0x" + hex.EncodeToString(sum[:])
}
