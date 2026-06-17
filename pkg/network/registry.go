package network

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EnvVar is the colon-separated path of universe directories scanned
// for chain.yaml — same shape as $PATH.
const EnvVar = "LUX_NETWORK_PATH"

// DefaultPath is the workspace convention: ~/work/<name>/universe.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return strings.Join([]string{
		filepath.Join(home, "work", "lux", "universe"),
		filepath.Join(home, "work", "zoo", "universe"),
		filepath.Join(home, "work", "hanzo", "universe"),
		filepath.Join(home, "work", "pars", "universe"),
		filepath.Join(home, "work", "osage", "universe"),
		filepath.Join(home, "work", "adnexus", "universe"),
	}, ":")
}

// Registry holds networks discovered from $LUX_NETWORK_PATH.
type Registry struct {
	specs map[string]*Spec // slug → Spec
}

// Discover scans $LUX_NETWORK_PATH for chain.yaml files. Networks
// later in the path do NOT override earlier ones — first-wins, like
// $PATH. Unreadable / malformed files are skipped with a warning.
func Discover() (*Registry, error) {
	pathStr := os.Getenv(EnvVar)
	if pathStr == "" {
		pathStr = DefaultPath()
	}
	r := &Registry{specs: map[string]*Spec{}}
	for _, dir := range strings.Split(pathStr, ":") {
		if dir == "" {
			continue
		}
		yaml := filepath.Join(dir, "chain.yaml")
		st, err := os.Stat(yaml)
		if err != nil || st.IsDir() {
			continue
		}
		s, err := ParseFile(yaml)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", yaml, err)
			continue
		}
		name := s.Network.Slug
		if _, dup := r.specs[name]; !dup {
			r.specs[name] = s
		}
	}
	return r, nil
}

// Lookup returns the network spec by name slug.
func (r *Registry) Lookup(name string) (*Spec, error) {
	s, ok := r.specs[name]
	if !ok {
		known := r.Names()
		return nil, fmt.Errorf("network %q not discovered under $%s (known: %v)", name, EnvVar, known)
	}
	return s, nil
}

// Names returns all discovered network slugs, sorted.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.specs))
	for n := range r.specs {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Refs returns all (name/env) pairs in the registry, sorted.
func (r *Registry) Refs() []string {
	var out []string
	for _, s := range r.specs {
		for env := range s.Networks {
			out = append(out, s.Network.Slug+"/"+env)
		}
	}
	sort.Strings(out)
	return out
}
