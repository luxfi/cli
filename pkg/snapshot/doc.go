// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package snapshot provides functionality for creating, managing, and restoring
// network snapshots with support for multi-node coordination, database flushing,
// and chunked uploads for GitHub.
//
// Key Features:
//   - Coordinated snapshots across multiple nodes
//   - Database flushing for PebbleDB, BadgerDB, and LevelDB
//   - Automatic chunking into 99MB pieces for GitHub upload
//   - Checksum verification and metadata management
//   - Parallel snapshot creation for minimal downtime
//
// Usage:
//   manager := snapshot.NewSnapshotManager("~/.lux", "mainnet", 5)
//   err := manager.CreateSnapshot("production-backup")
//   if err != nil {
//       // handle error
//   }
//
// The snapshot system is designed to work with Lux networks of any size,
// providing consistent state capture for rollback and recovery scenarios.
package snapshot