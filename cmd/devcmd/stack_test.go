// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package devcmd

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPortForApp(t *testing.T) {
	tests := []struct {
		portBase   int
		chainIndex int
		want       int
	}{
		{9650, 0, 9650},
		{9650, 1, 9750},
		{9650, 2, 9850},
		{3001, 0, 3001},
		{3001, 1, 3101},
		{3001, 3, 3301},
	}
	for _, tt := range tests {
		got := PortForApp(tt.portBase, tt.chainIndex)
		if got != tt.want {
			t.Errorf("PortForApp(%d, %d) = %d, want %d", tt.portBase, tt.chainIndex, got, tt.want)
		}
	}
}

func TestChainInstanceName(t *testing.T) {
	tests := []struct {
		app   string
		index int
		want  string
	}{
		{"luxd", 0, "luxd"},
		{"luxd", 1, "luxd-1"},
		{"explorer", 0, "explorer"},
		{"explorer", 2, "explorer-2"},
	}
	for _, tt := range tests {
		got := chainInstanceName(tt.app, tt.index)
		if got != tt.want {
			t.Errorf("chainInstanceName(%q, %d) = %q, want %q", tt.app, tt.index, got, tt.want)
		}
	}
}

func TestIsDigitSuffix(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"explorer-1", true},
		{"luxd-0", true},
		{"luxd-12", true},
		{"explorer", false},
		{"explorer-", false},
		{"explorer-abc", false},
	}
	for _, tt := range tests {
		got := isDigitSuffix(tt.input)
		if got != tt.want {
			t.Errorf("isDigitSuffix(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		input string
		want  string
	}{
		{"~/.lux/dev/data", filepath.Join(home, ".lux/dev/data")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}
	for _, tt := range tests {
		got := expandPath(tt.input)
		if got != tt.want {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Chains != 1 {
		t.Errorf("default chains = %d, want 1", cfg.Chains)
	}
	if len(cfg.Apps) != 8 {
		t.Errorf("default apps = %d, want 8", len(cfg.Apps))
	}

	// luxd must be first and enabled
	if cfg.Apps[0].Name != "luxd" {
		t.Errorf("first app = %q, want luxd", cfg.Apps[0].Name)
	}
	if !cfg.Apps[0].Enabled {
		t.Error("luxd should be enabled by default")
	}

	// exchange must be disabled
	exchange := findApp(cfg, "exchange")
	if exchange == nil {
		t.Fatal("exchange not found in default config")
	}
	if exchange.Enabled {
		t.Error("exchange should be disabled by default")
	}
}

func TestConfigSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-stack.yaml")

	// Override the config path for this test
	origPath := stackConfigPath
	stackConfigPath = path
	defer func() { stackConfigPath = origPath }()

	cfg := defaultConfig()
	cfg.Chains = 3

	if err := saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if loaded.Chains != 3 {
		t.Errorf("loaded chains = %d, want 3", loaded.Chains)
	}
	if len(loaded.Apps) != len(cfg.Apps) {
		t.Errorf("loaded apps = %d, want %d", len(loaded.Apps), len(cfg.Apps))
	}
}

func TestConfigYAMLRoundTrip(t *testing.T) {
	cfg := defaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded StackConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.Chains != cfg.Chains {
		t.Errorf("chains: got %d, want %d", loaded.Chains, cfg.Chains)
	}
	for i, app := range loaded.Apps {
		if app.Name != cfg.Apps[i].Name {
			t.Errorf("app[%d].name: got %q, want %q", i, app.Name, cfg.Apps[i].Name)
		}
		if app.PortBase != cfg.Apps[i].PortBase {
			t.Errorf("app[%d].port_base: got %d, want %d", i, app.PortBase, cfg.Apps[i].PortBase)
		}
		if app.Enabled != cfg.Apps[i].Enabled {
			t.Errorf("app[%d].enabled: got %v, want %v", i, app.Enabled, cfg.Apps[i].Enabled)
		}
	}
}

func TestPIDFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	name := "test-app"
	pid := 12345

	if err := writePID(dir, name, pid); err != nil {
		t.Fatalf("writePID: %v", err)
	}

	got, err := readPID(dir, name)
	if err != nil {
		t.Fatalf("readPID: %v", err)
	}
	if got != pid {
		t.Errorf("readPID = %d, want %d", got, pid)
	}

	// Remove and verify gone
	removePIDFile(dir, name)
	_, err = readPID(dir, name)
	if err == nil {
		t.Error("expected error after removePIDFile")
	}
}

func TestReadPIDNonExistent(t *testing.T) {
	dir := t.TempDir()
	_, err := readPID(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent PID file")
	}
}

func TestFindApp(t *testing.T) {
	cfg := defaultConfig()

	luxd := findApp(cfg, "luxd")
	if luxd == nil {
		t.Fatal("luxd not found")
	}
	if luxd.PortBase != 9650 {
		t.Errorf("luxd port_base = %d, want 9650", luxd.PortBase)
	}

	missing := findApp(cfg, "nonexistent")
	if missing != nil {
		t.Error("expected nil for missing app")
	}
}

func TestPortDeconflictionMultiChain(t *testing.T) {
	cfg := defaultConfig()
	cfg.Chains = 3

	// Verify no port collisions across all apps and chains
	ports := make(map[int]string)
	for _, app := range cfg.Apps {
		for i := 0; i < cfg.Chains; i++ {
			port := PortForApp(app.PortBase, i)
			name := chainInstanceName(app.Name, i)
			if existing, ok := ports[port]; ok {
				t.Errorf("port collision: %d used by both %s and %s", port, existing, name)
			}
			ports[port] = name
		}
	}
}

// TestPortForAppChecked_RejectsOverflow covers the Red-#7 vector:
// large --chains values must not silently produce out-of-range TCP
// ports (which would either cause listener failures, overflow into
// privileged ranges, or wrap into negative numbers on some systems).
func TestPortForAppChecked_RejectsOverflow(t *testing.T) {
	// Well within range.
	if p, err := PortForAppChecked(9650, 0); err != nil || p != 9650 {
		t.Fatalf("portBase 9650 chain 0: got p=%d err=%v", p, err)
	}
	if p, err := PortForAppChecked(9650, 4); err != nil || p != 10050 {
		t.Fatalf("portBase 9650 chain 4: got p=%d err=%v", p, err)
	}

	// Out-of-range. With portStride=100, base 9650 + 1000 chains = 109,650 > 65535.
	if _, err := PortForAppChecked(9650, 1000); err == nil {
		t.Fatal("expected overflow error for portBase=9650 chainIndex=1000")
	}
	// Exactly at the boundary should also fail for a large stride.
	if _, err := PortForAppChecked(60000, 100); err == nil {
		t.Fatal("expected overflow error for portBase=60000 chainIndex=100")
	}
}

// TestValidateStackConfig_RejectsInjections confirms the config loader
// refuses shapes that would let a hostile stack.yaml escape its sandbox.
// Each sub-case is a single-field delta from a known-good config.
func TestValidateStackConfig_RejectsInjections(t *testing.T) {
	good := func() *StackConfig {
		return &StackConfig{
			Chains: 2,
			Apps: []AppEntry{
				{Name: "luxd", PortBase: 9650, Enabled: true},
				{Name: "explorer", PortBase: 3001, Enabled: true, Binary: "ghcr.io/luxfi/explorer:local"},
			},
		}
	}

	if err := validateStackConfig(good()); err != nil {
		t.Fatalf("baseline config should validate: %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(*StackConfig)
		wantErr bool
	}{
		{"chains=0", func(c *StackConfig) { c.Chains = 0 }, true},
		{"chains=33", func(c *StackConfig) { c.Chains = 33 }, true},
		{"app name path-traversal", func(c *StackConfig) { c.Apps[0].Name = "../evil" }, true},
		{"app name uppercase", func(c *StackConfig) { c.Apps[0].Name = "LuxD" }, true},
		{"app name empty", func(c *StackConfig) { c.Apps[0].Name = "" }, true},
		{"port base 0", func(c *StackConfig) { c.Apps[0].PortBase = 0 }, true},
		{"port base 65536", func(c *StackConfig) { c.Apps[0].PortBase = 65536 }, true},
		{"port overflow with --chains", func(c *StackConfig) { c.Chains = 32; c.Apps[0].PortBase = 65000 }, true},
		{"binary shell injection semicolon", func(c *StackConfig) {
			c.Apps[1].Binary = "ghcr.io/luxfi/explorer:local; rm -rf /"
		}, true},
		{"binary shell injection backtick", func(c *StackConfig) {
			c.Apps[1].Binary = "ghcr.io/x:`id`"
		}, true},
		{"binary shell injection pipe", func(c *StackConfig) {
			c.Apps[1].Binary = "foo | tee /etc/passwd"
		}, true},
		{"benign slashes allowed", func(c *StackConfig) {
			c.Apps[1].Binary = "/usr/local/bin/explorer-0.1"
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := good()
			tc.mutate(cfg)
			err := validateStackConfig(cfg)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
