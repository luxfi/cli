// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package devcmd

import (
	"path/filepath"
	"testing"

	"github.com/luxfi/constants"
)

func TestHasBuildTag(t *testing.T) {
	cases := []struct {
		tags string
		tag  string
		want bool
	}{
		{"", "dchain", false},
		{"dchain", "dchain", true},
		{" dchain ", "dchain", true},          // surrounding space trimmed
		{"foo,dchain", "dchain", true},        // multi-tag
		{"dchain,foo", "dchain", true},        // multi-tag, first
		{"foo, dchain , bar", "dchain", true}, // spaced multi-tag
		{"dchains", "dchain", false},          // no substring match
		{"foo,bar", "dchain", false},
	}
	for _, c := range cases {
		if got := hasBuildTag(c.tags, c.tag); got != c.want {
			t.Errorf("hasBuildTag(%q, %q) = %v, want %v", c.tags, c.tag, got, c.want)
		}
	}
}

func TestResolvePluginDir(t *testing.T) {
	const base = "/home/u/.lux"
	want := filepath.Join(base, constants.PluginsDir, "current")

	// Explicit always wins, dchain or not.
	if got := resolvePluginDir("/custom/plugins", base, true); got != "/custom/plugins" {
		t.Errorf("explicit+dchain = %q, want /custom/plugins", got)
	}
	if got := resolvePluginDir("/custom/plugins", base, false); got != "/custom/plugins" {
		t.Errorf("explicit+nodchain = %q, want /custom/plugins", got)
	}

	// dchain with no explicit -> ~/.lux/plugins/current (deterministic EVM plugin).
	if got := resolvePluginDir("", base, true); got != want {
		t.Errorf("dchain default = %q, want %q", got, want)
	}

	// non-dchain with no explicit -> "" (luxd uses its own default, unchanged).
	if got := resolvePluginDir("", base, false); got != "" {
		t.Errorf("non-dchain default = %q, want empty", got)
	}
}
