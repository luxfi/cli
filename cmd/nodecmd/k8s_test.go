// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"testing"
	"time"
)

func TestResolveNamespace(t *testing.T) {
	tests := []struct {
		name      string
		mainnet   bool
		testnet   bool
		devnet    bool
		namespace string
		want      string
		wantErr   bool
	}{
		{
			name:    "mainnet flag",
			mainnet: true,
			want:    "lux-mainnet",
		},
		{
			name:    "testnet flag",
			testnet: true,
			want:    "lux-testnet",
		},
		{
			name:   "devnet flag",
			devnet: true,
			want:   "lux-devnet",
		},
		{
			name:      "explicit namespace overrides",
			mainnet:   true,
			namespace: "custom-ns",
			want:      "custom-ns",
		},
		{
			name:    "no flags = error",
			wantErr: true,
		},
		{
			name:    "multiple network flags = error",
			mainnet: true,
			testnet: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set package-level vars
			flagMainnet = tt.mainnet
			flagTestnet = tt.testnet
			flagDevnet = tt.devnet
			flagNamespace = tt.namespace

			// Reset after test
			defer func() {
				flagMainnet = false
				flagTestnet = false
				flagDevnet = false
				flagNamespace = ""
			}()

			got, err := resolveNamespace()
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveNamespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveNetwork(t *testing.T) {
	tests := []struct {
		name      string
		mainnet   bool
		testnet   bool
		devnet    bool
		namespace string
		want      string
		wantErr   bool
	}{
		{name: "mainnet flag", mainnet: true, want: "mainnet"},
		{name: "testnet flag", testnet: true, want: "testnet"},
		{name: "devnet flag", devnet: true, want: "devnet"},
		{name: "namespace lux-mainnet", namespace: "lux-mainnet", want: "mainnet"},
		{name: "namespace lux-testnet", namespace: "lux-testnet", want: "testnet"},
		{name: "namespace lux-devnet", namespace: "lux-devnet", want: "devnet"},
		{name: "custom namespace = error", namespace: "custom-ns", wantErr: true},
		{name: "no flags = error", wantErr: true},
		{name: "multiple flags = error", mainnet: true, testnet: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagMainnet = tt.mainnet
			flagTestnet = tt.testnet
			flagDevnet = tt.devnet
			flagNamespace = tt.namespace
			defer func() {
				flagMainnet = false
				flagTestnet = false
				flagDevnet = false
				flagNamespace = ""
			}()

			got, err := resolveNetwork()
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveNetwork() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveNetwork() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultChartPath(t *testing.T) {
	// Without env var, should return ~/work/lux/devops/charts/lux
	t.Setenv("LUX_CHART_PATH", "")
	p := defaultChartPath()
	if p == "" {
		t.Error("defaultChartPath() returned empty string")
	}

	// With env var, should return the env value
	t.Setenv("LUX_CHART_PATH", "/custom/chart")
	p = defaultChartPath()
	if p != "/custom/chart" {
		t.Errorf("defaultChartPath() = %q, want /custom/chart", p)
	}
}

func TestNetworkFlag(t *testing.T) {
	tests := []struct {
		namespace string
		want      string
	}{
		{"lux-mainnet", "mainnet"},
		{"lux-testnet", "testnet"},
		{"lux-devnet", "devnet"},
		{"custom", "namespace custom"},
	}

	for _, tt := range tests {
		got := networkFlag(tt.namespace)
		if got != tt.want {
			t.Errorf("networkFlag(%q) = %q, want %q", tt.namespace, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		secs int
		want string
	}{
		{"seconds", 45, "45s"},
		{"minutes", 300, "5m"},
		{"hours", 7200, "2h"},
		{"days", 172800, "2d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(time.Duration(tt.secs) * time.Second)
			if got != tt.want {
				t.Errorf("formatDuration(%ds) = %q, want %q", tt.secs, got, tt.want)
			}
		})
	}
}
