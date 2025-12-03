// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package netspec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// CurrentAPIVersion is the current schema version
	CurrentAPIVersion = "lux.network/v1"

	// KindNetwork is the kind for network specs
	KindNetwork = "Network"
)

// ParseFile reads and parses a network specification from a file.
// Supports both YAML and JSON formats based on file extension.
func ParseFile(path string) (*NetworkSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return ParseYAML(data)
	case ".json":
		return ParseJSON(data)
	default:
		// Try YAML first, fall back to JSON
		spec, err := ParseYAML(data)
		if err != nil {
			return ParseJSON(data)
		}
		return spec, nil
	}
}

// ParseYAML parses a YAML network specification.
func ParseYAML(data []byte) (*NetworkSpec, error) {
	var spec NetworkSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	if err := validate(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// ParseJSON parses a JSON network specification.
func ParseJSON(data []byte) (*NetworkSpec, error) {
	var spec NetworkSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	if err := validate(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// validate checks that a NetworkSpec is well-formed.
func validate(spec *NetworkSpec) error {
	// Check API version
	if spec.APIVersion == "" {
		spec.APIVersion = CurrentAPIVersion
	}
	if spec.APIVersion != CurrentAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q, expected %q", spec.APIVersion, CurrentAPIVersion)
	}

	// Check kind
	if spec.Kind == "" {
		spec.Kind = KindNetwork
	}
	if spec.Kind != KindNetwork {
		return fmt.Errorf("unsupported kind %q, expected %q", spec.Kind, KindNetwork)
	}

	// Validate network
	if spec.Network.Name == "" {
		return fmt.Errorf("network.name is required")
	}

	// Validate name format (letters, numbers, spaces only)
	for _, r := range spec.Network.Name {
		if r > 127 || !(isLetter(r) || isDigit(r) || r == ' ' || r == '-' || r == '_') {
			return fmt.Errorf("network.name contains invalid character: %c", r)
		}
	}

	// Default nodes to 5 if not specified
	if spec.Network.Nodes == 0 {
		spec.Network.Nodes = 5
	}

	// Validate subnets
	subnetNames := make(map[string]bool)
	for i := range spec.Network.Subnets {
		subnet := &spec.Network.Subnets[i]

		if subnet.Name == "" {
			return fmt.Errorf("subnet[%d].name is required", i)
		}

		// Check for duplicate names
		if subnetNames[subnet.Name] {
			return fmt.Errorf("duplicate subnet name: %s", subnet.Name)
		}
		subnetNames[subnet.Name] = true

		// Validate VM type
		if subnet.VM == "" {
			subnet.VM = "subnet-evm"
		}
		if !isValidVM(subnet.VM) {
			return fmt.Errorf("subnet %q has invalid vm: %s", subnet.Name, subnet.VM)
		}

		// Default validators to 3 if not specified
		if subnet.Validators == 0 {
			subnet.Validators = 3
		}

		// Ensure validators don't exceed network nodes
		if subnet.Validators > spec.Network.Nodes {
			return fmt.Errorf("subnet %q validators (%d) exceeds network nodes (%d)",
				subnet.Name, subnet.Validators, spec.Network.Nodes)
		}

		// Validate validator management if specified
		if subnet.ValidatorManagement != "" && !isValidValidatorManagement(subnet.ValidatorManagement) {
			return fmt.Errorf("subnet %q has invalid validatorManagement: %s", subnet.Name, subnet.ValidatorManagement)
		}

		// Check genesis file exists if specified
		if subnet.Genesis != "" {
			if _, err := os.Stat(subnet.Genesis); err != nil {
				return fmt.Errorf("subnet %q genesis file not found: %s", subnet.Name, subnet.Genesis)
			}
		}
	}

	return nil
}

// isValidVM checks if a VM type is supported.
func isValidVM(vm string) bool {
	for _, v := range ValidVMs {
		if v == vm {
			return true
		}
	}
	return false
}

// isValidValidatorManagement checks if a validator management type is supported.
func isValidValidatorManagement(vm string) bool {
	for _, v := range ValidValidatorManagement {
		if v == vm {
			return true
		}
	}
	return false
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// WriteYAML writes a network specification to YAML format.
func WriteYAML(spec *NetworkSpec, path string) error {
	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// WriteJSON writes a network specification to JSON format.
func WriteJSON(spec *NetworkSpec, path string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// StateToYAML converts a NetworkState to YAML.
func StateToYAML(state *NetworkState) ([]byte, error) {
	return yaml.Marshal(state)
}

// StateToJSON converts a NetworkState to JSON.
func StateToJSON(state *NetworkState) ([]byte, error) {
	return json.MarshalIndent(state, "", "  ")
}
