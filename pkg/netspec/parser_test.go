// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package netspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *NetworkSpec)
	}{
		{
			name: "valid minimal spec",
			input: `
apiVersion: lux.network/v1
kind: Network
network:
  name: testnet
`,
			wantErr: false,
			check: func(t *testing.T, spec *NetworkSpec) {
				if spec.Network.Name != "testnet" {
					t.Errorf("expected name 'testnet', got %q", spec.Network.Name)
				}
				if spec.Network.Nodes != 5 {
					t.Errorf("expected default nodes 5, got %d", spec.Network.Nodes)
				}
			},
		},
		{
			name: "valid full spec",
			input: `
apiVersion: lux.network/v1
kind: Network
network:
  name: mydevnet
  nodes: 7
  luxdVersion: v1.20.3
  subnets:
    - name: mychain
      vm: subnet-evm
      chainId: 12345
      tokenSymbol: MYT
      validators: 3
      testDefaults: true
    - name: customchain
      vm: custom
      validators: 5
`,
			wantErr: false,
			check: func(t *testing.T, spec *NetworkSpec) {
				if spec.Network.Name != "mydevnet" {
					t.Errorf("expected name 'mydevnet', got %q", spec.Network.Name)
				}
				if spec.Network.Nodes != 7 {
					t.Errorf("expected nodes 7, got %d", spec.Network.Nodes)
				}
				if len(spec.Network.Subnets) != 2 {
					t.Errorf("expected 2 subnets, got %d", len(spec.Network.Subnets))
				}
				if spec.Network.Subnets[0].ChainID != 12345 {
					t.Errorf("expected chainId 12345, got %d", spec.Network.Subnets[0].ChainID)
				}
			},
		},
		{
			name: "defaults applied",
			input: `
network:
  name: test
  subnets:
    - name: chain1
`,
			wantErr: false,
			check: func(t *testing.T, spec *NetworkSpec) {
				if spec.APIVersion != CurrentAPIVersion {
					t.Errorf("expected apiVersion %q, got %q", CurrentAPIVersion, spec.APIVersion)
				}
				if spec.Kind != KindNetwork {
					t.Errorf("expected kind %q, got %q", KindNetwork, spec.Kind)
				}
				if spec.Network.Subnets[0].VM != "subnet-evm" {
					t.Errorf("expected default VM 'subnet-evm', got %q", spec.Network.Subnets[0].VM)
				}
				if spec.Network.Subnets[0].Validators != 3 {
					t.Errorf("expected default validators 3, got %d", spec.Network.Subnets[0].Validators)
				}
			},
		},
		{
			name: "missing name",
			input: `
apiVersion: lux.network/v1
kind: Network
network:
  nodes: 5
`,
			wantErr: true,
		},
		{
			name: "invalid API version",
			input: `
apiVersion: wrong/v1
kind: Network
network:
  name: test
`,
			wantErr: true,
		},
		{
			name: "invalid kind",
			input: `
apiVersion: lux.network/v1
kind: WrongKind
network:
  name: test
`,
			wantErr: true,
		},
		{
			name: "invalid VM type",
			input: `
network:
  name: test
  subnets:
    - name: chain1
      vm: invalid-vm
`,
			wantErr: true,
		},
		{
			name: "validators exceed nodes",
			input: `
network:
  name: test
  nodes: 3
  subnets:
    - name: chain1
      validators: 5
`,
			wantErr: true,
		},
		{
			name: "duplicate subnet names",
			input: `
network:
  name: test
  subnets:
    - name: chain1
    - name: chain1
`,
			wantErr: true,
		},
		{
			name: "invalid name character",
			input: `
network:
  name: test@network
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := ParseYAML([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.check != nil {
				tt.check(t, spec)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	input := `{
		"apiVersion": "lux.network/v1",
		"kind": "Network",
		"network": {
			"name": "jsonnet",
			"nodes": 3,
			"subnets": [
				{
					"name": "mychain",
					"vm": "subnet-evm",
					"chainId": 99999
				}
			]
		}
	}`

	spec, err := ParseJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Network.Name != "jsonnet" {
		t.Errorf("expected name 'jsonnet', got %q", spec.Network.Name)
	}
	if spec.Network.Nodes != 3 {
		t.Errorf("expected nodes 3, got %d", spec.Network.Nodes)
	}
	if len(spec.Network.Subnets) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(spec.Network.Subnets))
	}
}

func TestParseFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Test YAML file
	yamlPath := filepath.Join(tmpDir, "spec.yaml")
	yamlContent := `
network:
  name: filetest
  nodes: 5
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write yaml file: %v", err)
	}

	spec, err := ParseFile(yamlPath)
	if err != nil {
		t.Fatalf("failed to parse yaml file: %v", err)
	}
	if spec.Network.Name != "filetest" {
		t.Errorf("expected name 'filetest', got %q", spec.Network.Name)
	}

	// Test JSON file
	jsonPath := filepath.Join(tmpDir, "spec.json")
	jsonContent := `{"network": {"name": "jsonfile", "nodes": 3}}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write json file: %v", err)
	}

	spec, err = ParseFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to parse json file: %v", err)
	}
	if spec.Network.Name != "jsonfile" {
		t.Errorf("expected name 'jsonfile', got %q", spec.Network.Name)
	}
}

func TestWriteYAML(t *testing.T) {
	spec := &NetworkSpec{
		APIVersion: CurrentAPIVersion,
		Kind:       KindNetwork,
		Network: NetworkConfig{
			Name:  "writetest",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{
					Name:        "chain1",
					VM:          "subnet-evm",
					ChainID:     12345,
					TokenSymbol: "TST",
					Validators:  3,
				},
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "out.yaml")

	if err := WriteYAML(spec, path); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	// Read it back
	readSpec, err := ParseFile(path)
	if err != nil {
		t.Fatalf("failed to read back yaml: %v", err)
	}

	if readSpec.Network.Name != spec.Network.Name {
		t.Errorf("name mismatch: got %q, want %q", readSpec.Network.Name, spec.Network.Name)
	}
	if len(readSpec.Network.Subnets) != len(spec.Network.Subnets) {
		t.Errorf("subnet count mismatch: got %d, want %d",
			len(readSpec.Network.Subnets), len(spec.Network.Subnets))
	}
}

func TestWriteJSON(t *testing.T) {
	spec := &NetworkSpec{
		APIVersion: CurrentAPIVersion,
		Kind:       KindNetwork,
		Network: NetworkConfig{
			Name:  "jsonwrite",
			Nodes: 7,
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "out.json")

	if err := WriteJSON(spec, path); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}

	// Read it back
	readSpec, err := ParseFile(path)
	if err != nil {
		t.Fatalf("failed to read back json: %v", err)
	}

	if readSpec.Network.Name != spec.Network.Name {
		t.Errorf("name mismatch: got %q, want %q", readSpec.Network.Name, spec.Network.Name)
	}
	if readSpec.Network.Nodes != spec.Network.Nodes {
		t.Errorf("nodes mismatch: got %d, want %d", readSpec.Network.Nodes, spec.Network.Nodes)
	}
}
