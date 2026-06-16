// Package brand resolves a (brand, env) tuple to a runtime profile for
// booting, deploying to, snapshotting, and stopping a sovereign-L1 node.
//
// The source of truth is each brand's `chain.yaml` discovered under
// $LUX_BRAND_PATH (default ~/work/lux/universe:~/work/zoo/universe:...).
// One typed schema, one parse path, one resolution rule — every CLI
// verb consumes the same RuntimeProfile.
//
// Identity: a sovereign L1 is uniquely identified by (brand, env). At
// runtime, this maps to a unique (networkID, httpPort) pair — networkID
// is the network's name on the wire, httpPort is where it speaks
// locally. Stop probes httpPort and verifies networkID matches before
// signaling: identity by what-the-node-IS, not by what-PID-it-was.
package brand

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const SchemaVersion = "1"

// Brand is the parsed chain.yaml for one brand universe.
type Brand struct {
	Version   string             `yaml:"version"`
	Network   NetworkBlock       `yaml:"network"`
	Networks  map[string]NetCfg  `yaml:"networks"`
	Chains    []ChainCfg         `yaml:"chains"`
	Token     TokenCfg           `yaml:"token"`
	BrandInfo BrandMeta          `yaml:"brand"`
	Runtime   RuntimeBlock       `yaml:"runtime"`
	Precompiles []map[string]any `yaml:"precompiles"`

	// SourcePath is the absolute path to the chain.yaml file. All
	// relative paths in the file resolve against its directory.
	SourcePath string `yaml:"-"`
}

// NetworkBlock is the top-level `network:` stanza.
type NetworkBlock struct {
	Name        string `yaml:"name"`
	Slug        string `yaml:"slug"`
	Type        string `yaml:"type"`   // l1 | l2 | l3
	Parent      string `yaml:"parent"` // null for sovereign L1
	Validators  int    `yaml:"validators"`
	DBType      string `yaml:"dbType"`
	Compression string `yaml:"compression"`
}

// NetCfg is one `networks.<env>:` block.
type NetCfg struct {
	NetworkID   uint32 `yaml:"networkID"`
	HTTPPort    int    `yaml:"httpPort"`
	StakingPort int    `yaml:"stakingPort"`
	GenesisFile string `yaml:"genesisFile"` // relative to chain.yaml dir
	RPCUrl      string `yaml:"rpcUrl"`
	WSUrl       string `yaml:"wsUrl"`
	Explorer    string `yaml:"explorer"`
	Cluster     string `yaml:"cluster"`
	Namespace   string `yaml:"namespace"`
	ImageTag    string `yaml:"imageTag"`

	// EVM-side numbers, optional. The sovereign-L1 rule is
	// networkID == primaryEvmChainID — when set explicitly, must match.
	PrimaryEvmChainID uint64 `yaml:"primaryEvmChainID"`
}

// ChainCfg is one entry in `chains:`.
type ChainCfg struct {
	Name        string `yaml:"name"`
	Slug        string `yaml:"slug"`
	VMType      string `yaml:"vmType"`
	Description string `yaml:"description"`
	RPCPath     string `yaml:"rpcPath"`
	WSPath      string `yaml:"wsPath"`
	GenesisFile string `yaml:"genesisFile"`
}

type TokenCfg struct {
	Name     string `yaml:"name"`
	Symbol   string `yaml:"symbol"`
	Decimals int    `yaml:"decimals"`
}

type BrandMeta struct {
	DisplayName string         `yaml:"displayName"`
	LegalEntity string         `yaml:"legalEntity"`
	Domains     map[string]any `yaml:"domains"`
}

// RuntimeBlock declares the templates the CLI expands. All path values
// support {brand}, {env}, {networkID} substitution.
type RuntimeBlock struct {
	DataDirTemplate       string `yaml:"dataDirTemplate"`
	SnapshotDir           string `yaml:"snapshotDir"`
	SnapshotNameTemplate  string `yaml:"snapshotNameTemplate"`
	ServiceLabelTemplate  string `yaml:"serviceLabelTemplate"`
	LogLevel              string `yaml:"logLevel"`
}

// RuntimeProfile is what every verb consumes. Resolved from a Brand +
// env string by Resolve(); all paths absolute, all defaults applied.
type RuntimeProfile struct {
	Brand        string
	Env          string
	NetworkID    uint32
	HTTPPort     int
	StakingPort  int
	DataDir      string // absolute
	LogDir       string // absolute (always <dataDir>/logs)
	GenesisFile  string // absolute, may be empty (use luxd embedded)
	RPCUrl       string // remote (production) RPC if applicable
	LocalRPCUrl  string // http://127.0.0.1:<httpPort>/ext/bc/C/rpc
	SnapshotDir  string // absolute
	SnapshotName string // expanded
	ServiceLabel string // expanded, e.g. ai.lux.zoo.devnet
	LogLevel     string
}

// String prints "brand/env" identity used in CLI args.
func (p RuntimeProfile) String() string { return p.Brand + "/" + p.Env }

// ChainLock is the on-disk invariant manifest in <DataDir>/chain.lock.
// First-boot writes it; subsequent boots reject mismatches.
type ChainLock struct {
	Version     string `json:"version"`
	Brand       string `json:"brand"`
	Env         string `json:"env"`
	NetworkID   uint32 `json:"networkID"`
	HTTPPort    int    `json:"httpPort"`
	StakingPort int    `json:"stakingPort"`
	GenesisHash string `json:"genesisHash"`
	CreatedAt   string `json:"createdAt"`
}

// ParseRef splits "network/env" into its parts. Empty env or extra
// slashes are errors.
func ParseRef(ref string) (string, string, error) {
	parts := strings.Split(ref, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("network ref %q must be <network>/<env> (e.g. zoo/devnet)", ref)
	}
	return parts[0], parts[1], nil
}

// HashGenesis returns the lowercased hex sha256 of the genesis blob —
// the value that goes in chain.lock to bind a data-dir to its genesis.
func HashGenesis(data []byte) string {
	sum := sha256.Sum256(data)
	return "0x" + hex.EncodeToString(sum[:])
}
