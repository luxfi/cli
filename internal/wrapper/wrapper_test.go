// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package wrapper

import (
	"os"
	"testing"
)

func TestRewriteArgs(t *testing.T) {
	tests := []struct {
		name     string
		argv     []string
		wantArgs []string
	}{
		{
			name:     "lux passthrough",
			argv:     []string{"lux", "chain", "list"},
			wantArgs: []string{"lux", "chain", "list"},
		},
		{
			name:     "lux no args passthrough",
			argv:     []string{"lux"},
			wantArgs: []string{"lux"},
		},
		{
			name:     "lux-zk rewrites",
			argv:     []string{"lux-zk", "--help"},
			wantArgs: []string{"lux-zk", "zk", "--help"},
		},
		{
			name:     "bare zk rewrites",
			argv:     []string{"zk", "ceremony", "init"},
			wantArgs: []string{"zk", "zk", "ceremony", "init"},
		},
		{
			name:     "lux-fhe rewrites",
			argv:     []string{"lux-fhe", "keygen"},
			wantArgs: []string{"lux-fhe", "fhe", "keygen"},
		},
		{
			name:     "bare fhe rewrites",
			argv:     []string{"fhe"},
			wantArgs: []string{"fhe", "fhe"},
		},
		{
			name:     "lux-mpc rewrites",
			argv:     []string{"lux-mpc", "node", "status"},
			wantArgs: []string{"lux-mpc", "mpc", "node", "status"},
		},
		{
			name:     "bare mpc rewrites",
			argv:     []string{"mpc", "node", "init"},
			wantArgs: []string{"mpc", "mpc", "node", "init"},
		},
		{
			name:     "lux-kms rewrites",
			argv:     []string{"lux-kms", "key", "list"},
			wantArgs: []string{"lux-kms", "kms", "key", "list"},
		},
		{
			name:     "bare kms rewrites",
			argv:     []string{"kms", "server", "start"},
			wantArgs: []string{"kms", "kms", "server", "start"},
		},
		{
			name:     "lux-rt rewrites",
			argv:     []string{"lux-rt", "keygen"},
			wantArgs: []string{"lux-rt", "rt", "keygen"},
		},
		{
			name:     "bare rt rewrites",
			argv:     []string{"rt", "sign"},
			wantArgs: []string{"rt", "rt", "sign"},
		},
		{
			name:     "bare corona rewrites",
			argv:     []string{"corona", "verify"},
			wantArgs: []string{"corona", "corona", "verify"},
		},
		{
			name:     "lux-explore rewrites",
			argv:     []string{"lux-explore"},
			wantArgs: []string{"lux-explore", "explore"},
		},
		{
			name:     "bare explore rewrites",
			argv:     []string{"explore"},
			wantArgs: []string{"explore", "explore"},
		},
		{
			name:     "platform suffix lux-linux-amd64 passthrough",
			argv:     []string{"lux-linux-amd64", "zk", "--help"},
			wantArgs: []string{"lux-linux-amd64", "zk", "--help"},
		},
		{
			name:     "platform suffix lux-darwin-arm64 passthrough",
			argv:     []string{"lux-darwin-arm64", "chain", "list"},
			wantArgs: []string{"lux-darwin-arm64", "chain", "list"},
		},
		{
			name:     "unknown lux-foo no rewrite",
			argv:     []string{"lux-unknown", "foo"},
			wantArgs: []string{"lux-unknown", "foo"},
		},
		{
			name:     "unknown bare binary no rewrite",
			argv:     []string{"kubectl", "get", "pods"},
			wantArgs: []string{"kubectl", "get", "pods"},
		},
		{
			name:     "bare domain no extra args",
			argv:     []string{"zk"},
			wantArgs: []string{"zk", "zk"},
		},
		{
			name:     "full path stripped to base",
			argv:     []string{"/usr/local/bin/lux-zk", "ceremony"},
			wantArgs: []string{"/usr/local/bin/lux-zk", "zk", "ceremony"},
		},
		{
			name:     "full path lux passthrough",
			argv:     []string{"/usr/local/bin/lux", "zk"},
			wantArgs: []string{"/usr/local/bin/lux", "zk"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origArgs := os.Args
			defer func() { os.Args = origArgs }()

			os.Args = make([]string, len(tt.argv))
			copy(os.Args, tt.argv)
			RewriteArgs()

			if len(os.Args) != len(tt.wantArgs) {
				t.Fatalf("got %d args %v, want %d args %v",
					len(os.Args), os.Args, len(tt.wantArgs), tt.wantArgs)
			}
			for i := range os.Args {
				if os.Args[i] != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q (full: %v)",
						i, os.Args[i], tt.wantArgs[i], os.Args)
				}
			}
		})
	}
}
