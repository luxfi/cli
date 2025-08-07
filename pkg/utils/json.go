// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// ValidateJSON takes a json string and returns it's byte representation
// if it contains valid JSON
func ValidateJSON(path string) ([]byte, error) {
	var content map[string]interface{}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// if the file is not valid json, this fails
	if err := json.Unmarshal(contentBytes, &content); err != nil {
		return nil, fmt.Errorf("this looks like invalid JSON: %w", err)
	}

	return contentBytes, nil
}

// ReadJSON reads a JSON file and unmarshals it into the provided interface
func ReadJSON(path string, v interface{}) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(contentBytes, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", path, err)
	}

	return nil
}

// GetJSONKey retrieves a value from a map by key and returns it as the specified type
func GetJSONKey[T any](m map[string]interface{}, key string) (T, error) {
	var zero T
	value, ok := m[key]
	if !ok {
		return zero, fmt.Errorf("key %s not found in map", key)
	}
	
	typedValue, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("value for key %s is not of the expected type", key)
	}
	
	return typedValue, nil
}

// WriteJSON writes the provided interface to a JSON file
func WriteJSON(path string, v interface{}) error {
	contentBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	if err := os.WriteFile(path, contentBytes, 0644); err != nil {
		return fmt.Errorf("failed to write JSON to %s: %w", path, err)
	}
	
	return nil
}
