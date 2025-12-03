// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package netspec

import (
	"fmt"
	"strings"
)

// Diff compares desired spec against current state and returns needed changes.
// This enables idempotent apply - only changes what's different.
func Diff(desired *NetworkSpec, current *NetworkState) *DiffResult {
	result := &DiffResult{}
	var changes []string

	// Check if network exists
	if current == nil || current.Name == "" {
		result.NetworkChanges = true
		for _, s := range desired.Network.Subnets {
			result.SubnetsToCreate = append(result.SubnetsToCreate, s)
		}
		result.Summary = fmt.Sprintf("Network %q does not exist. Will create with %d nodes and %d subnets.",
			desired.Network.Name, desired.Network.Nodes, len(desired.Network.Subnets))
		return result
	}

	// Check node count changes
	if desired.Network.Nodes != current.Nodes {
		result.NetworkChanges = true
		result.NeedsRestart = true
		changes = append(changes, fmt.Sprintf("nodes: %d -> %d", current.Nodes, desired.Network.Nodes))
	}

	// Build map of current subnets
	currentSubnets := make(map[string]*SubnetState)
	for i := range current.Subnets {
		s := &current.Subnets[i]
		currentSubnets[s.Name] = s
	}

	// Check for subnets to create or update
	desiredSubnets := make(map[string]bool)
	for _, desired := range desired.Network.Subnets {
		desiredSubnets[desired.Name] = true

		if existing, ok := currentSubnets[desired.Name]; ok {
			// Check if update needed
			if needsUpdate(desired, existing) {
				result.SubnetsToUpdate = append(result.SubnetsToUpdate, desired)
				changes = append(changes, fmt.Sprintf("update subnet %q", desired.Name))
			}
		} else {
			// Subnet doesn't exist, needs creation
			result.SubnetsToCreate = append(result.SubnetsToCreate, desired)
			changes = append(changes, fmt.Sprintf("create subnet %q", desired.Name))
		}
	}

	// Check for subnets to delete (in current but not in desired)
	for name := range currentSubnets {
		if !desiredSubnets[name] {
			result.SubnetsToDelete = append(result.SubnetsToDelete, name)
			changes = append(changes, fmt.Sprintf("delete subnet %q", name))
		}
	}

	// Build summary
	if len(changes) == 0 {
		result.Summary = "Network is up to date. No changes needed."
	} else {
		result.Summary = fmt.Sprintf("Changes needed: %s", strings.Join(changes, ", "))
	}

	return result
}

// needsUpdate checks if a subnet's configuration differs from current state.
func needsUpdate(desired SubnetSpec, current *SubnetState) bool {
	// Check VM type
	if desired.VM != current.VM {
		return true
	}

	// Check VM version if specified
	if desired.VMVersion != "" && desired.VMVersion != current.VMVersion {
		return true
	}

	// Check chain ID for EVM chains
	if desired.ChainID != 0 && desired.ChainID != current.ChainID {
		return true
	}

	return false
}

// HasChanges returns true if there are any changes to apply.
func (d *DiffResult) HasChanges() bool {
	return d.NetworkChanges ||
		len(d.SubnetsToCreate) > 0 ||
		len(d.SubnetsToUpdate) > 0 ||
		len(d.SubnetsToDelete) > 0
}

// String returns a human-readable representation of the diff.
func (d *DiffResult) String() string {
	if !d.HasChanges() {
		return "No changes needed."
	}

	var lines []string

	if d.NetworkChanges {
		lines = append(lines, "Network configuration changes required")
	}

	if len(d.SubnetsToCreate) > 0 {
		names := make([]string, len(d.SubnetsToCreate))
		for i, s := range d.SubnetsToCreate {
			names[i] = s.Name
		}
		lines = append(lines, fmt.Sprintf("Subnets to create: %s", strings.Join(names, ", ")))
	}

	if len(d.SubnetsToUpdate) > 0 {
		names := make([]string, len(d.SubnetsToUpdate))
		for i, s := range d.SubnetsToUpdate {
			names[i] = s.Name
		}
		lines = append(lines, fmt.Sprintf("Subnets to update: %s", strings.Join(names, ", ")))
	}

	if len(d.SubnetsToDelete) > 0 {
		lines = append(lines, fmt.Sprintf("Subnets to delete: %s", strings.Join(d.SubnetsToDelete, ", ")))
	}

	if d.NeedsRestart {
		lines = append(lines, "Network restart required")
	}

	return strings.Join(lines, "\n")
}
