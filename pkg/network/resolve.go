package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve maps "name/env" to a fully-expanded Profile via the
// registry. All paths absolute, all defaults applied.
func Resolve(ref string) (*Profile, error) {
	name, env, err := ParseRef(ref)
	if err != nil {
		return nil, err
	}
	reg, err := Discover()
	if err != nil {
		return nil, err
	}
	s, err := reg.Lookup(name)
	if err != nil {
		return nil, err
	}
	return ResolveSpec(s, env)
}

// ResolveSpec resolves a Profile given an already-parsed Spec.
func ResolveSpec(s *Spec, env string) (*Profile, error) {
	e, err := s.At(env)
	if err != nil {
		return nil, err
	}
	p := &Profile{
		Name:              s.Network.Slug,
		Env:               env,
		NetworkID:         e.NetworkID,
		PrimaryEvmChainID: e.PrimaryEvmChainID,
		HTTPPort:          e.HTTPPort,
		StakingPort:       e.StakingPort,
		RPCUrl:            e.RPCUrl,
		LocalRPCUrl:       fmt.Sprintf("http://127.0.0.1:%d/ext/bc/C/rpc", e.HTTPPort),
		LogLevel:          "info",
	}
	if s.Runtime.LogLevel != "" {
		p.LogLevel = s.Runtime.LogLevel
	}

	// dataDirTemplate default: ~/.lux/{name}-{env}
	tmpl := s.Runtime.DataDirTemplate
	if tmpl == "" {
		tmpl = "~/.lux/{name}-{env}"
	}
	p.DataDir, err = expandPath(tmpl, p)
	if err != nil {
		return nil, fmt.Errorf("dataDirTemplate: %w", err)
	}
	p.LogDir = filepath.Join(p.DataDir, "logs")

	// snapshotDir default: ~/work/lux/snapshots
	sd := s.Runtime.SnapshotDir
	if sd == "" {
		sd = "~/work/lux/snapshots"
	}
	p.SnapshotDir, err = expandPath(sd, p)
	if err != nil {
		return nil, fmt.Errorf("snapshotDir: %w", err)
	}

	// snapshotNameTemplate default: {networkID}-{name}-{env}-with-contracts
	snapTmpl := s.Runtime.SnapshotNameTemplate
	if snapTmpl == "" {
		snapTmpl = "{networkID}-{name}-{env}-with-contracts"
	}
	p.SnapshotName = applyTokens(snapTmpl, p)

	// serviceLabelTemplate default: ai.lux.{name}.{env}
	lbl := s.Runtime.ServiceLabelTemplate
	if lbl == "" {
		lbl = "ai.lux.{name}.{env}"
	}
	p.ServiceLabel = applyTokens(lbl, p)

	if e.GenesisFile != "" {
		gp := e.GenesisFile
		if !filepath.IsAbs(gp) {
			gp = filepath.Join(s.Dir(), gp)
		}
		gp = filepath.Clean(gp)
		if _, err := os.Stat(gp); err == nil {
			p.GenesisFile = gp
		} else {
			p.GenesisFile = ""
		}
	}
	return p, nil
}

func expandPath(p string, prof *Profile) (string, error) {
	expanded := applyTokens(p, prof)
	if strings.HasPrefix(expanded, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		expanded = filepath.Join(home, expanded[1:])
	}
	return filepath.Clean(expanded), nil
}

func applyTokens(s string, p *Profile) string {
	r := strings.NewReplacer(
		"{name}", p.Name,
		"{env}", p.Env,
		"{networkID}", fmt.Sprintf("%d", p.NetworkID),
		"{primaryEvmChainID}", fmt.Sprintf("%d", p.PrimaryEvmChainID),
		"{httpPort}", fmt.Sprintf("%d", p.HTTPPort),
		"{stakingPort}", fmt.Sprintf("%d", p.StakingPort),
	)
	return r.Replace(s)
}
