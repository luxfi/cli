package network

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ParseFile loads and validates a network's chain.yaml.
func ParseFile(path string) (*Spec, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", abs, err)
	}
	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", abs, err)
	}
	if s.Version != SchemaVersion {
		return nil, fmt.Errorf("%s: unsupported chain.yaml version %q (want %q)", abs, s.Version, SchemaVersion)
	}
	s.SourcePath = abs
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", abs, err)
	}
	return &s, nil
}

func (s *Spec) validate() error {
	if s.Network.Slug == "" {
		return fmt.Errorf("network.slug required")
	}
	if len(s.Networks) == 0 {
		return fmt.Errorf("networks block must list at least one env")
	}
	seenPort := map[int]string{}
	seenNID := map[uint32]string{}
	for env, e := range s.Networks {
		if e.NetworkID == 0 {
			return fmt.Errorf("networks.%s.networkID required", env)
		}
		if e.HTTPPort == 0 {
			return fmt.Errorf("networks.%s.httpPort required", env)
		}
		if e.StakingPort == 0 {
			return fmt.Errorf("networks.%s.stakingPort required", env)
		}
		if e.HTTPPort == e.StakingPort {
			return fmt.Errorf("networks.%s: httpPort and stakingPort collide on %d", env, e.HTTPPort)
		}
		if prev, dup := seenPort[e.HTTPPort]; dup {
			return fmt.Errorf("networks.%s.httpPort %d collides with %s", env, e.HTTPPort, prev)
		}
		if prev, dup := seenPort[e.StakingPort]; dup {
			return fmt.Errorf("networks.%s.stakingPort %d collides with %s", env, e.StakingPort, prev)
		}
		if prev, dup := seenNID[e.NetworkID]; dup {
			return fmt.Errorf("networks.%s.networkID %d collides with %s", env, e.NetworkID, prev)
		}
		seenPort[e.HTTPPort] = env + ".http"
		seenPort[e.StakingPort] = env + ".staking"
		seenNID[e.NetworkID] = env

		// NetworkID and PrimaryEvmChainID are DISTINCT identifiers. Lux
		// keeps them apart by design (NID 1 / EVM 96369). Sovereign-L1
		// brand forks collapse them by convention. No equality check;
		// the CLI surfaces both.
	}
	return nil
}

// At returns the per-env block, error if env not declared.
func (s *Spec) At(env string) (Env, error) {
	e, ok := s.Networks[env]
	if !ok {
		envs := make([]string, 0, len(s.Networks))
		for k := range s.Networks {
			envs = append(envs, k)
		}
		return Env{}, fmt.Errorf("network %s has no env %q (have: %v)", s.Network.Slug, env, envs)
	}
	return e, nil
}

// Dir returns the directory containing this network's chain.yaml.
func (s *Spec) Dir() string { return filepath.Dir(s.SourcePath) }
