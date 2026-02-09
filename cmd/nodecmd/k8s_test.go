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
