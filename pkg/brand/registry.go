package brand

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

// DefaultPath is the workspace convention: ~/work/<brand>/universe.
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

// Registry holds brands discovered from $LUX_BRAND_PATH.
type Registry struct {
	brands map[string]*Brand // slug → Brand
}

// Discover scans $LUX_NETWORK_PATH for chain.yaml files. Networks later
// in the path do NOT override earlier ones — first-wins, like $PATH.
// Unreadable / malformed chain.yaml files are skipped with a warning.
func Discover() (*Registry, error) {
	pathStr := os.Getenv(EnvVar)
	if pathStr == "" {
		pathStr = DefaultPath()
	}
	r := &Registry{brands: map[string]*Brand{}}
	for _, dir := range strings.Split(pathStr, ":") {
		if dir == "" {
			continue
		}
		yaml := filepath.Join(dir, "chain.yaml")
		st, err := os.Stat(yaml)
		if err != nil || st.IsDir() {
			continue
		}
		b, err := ParseFile(yaml)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", yaml, err)
			continue
		}
		slug := b.Network.Slug
		if _, dup := r.brands[slug]; !dup {
			r.brands[slug] = b
		}
	}
	return r, nil
}

// Lookup returns the network by slug; nil error means found.
func (r *Registry) Lookup(slug string) (*Brand, error) {
	b, ok := r.brands[slug]
	if !ok {
		known := r.Slugs()
		return nil, fmt.Errorf("network %q not discovered under $%s (known: %v)", slug, EnvVar, known)
	}
	return b, nil
}

// Slugs returns all discovered brand slugs, sorted.
func (r *Registry) Slugs() []string {
	out := make([]string, 0, len(r.brands))
	for s := range r.brands {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// Refs returns all (brand/env) pairs in the registry, sorted.
func (r *Registry) Refs() []string {
	var out []string
	for _, b := range r.brands {
		for env := range b.Networks {
			out = append(out, b.Network.Slug+"/"+env)
		}
	}
	sort.Strings(out)
	return out
}
