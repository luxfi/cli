// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package netspec

import (
	"testing"
)

func TestDiff_NilState(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "newnet",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "chain1", VM: "subnet-evm"},
				{Name: "chain2", VM: "custom"},
			},
		},
	}

	diff := Diff(spec, nil)

	if !diff.NetworkChanges {
		t.Error("expected NetworkChanges to be true for nil state")
	}
	if len(diff.SubnetsToCreate) != 2 {
		t.Errorf("expected 2 subnets to create, got %d", len(diff.SubnetsToCreate))
	}
	if !diff.HasChanges() {
		t.Error("expected HasChanges() to return true")
	}
}

func TestDiff_EmptyState(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "newnet",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "chain1", VM: "subnet-evm"},
			},
		},
	}

	state := &NetworkState{Name: ""}

	diff := Diff(spec, state)

	if !diff.NetworkChanges {
		t.Error("expected NetworkChanges to be true for empty state")
	}
}

func TestDiff_NoChanges(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "existing",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "chain1", VM: "subnet-evm", VMVersion: "v1.0.0"},
			},
		},
	}

	state := &NetworkState{
		Name:  "existing",
		Nodes: 5,
		Subnets: []SubnetState{
			{Name: "chain1", VM: "subnet-evm", VMVersion: "v1.0.0", Deployed: true},
		},
	}

	diff := Diff(spec, state)

	if diff.HasChanges() {
		t.Error("expected no changes for matching state")
	}
	if diff.NetworkChanges {
		t.Error("expected NetworkChanges to be false")
	}
}

func TestDiff_NodeCountChange(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "testnet",
			Nodes: 7,
		},
	}

	state := &NetworkState{
		Name:  "testnet",
		Nodes: 5,
	}

	diff := Diff(spec, state)

	if !diff.NetworkChanges {
		t.Error("expected NetworkChanges for node count change")
	}
	if !diff.NeedsRestart {
		t.Error("expected NeedsRestart for node count change")
	}
}

func TestDiff_CreateSubnet(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "testnet",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "existing", VM: "subnet-evm"},
				{Name: "new", VM: "custom"},
			},
		},
	}

	state := &NetworkState{
		Name:  "testnet",
		Nodes: 5,
		Subnets: []SubnetState{
			{Name: "existing", VM: "subnet-evm"},
		},
	}

	diff := Diff(spec, state)

	if len(diff.SubnetsToCreate) != 1 {
		t.Errorf("expected 1 subnet to create, got %d", len(diff.SubnetsToCreate))
	}
	if diff.SubnetsToCreate[0].Name != "new" {
		t.Errorf("expected subnet 'new' to create, got %q", diff.SubnetsToCreate[0].Name)
	}
}

func TestDiff_UpdateSubnet(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "testnet",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "chain1", VM: "subnet-evm", VMVersion: "v2.0.0"},
			},
		},
	}

	state := &NetworkState{
		Name:  "testnet",
		Nodes: 5,
		Subnets: []SubnetState{
			{Name: "chain1", VM: "subnet-evm", VMVersion: "v1.0.0"},
		},
	}

	diff := Diff(spec, state)

	if len(diff.SubnetsToUpdate) != 1 {
		t.Errorf("expected 1 subnet to update, got %d", len(diff.SubnetsToUpdate))
	}
}

func TestDiff_DeleteSubnet(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "testnet",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "keep", VM: "subnet-evm"},
			},
		},
	}

	state := &NetworkState{
		Name:  "testnet",
		Nodes: 5,
		Subnets: []SubnetState{
			{Name: "keep", VM: "subnet-evm"},
			{Name: "remove", VM: "custom"},
		},
	}

	diff := Diff(spec, state)

	if len(diff.SubnetsToDelete) != 1 {
		t.Errorf("expected 1 subnet to delete, got %d", len(diff.SubnetsToDelete))
	}
	if diff.SubnetsToDelete[0] != "remove" {
		t.Errorf("expected subnet 'remove' to delete, got %q", diff.SubnetsToDelete[0])
	}
}

func TestDiff_VMTypeChange(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "testnet",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "chain1", VM: "custom"},
			},
		},
	}

	state := &NetworkState{
		Name:  "testnet",
		Nodes: 5,
		Subnets: []SubnetState{
			{Name: "chain1", VM: "subnet-evm"},
		},
	}

	diff := Diff(spec, state)

	if len(diff.SubnetsToUpdate) != 1 {
		t.Errorf("expected 1 subnet to update for VM type change, got %d", len(diff.SubnetsToUpdate))
	}
}

func TestDiff_ChainIDChange(t *testing.T) {
	spec := &NetworkSpec{
		Network: NetworkConfig{
			Name:  "testnet",
			Nodes: 5,
			Subnets: []SubnetSpec{
				{Name: "chain1", VM: "subnet-evm", ChainID: 99999},
			},
		},
	}

	state := &NetworkState{
		Name:  "testnet",
		Nodes: 5,
		Subnets: []SubnetState{
			{Name: "chain1", VM: "subnet-evm", ChainID: 12345},
		},
	}

	diff := Diff(spec, state)

	if len(diff.SubnetsToUpdate) != 1 {
		t.Errorf("expected 1 subnet to update for chain ID change, got %d", len(diff.SubnetsToUpdate))
	}
}

func TestDiffResult_String(t *testing.T) {
	diff := &DiffResult{
		NetworkChanges:  true,
		SubnetsToCreate: []SubnetSpec{{Name: "new1"}, {Name: "new2"}},
		SubnetsToUpdate: []SubnetSpec{{Name: "upd1"}},
		SubnetsToDelete: []string{"del1"},
		NeedsRestart:    true,
	}

	str := diff.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
	if str == "No changes needed." {
		t.Error("expected changes description")
	}
}

func TestDiffResult_HasChanges(t *testing.T) {
	tests := []struct {
		name     string
		diff     DiffResult
		expected bool
	}{
		{
			name:     "no changes",
			diff:     DiffResult{},
			expected: false,
		},
		{
			name:     "network changes only",
			diff:     DiffResult{NetworkChanges: true},
			expected: true,
		},
		{
			name:     "create only",
			diff:     DiffResult{SubnetsToCreate: []SubnetSpec{{Name: "x"}}},
			expected: true,
		},
		{
			name:     "update only",
			diff:     DiffResult{SubnetsToUpdate: []SubnetSpec{{Name: "x"}}},
			expected: true,
		},
		{
			name:     "delete only",
			diff:     DiffResult{SubnetsToDelete: []string{"x"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.HasChanges(); got != tt.expected {
				t.Errorf("HasChanges() = %v, want %v", got, tt.expected)
			}
		})
	}
}
