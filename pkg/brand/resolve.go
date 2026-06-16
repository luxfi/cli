package brand

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve maps "brand/env" to a fully-expanded RuntimeProfile via the
// registry. All paths absolute, all defaults applied.
func Resolve(ref string) (*RuntimeProfile, error) {
	slug, env, err := ParseRef(ref)
	if err != nil {
		return nil, err
	}
	reg, err := Discover()
	if err != nil {
		return nil, err
	}
	b, err := reg.Lookup(slug)
	if err != nil {
		return nil, err
	}
	return ResolveBrand(b, env)
}

// ResolveBrand resolves a profile given an already-parsed Brand. Useful
// for tests and when the caller already owns the Brand.
func ResolveBrand(b *Brand, env string) (*RuntimeProfile, error) {
	net, err := b.Env(env)
	if err != nil {
		return nil, err
	}
	p := &RuntimeProfile{
		Brand:       b.Network.Slug,
		Env:         env,
		NetworkID:   net.NetworkID,
		HTTPPort:    net.HTTPPort,
		StakingPort: net.StakingPort,
		RPCUrl:      net.RPCUrl,
		LocalRPCUrl: fmt.Sprintf("http://127.0.0.1:%d/ext/bc/C/rpc", net.HTTPPort),
		LogLevel:    "info",
	}
	if b.Runtime.LogLevel != "" {
		p.LogLevel = b.Runtime.LogLevel
	}

	// dataDirTemplate default: ~/.lux/{brand}-{env}
	tmpl := b.Runtime.DataDirTemplate
	if tmpl == "" {
		tmpl = "~/.lux/{brand}-{env}"
	}
	p.DataDir, err = expandPath(tmpl, p)
	if err != nil {
		return nil, fmt.Errorf("dataDirTemplate: %w", err)
	}
	p.LogDir = filepath.Join(p.DataDir, "logs")

	// snapshotDir default: ~/work/lux/snapshots
	sd := b.Runtime.SnapshotDir
	if sd == "" {
		sd = "~/work/lux/snapshots"
	}
	p.SnapshotDir, err = expandPath(sd, p)
	if err != nil {
		return nil, fmt.Errorf("snapshotDir: %w", err)
	}

	// snapshotNameTemplate default: {networkID}-{brand}-{env}-with-contracts
	snapTmpl := b.Runtime.SnapshotNameTemplate
	if snapTmpl == "" {
		snapTmpl = "{networkID}-{brand}-{env}-with-contracts"
	}
	p.SnapshotName = applyTokens(snapTmpl, p)

	// serviceLabelTemplate default: ai.lux.{brand}.{env}
	lbl := b.Runtime.ServiceLabelTemplate
	if lbl == "" {
		lbl = "ai.lux.{brand}.{env}"
	}
	p.ServiceLabel = applyTokens(lbl, p)

	// Genesis: relative to chain.yaml dir, absolute on resolution.
	if net.GenesisFile != "" {
		gp := net.GenesisFile
		if !filepath.IsAbs(gp) {
			gp = filepath.Join(b.Dir(), gp)
		}
		gp = filepath.Clean(gp)
		if _, err := os.Stat(gp); err == nil {
			p.GenesisFile = gp
		} else {
			// Allow missing genesis file (luxd embedded), but warn caller.
			p.GenesisFile = ""
		}
	}
	return p, nil
}

func expandPath(p string, prof *RuntimeProfile) (string, error) {
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

func applyTokens(s string, p *RuntimeProfile) string {
	r := strings.NewReplacer(
		"{brand}", p.Brand,
		"{env}", p.Env,
		"{networkID}", fmt.Sprintf("%d", p.NetworkID),
		"{httpPort}", fmt.Sprintf("%d", p.HTTPPort),
		"{stakingPort}", fmt.Sprintf("%d", p.StakingPort),
	)
	return r.Replace(s)
}
