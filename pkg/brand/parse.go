package brand

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ParseFile loads and validates a brand's chain.yaml.
func ParseFile(path string) (*Brand, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", abs, err)
	}
	var b Brand
	if err := yaml.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse %s: %w", abs, err)
	}
	if b.Version != SchemaVersion {
		return nil, fmt.Errorf("%s: unsupported chain.yaml version %q (want %q)", abs, b.Version, SchemaVersion)
	}
	b.SourcePath = abs
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", abs, err)
	}
	return &b, nil
}

func (b *Brand) validate() error {
	if b.Network.Slug == "" {
		return fmt.Errorf("network.slug required")
	}
	if len(b.Networks) == 0 {
		return fmt.Errorf("networks block must list at least one env")
	}
	seenPort := map[int]string{}
	seenNID := map[uint32]string{}
	for env, n := range b.Networks {
		if n.NetworkID == 0 {
			return fmt.Errorf("networks.%s.networkID required (sovereign-L1 rule)", env)
		}
		if n.HTTPPort == 0 {
			return fmt.Errorf("networks.%s.httpPort required", env)
		}
		if n.StakingPort == 0 {
			return fmt.Errorf("networks.%s.stakingPort required", env)
		}
		if n.HTTPPort == n.StakingPort {
			return fmt.Errorf("networks.%s: httpPort and stakingPort collide on %d", env, n.HTTPPort)
		}
		if prev, dup := seenPort[n.HTTPPort]; dup {
			return fmt.Errorf("networks.%s.httpPort %d collides with %s", env, n.HTTPPort, prev)
		}
		if prev, dup := seenPort[n.StakingPort]; dup {
			return fmt.Errorf("networks.%s.stakingPort %d collides with %s", env, n.StakingPort, prev)
		}
		if prev, dup := seenNID[n.NetworkID]; dup {
			return fmt.Errorf("networks.%s.networkID %d collides with %s", env, n.NetworkID, prev)
		}
		seenPort[n.HTTPPort] = env + ".http"
		seenPort[n.StakingPort] = env + ".staking"
		seenNID[n.NetworkID] = env

		// Sovereign-L1 invariant: when both networkID and primaryEvmChainID
		// are set, they must match.
		if n.PrimaryEvmChainID != 0 && n.PrimaryEvmChainID != uint64(n.NetworkID) {
			return fmt.Errorf(
				"networks.%s: primaryEvmChainID %d != networkID %d (sovereign-L1 rule)",
				env, n.PrimaryEvmChainID, n.NetworkID,
			)
		}
	}
	return nil
}

// Env returns the per-env network block, error if env not declared.
func (b *Brand) Env(env string) (NetCfg, error) {
	n, ok := b.Networks[env]
	if !ok {
		envs := make([]string, 0, len(b.Networks))
		for e := range b.Networks {
			envs = append(envs, e)
		}
		return NetCfg{}, fmt.Errorf("network %s has no env %q (have: %v)", b.Network.Slug, env, envs)
	}
	return n, nil
}

// Dir returns the directory containing this brand's chain.yaml.
func (b *Brand) Dir() string { return filepath.Dir(b.SourcePath) }
