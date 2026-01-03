// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package models contains data structures and types used throughout the CLI.
package models

// Exportable wraps sidecar and genesis data for export operations.
type Exportable struct {
	Sidecar Sidecar
	Genesis []byte
}
