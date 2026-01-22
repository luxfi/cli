// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/database"
	"github.com/luxfi/database/badgerdb"
)

// ChunkSize is the maximum size for a backup chunk (99MB to fit GitHub limits)
const ChunkSize = int64(99 * 1024 * 1024)

// SnapshotManifest represents the manifest file for a snapshot
type SnapshotManifest struct {
	Network            string          `json:"network"`
	ChainID            uint64          `json:"chain_id"`
	NodeID             uint64          `json:"node_id,omitempty"`            // Node ID (1-5)
	ChainDataID        string          `json:"chain_data_id,omitempty"`      // If set, this is chainData not main DB
	Base               SnapshotEntry   `json:"base"`
	Incrementals       []SnapshotEntry `json:"incrementals"`
	StateRoot          string          `json:"state_root"`
	CreatedAt          string          `json:"created_at"`
	LastVersion        uint64          `json:"last_version"`
	PrevManifestSHA256 string          `json:"prev_manifest_sha256,omitempty"`
}

// SnapshotEntry represents a backup entry (base or incremental)
type SnapshotEntry struct {
	Height uint64 `json:"height"`
	Since  uint64 `json:"since"`
	Parts  []Part `json:"parts"`
}

// Part represents a single file part of a split stream
type Part struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// SnapshotManager handles database snapshots
type SnapshotManager struct {
	baseDir string
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(baseDir string) *SnapshotManager {
	return &SnapshotManager{
		baseDir: baseDir,
	}
}

// chunkWriter splits a single byte stream into ~chunkSize parts.
type chunkWriter struct {
	dir       string
	prefix    string
	chunkSize int64

	partIdx int
	f       *os.File
	n       int64
	h       hash.Hash

	parts []Part
}

func newChunkWriter(dir, prefix string, chunkSize int64) (*chunkWriter, error) {
	cw := &chunkWriter{dir: dir, prefix: prefix, chunkSize: chunkSize}
	return cw, cw.rotate()
}

func (cw *chunkWriter) rotate() error {
	// finalize previous
	if cw.f != nil {
		sum := hex.EncodeToString(cw.h.Sum(nil))
		if err := cw.f.Close(); err != nil {
			return err
		}
		cw.parts = append(cw.parts, Part{
			Name:   filepath.Base(cw.f.Name()),
			Bytes:  cw.n,
			SHA256: sum,
		})
	}

	name := filepath.Join(cw.dir, fmt.Sprintf("%s.part%05d.zst", cw.prefix, cw.partIdx))
	cw.partIdx++

	f, err := os.Create(name)
	if err != nil {
		return err
	}

	cw.f = f
	cw.n = 0
	cw.h = sha256.New()
	return nil
}

func (cw *chunkWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		if cw.n >= cw.chunkSize {
			if err := cw.rotate(); err != nil {
				return written, err
			}
		}

		space := cw.chunkSize - cw.n
		toWrite := int64(len(p))
		if toWrite > space {
			toWrite = space
		}

		n, err := cw.f.Write(p[:toWrite])
		if n > 0 {
			_, _ = cw.h.Write(p[:n])
			cw.n += int64(n)
			written += n
		}
		if err != nil {
			return written, err
		}
		p = p[toWrite:]
	}
	return written, nil
}

func (cw *chunkWriter) Close() ([]Part, error) {
	// finalize last
	if cw.f == nil {
		return cw.parts, nil
	}
	sum := hex.EncodeToString(cw.h.Sum(nil))
	if err := cw.f.Close(); err != nil {
		return nil, err
	}
	cw.parts = append(cw.parts, Part{
		Name:   filepath.Base(cw.f.Name()),
		Bytes:  cw.n,
		SHA256: sum,
	})
	cw.f = nil
	return cw.parts, nil
}

// snapshotTask represents a single snapshot operation
type snapshotTask struct {
	network     string
	nodeName    string
	nodeID      uint64
	dbPath      string
	chainDataID string // empty for main DB, set for chainData
	incremental bool
}

// snapshotResult represents the result of a snapshot operation
type snapshotResult struct {
	task    snapshotTask
	err     error
	mode    string // "base", "incremental", or "skipped"
}

// CreateSnapshot creates a snapshot of all discovered local networks and nodes
// Captures BOTH main database AND all chainData databases for complete state
// Operations run in parallel for speed
func (sm *SnapshotManager) CreateSnapshot(snapshotName string, incremental bool) error {
	ux.Logger.PrintToUser("Creating snapshot '%s' (incremental=%v)...", snapshotName, incremental)

	// Collect all snapshot tasks
	var tasks []snapshotTask

	runsDir := filepath.Join(sm.baseDir, "runs")
	netEntries, err := os.ReadDir(runsDir)
	if err != nil {
		return fmt.Errorf("failed to read runs dir: %w", err)
	}

	for _, netEntry := range netEntries {
		if !netEntry.IsDir() {
			continue
		}
		networkName := netEntry.Name()
		if networkName == "server" || strings.Contains(networkName, ".backup") {
			continue
		}

		netDir := filepath.Join(runsDir, networkName)
		currentLink := filepath.Join(netDir, "current")
		runDir := ""
		if target, err := os.Readlink(currentLink); err == nil {
			runDir = filepath.Join(netDir, target)
		} else {
			runEntries, _ := os.ReadDir(netDir)
			for _, re := range runEntries {
				if re.IsDir() && strings.HasPrefix(re.Name(), "run_") {
					runDir = filepath.Join(netDir, re.Name())
				}
			}
		}
		if runDir == "" {
			continue
		}

		nodeEntries, err := os.ReadDir(runDir)
		if err != nil {
			continue
		}

		for _, nodeEntry := range nodeEntries {
			if !nodeEntry.IsDir() || !strings.HasPrefix(nodeEntry.Name(), "node") {
				continue
			}
			nodeName := nodeEntry.Name()
			nodeIDStr := strings.TrimPrefix(nodeName, "node")
			nodeID, _ := strconv.ParseUint(nodeIDStr, 10, 64)

			// Main DB task
			dbPattern := filepath.Join(runDir, nodeName, "db", "*", "db")
			dbMatches, _ := filepath.Glob(dbPattern)
			if len(dbMatches) == 0 {
				dbMatches, _ = filepath.Glob(filepath.Join(runDir, nodeName, "db"))
			}
			if len(dbMatches) > 0 {
				tasks = append(tasks, snapshotTask{
					network:     networkName,
					nodeName:    nodeName,
					nodeID:      nodeID,
					dbPath:      dbMatches[0],
					chainDataID: "",
					incremental: incremental,
				})
			}

			// ChainData tasks
			chainDataPattern := filepath.Join(runDir, nodeName, "chainData", "network-*", "*", "db", "badgerdb")
			chainDBMatches, _ := filepath.Glob(chainDataPattern)
			for _, chainDBPath := range chainDBMatches {
				parts := strings.Split(chainDBPath, string(os.PathSeparator))
				var chainDataID string
				for i, p := range parts {
					if p == "db" && i > 0 {
						chainDataID = parts[i-1]
						break
					}
				}
				if chainDataID == "" {
					continue
				}
				tasks = append(tasks, snapshotTask{
					network:     networkName,
					nodeName:    nodeName,
					nodeID:      nodeID,
					dbPath:      chainDBPath,
					chainDataID: chainDataID,
					incremental: incremental,
				})
			}
		}
	}

	// Execute tasks in parallel
	var wg sync.WaitGroup
	results := make(chan snapshotResult, len(tasks))

	for _, task := range tasks {
		wg.Add(1)
		go func(t snapshotTask) {
			defer wg.Done()
			result := sm.executeSnapshotTask(t, snapshotName)
			results <- result
		}(task)
	}

	// Wait for all tasks to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and report results
	for result := range results {
		if result.mode == "skipped" {
			if result.task.chainDataID == "" {
				ux.Logger.PrintToUser("Skipping %s/%s main DB: locked", result.task.network, result.task.nodeName)
			} else {
				ux.Logger.PrintToUser("Skipping %s/%s chain %s: locked", result.task.network, result.task.nodeName, result.task.chainDataID[:8])
			}
		} else if result.err != nil {
			if result.task.chainDataID == "" {
				ux.Logger.PrintToUser("Warning: Failed %s/%s main DB: %v", result.task.network, result.task.nodeName, result.err)
			} else {
				ux.Logger.PrintToUser("Warning: Failed %s/%s chain %s: %v", result.task.network, result.task.nodeName, result.task.chainDataID[:8], result.err)
			}
		} else {
			if result.task.chainDataID == "" {
				ux.Logger.PrintToUser("✓ Snapshotted %s/%s main DB (%s)", result.task.network, result.task.nodeName, result.mode)
			} else {
				ux.Logger.PrintToUser("✓ Snapshotted %s/%s chain %s (%s)", result.task.network, result.task.nodeName, result.task.chainDataID[:8], result.mode)
			}
		}
	}

	return nil
}

// executeSnapshotTask executes a single snapshot task
func (sm *SnapshotManager) executeSnapshotTask(task snapshotTask, snapshotName string) snapshotResult {
	db, err := badgerdb.New(task.dbPath, nil, "", nil)
	if err != nil {
		return snapshotResult{task: task, mode: "skipped"}
	}
	defer db.Close()

	if task.chainDataID == "" {
		// Main DB snapshot
		var parentManifest *SnapshotManifest
		if task.incremental {
			parentManifest, _ = sm.GetLatestManifest(task.network, task.nodeID)
		}

		if parentManifest != nil {
			_, err = sm.CreateIncrementalSnapshot(task.network, task.nodeID, db, parentManifest, snapshotName)
			if err == nil {
				return snapshotResult{task: task, mode: "incremental"}
			}
			// Fall back to base
		}
		_, err = sm.CreateBaseSnapshot(task.network, task.nodeID, db, 0, "", snapshotName)
		return snapshotResult{task: task, err: err, mode: "base"}
	} else {
		// ChainData snapshot - also supports incremental
		var parentManifest *SnapshotManifest
		if task.incremental {
			parentManifest, _ = sm.GetLatestChainDataManifest(task.network, task.nodeID, task.chainDataID)
		}

		if parentManifest != nil {
			_, err = sm.CreateIncrementalChainDataSnapshot(task.network, task.nodeID, task.chainDataID, db, parentManifest, snapshotName)
			if err == nil {
				return snapshotResult{task: task, mode: "incremental"}
			}
			// Fall back to base
		}
		_, err = sm.CreateChainDataSnapshot(task.network, task.nodeID, task.chainDataID, db, snapshotName)
		return snapshotResult{task: task, err: err, mode: "base"}
	}
}

// CreateBaseSnapshot creates a full base snapshot using streaming chunking
func (sm *SnapshotManager) CreateBaseSnapshot(
	network string,
	chainID uint64,
	db database.Database,
	height uint64,
	stateRoot string,
	snapshotID string,
) (*SnapshotManifest, error) {

	if snapshotID == "" {
		snapshotID = time.Now().Format("2006-01-02")
	}
	snapshotDir := filepath.Join(sm.baseDir, "snapshots", snapshotID, network, fmt.Sprintf("chain_%d", chainID))
	chunksDir := filepath.Join(snapshotDir, "chunks")

	if err := os.MkdirAll(chunksDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %w", err)
	}

	backupPrefix := fmt.Sprintf("base_%d", height)

	// Setup pipeline: db.Backup -> zstd -> chunkWriter -> disk
	chunkWriter, err := newChunkWriter(chunksDir, backupPrefix, ChunkSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk writer: %w", err)
	}

	zstdWriter, err := zstd.NewWriter(chunkWriter, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to create zstd writer: %w", err)
	}

	lastVersion, err := db.Backup(zstdWriter, 0)
	if err != nil {
		zstdWriter.Close()
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to stream backup: %w", err)
	}

	if err := zstdWriter.Close(); err != nil {
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to close zstd writer: %w", err)
	}

	parts, err := chunkWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close chunk writer: %w", err)
	}

	manifest := &SnapshotManifest{
		Network: network,
		ChainID: chainID,
		Base: SnapshotEntry{
			Height: height,
			Since:  0,
			Parts:  parts,
		},
		Incrementals: []SnapshotEntry{},
		StateRoot:    stateRoot,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		LastVersion:  lastVersion,
	}

	if err := sm.writeManifest(snapshotDir, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// CreateIncrementalSnapshot creates an incremental snapshot using streaming chunking
func (sm *SnapshotManager) CreateIncrementalSnapshot(
	network string,
	chainID uint64,
	db database.Database,
	parent *SnapshotManifest,
	snapshotID string,
) (*SnapshotManifest, error) {

	if snapshotID == "" {
		snapshotID = time.Now().Format("2006-01-02")
	}
	snapshotDir := filepath.Join(sm.baseDir, "snapshots", snapshotID, network, fmt.Sprintf("chain_%d", chainID))
	chunksDir := filepath.Join(snapshotDir, "chunks")

	if err := os.MkdirAll(chunksDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %w", err)
	}

	// For a self-contained snapshot, we need to ensure parent parts are available.
	// We can hardlink them from the parent's directory.
	parentDir, err := sm.GetLatestSnapshotDir(network, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to locate parent snapshot: %w", err)
	}
	parentChunksDir := filepath.Join(parentDir, "chunks")

	// Only copy/link parts if we're writing to a different directory
	// If same directory, parts already exist
	if parentChunksDir != chunksDir {
		linkParts := func(parts []Part) error {
			for _, part := range parts {
				src := filepath.Join(parentChunksDir, part.Name)
				dst := filepath.Join(chunksDir, part.Name)
				// Skip if already exists
				if _, err := os.Stat(dst); err == nil {
					continue
				}
				if err := os.Link(src, dst); err != nil {
					if err := copyFile(src, dst); err != nil {
						return err
					}
				}
			}
			return nil
		}

		if err := linkParts(parent.Base.Parts); err != nil {
			return nil, err
		}
		for _, inc := range parent.Incrementals {
			if err := linkParts(inc.Parts); err != nil {
				return nil, err
			}
		}
	}

	// Create New Incremental
	incPrefix := fmt.Sprintf("inc_%d_%d", parent.LastVersion, time.Now().Unix())

	chunkWriter, err := newChunkWriter(chunksDir, incPrefix, ChunkSize)
	if err != nil {
		return nil, err
	}

	zstdWriter, err := zstd.NewWriter(chunkWriter, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		chunkWriter.Close()
		return nil, err
	}

	newVersion, err := db.Backup(zstdWriter, parent.LastVersion)
	if err != nil {
		zstdWriter.Close()
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to stream incremental backup: %w", err)
	}

	if err := zstdWriter.Close(); err != nil {
		chunkWriter.Close()
		return nil, err
	}

	parts, err := chunkWriter.Close()
	if err != nil {
		return nil, err
	}

	// Update Manifest
	manifest := &SnapshotManifest{
		Network: network,
		ChainID: chainID,
		Base:    parent.Base,
		Incrementals: append(parent.Incrementals, SnapshotEntry{
			Height: 0,
			Since:  parent.LastVersion,
			Parts:  parts,
		}),
		StateRoot:   parent.StateRoot,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		LastVersion: newVersion,
	}

	if err := sm.writeManifest(snapshotDir, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// CreateChainDataSnapshot creates a snapshot for a specific chain's data directory
func (sm *SnapshotManager) CreateChainDataSnapshot(
	network string,
	nodeID uint64,
	chainDataID string,
	db database.Database,
	snapshotID string,
) (*SnapshotManifest, error) {
	if snapshotID == "" {
		snapshotID = time.Now().Format("2006-01-02")
	}

	// Store chainData snapshots with pattern: chaindata_<nodeID>_<chainDataID[:16]>
	dirName := fmt.Sprintf("chaindata_%d_%s", nodeID, chainDataID[:16])
	snapshotDir := filepath.Join(sm.baseDir, "snapshots", snapshotID, network, dirName)
	chunksDir := filepath.Join(snapshotDir, "chunks")

	if err := os.MkdirAll(chunksDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %w", err)
	}

	backupPrefix := fmt.Sprintf("chaindata_%d", nodeID)

	chunkWriter, err := newChunkWriter(chunksDir, backupPrefix, ChunkSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk writer: %w", err)
	}

	zstdWriter, err := zstd.NewWriter(chunkWriter, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to create zstd writer: %w", err)
	}

	lastVersion, err := db.Backup(zstdWriter, 0)
	if err != nil {
		zstdWriter.Close()
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to stream backup: %w", err)
	}

	if err := zstdWriter.Close(); err != nil {
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to close zstd writer: %w", err)
	}

	parts, err := chunkWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close chunk writer: %w", err)
	}

	manifest := &SnapshotManifest{
		Network:     network,
		NodeID:      nodeID,
		ChainDataID: chainDataID, // Full chain ID for restore
		Base: SnapshotEntry{
			Height: 0,
			Since:  0,
			Parts:  parts,
		},
		Incrementals: []SnapshotEntry{},
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		LastVersion:  lastVersion,
	}

	if err := sm.writeManifest(snapshotDir, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// CreateIncrementalChainDataSnapshot creates an incremental snapshot for chainData
func (sm *SnapshotManager) CreateIncrementalChainDataSnapshot(
	network string,
	nodeID uint64,
	chainDataID string,
	db database.Database,
	parent *SnapshotManifest,
	snapshotID string,
) (*SnapshotManifest, error) {
	if snapshotID == "" {
		snapshotID = time.Now().Format("2006-01-02")
	}

	dirName := fmt.Sprintf("chaindata_%d_%s", nodeID, chainDataID[:16])
	snapshotDir := filepath.Join(sm.baseDir, "snapshots", snapshotID, network, dirName)
	chunksDir := filepath.Join(snapshotDir, "chunks")

	if err := os.MkdirAll(chunksDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %w", err)
	}

	// Link parent parts if different directory
	parentDir, err := sm.GetLatestChainDataSnapshotDir(network, nodeID, chainDataID)
	if err == nil {
		parentChunksDir := filepath.Join(parentDir, "chunks")
		if parentChunksDir != chunksDir {
			linkParts := func(parts []Part) error {
				for _, part := range parts {
					src := filepath.Join(parentChunksDir, part.Name)
					dst := filepath.Join(chunksDir, part.Name)
					if _, err := os.Stat(dst); err == nil {
						continue
					}
					if err := os.Link(src, dst); err != nil {
						if err := copyFile(src, dst); err != nil {
							return err
						}
					}
				}
				return nil
			}
			linkParts(parent.Base.Parts)
			for _, inc := range parent.Incrementals {
				linkParts(inc.Parts)
			}
		}
	}

	incPrefix := fmt.Sprintf("chaindata_%d_inc_%d", nodeID, time.Now().Unix())

	chunkWriter, err := newChunkWriter(chunksDir, incPrefix, ChunkSize)
	if err != nil {
		return nil, err
	}

	zstdWriter, err := zstd.NewWriter(chunkWriter, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		chunkWriter.Close()
		return nil, err
	}

	newVersion, err := db.Backup(zstdWriter, parent.LastVersion)
	if err != nil {
		zstdWriter.Close()
		chunkWriter.Close()
		return nil, fmt.Errorf("failed to stream incremental backup: %w", err)
	}

	if err := zstdWriter.Close(); err != nil {
		chunkWriter.Close()
		return nil, err
	}

	parts, err := chunkWriter.Close()
	if err != nil {
		return nil, err
	}

	manifest := &SnapshotManifest{
		Network:     network,
		NodeID:      nodeID,
		ChainDataID: chainDataID,
		Base:        parent.Base,
		Incrementals: append(parent.Incrementals, SnapshotEntry{
			Height: 0,
			Since:  parent.LastVersion,
			Parts:  parts,
		}),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		LastVersion: newVersion,
	}

	if err := sm.writeManifest(snapshotDir, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// GetLatestChainDataManifest finds the most recent manifest for a chainData snapshot
func (sm *SnapshotManager) GetLatestChainDataManifest(network string, nodeID uint64, chainDataID string) (*SnapshotManifest, error) {
	snapshotRoot := filepath.Join(sm.baseDir, "snapshots")
	entries, err := os.ReadDir(snapshotRoot)
	if err != nil {
		return nil, err
	}
	dirName := fmt.Sprintf("chaindata_%d_%s", nodeID, chainDataID[:16])
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(snapshotRoot, entry.Name(), network, dirName, "manifest.json")
		if _, err := os.Stat(manifestPath); err == nil {
			data, err := os.ReadFile(manifestPath)
			if err == nil {
				var m SnapshotManifest
				if err := json.Unmarshal(data, &m); err == nil {
					return &m, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no chaindata manifest found")
}

// GetLatestChainDataSnapshotDir finds the most recent snapshot directory for chainData
func (sm *SnapshotManager) GetLatestChainDataSnapshotDir(network string, nodeID uint64, chainDataID string) (string, error) {
	snapshotRoot := filepath.Join(sm.baseDir, "snapshots")
	entries, err := os.ReadDir(snapshotRoot)
	if err != nil {
		return "", err
	}
	dirName := fmt.Sprintf("chaindata_%d_%s", nodeID, chainDataID[:16])
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(snapshotRoot, entry.Name(), network, dirName)
		if _, err := os.Stat(filepath.Join(path, "manifest.json")); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no chaindata snapshot found")
}

// RestoreChainSnapshot restores a snapshot using streaming from chunks
func (sm *SnapshotManager) RestoreChainSnapshot(
	network string,
	chainID uint64,
	manifest *SnapshotManifest,
	dbDir string,
	snapshotID string,
) error {

	// Clear existing database - BadgerDB Load requires empty database
	if _, err := os.Stat(dbDir); err == nil {
		if err := os.RemoveAll(dbDir); err != nil {
			return fmt.Errorf("failed to clear existing db: %w", err)
		}
	}

	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := badgerdb.New(dbDir, nil, "", nil)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}
	defer db.Close()

	chainDir := filepath.Join(sm.baseDir, "snapshots", snapshotID, network, fmt.Sprintf("chain_%d", chainID))
	chunksDir := filepath.Join(chainDir, "chunks")

	// Restore Base
	if err := sm.loadFromParts(db, chunksDir, manifest.Base.Parts); err != nil {
		return fmt.Errorf("failed to restore base: %w", err)
	}

	// Restore Incrementals
	for _, inc := range manifest.Incrementals {
		if err := sm.loadFromParts(db, chunksDir, inc.Parts); err != nil {
			return fmt.Errorf("failed to restore incremental: %w", err)
		}
	}

	ux.Logger.PrintToUser("🧹 Optimizing database...")
	if err := db.Compact(nil, nil); err != nil {
		ux.Logger.PrintToUser("Warning: Compact failed: %v", err)
	}

	ux.Logger.PrintToUser("✅ Restored snapshot to %s", dbDir)
	return nil
}

// loadFromParts streams chunks -> MultiReader -> zstd -> db.Load
func (sm *SnapshotManager) loadFromParts(db database.Database, chunksDir string, parts []Part) error {
	if len(parts) == 0 {
		return nil
	}

	partPaths := make([]string, len(parts))
	for i, part := range parts {
		partPaths[i] = filepath.Join(chunksDir, part.Name)
	}

	// Sort by name ensures correct order (assuming part%05d naming)
	sort.Strings(partPaths)

	ux.Logger.PrintToUser("📥 Restoring from %s (%d parts)", parts[0].Name, len(parts))

	files := make([]*os.File, 0, len(partPaths))
	readers := make([]io.Reader, 0, len(partPaths))
	for _, p := range partPaths {
		f, err := os.Open(p)
		if err != nil {
			for _, ff := range files {
				_ = ff.Close()
			}
			return err
		}
		files = append(files, f)
		readers = append(readers, f)
	}
	defer func() {
		for _, f := range files {
			_ = f.Close()
		}
	}()

	compressed := io.MultiReader(readers...)
	zr, err := zstd.NewReader(compressed)
	if err != nil {
		return err
	}
	defer zr.Close()

	if err := db.Load(zr); err != nil {
		return fmt.Errorf("db load failed: %w", err)
	}
	return nil
}

// Squash combines base + incrementals into a new base
func (sm *SnapshotManager) Squash(network string, chainID uint64, snapshotName string) error {
	ux.Logger.PrintToUser("Squashing snapshots for %s chain %d in %s...", network, chainID, snapshotName)

	snapshotRoot := filepath.Join(sm.baseDir, "snapshots", snapshotName)
	chainDir := filepath.Join(snapshotRoot, network, fmt.Sprintf("chain_%d", chainID))
	manifestPath := filepath.Join(chainDir, "manifest.json")
	chunksDir := filepath.Join(chainDir, "chunks")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}
	var manifest SnapshotManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	if len(manifest.Incrementals) == 0 {
		ux.Logger.PrintToUser("No incrementals to squash.")
		return nil
	}

	tempDir, err := os.MkdirTemp("", "lux-squash-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	db, err := badgerdb.New(tempDir, nil, "", nil)
	if err != nil {
		return fmt.Errorf("failed to open temp db: %w", err)
	}

	// Restore to temp using streaming
	if err := sm.loadFromParts(db, chunksDir, manifest.Base.Parts); err != nil {
		db.Close()
		return err
	}
	for _, inc := range manifest.Incrementals {
		if err := sm.loadFromParts(db, chunksDir, inc.Parts); err != nil {
			db.Close()
			return err
		}
	}

	// Optimize
	if err := db.Compact(nil, nil); err != nil {
		ux.Logger.PrintToUser("Warning: Compact failed: %v", err)
	}

	// Create new Base
	newBasePrefix := fmt.Sprintf("base_%d_squashed_%d", 0, time.Now().Unix())

	chunkWriter, err := newChunkWriter(chunksDir, newBasePrefix, ChunkSize)
	if err != nil {
		db.Close()
		return err
	}

	zstdWriter, err := zstd.NewWriter(chunkWriter, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		chunkWriter.Close()
		db.Close()
		return err
	}

	lastVersion, err := db.Backup(zstdWriter, 0)
	zstdWriter.Close()
	parts, _ := chunkWriter.Close()
	db.Close()

	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Cleanup old files
	// Note: Careful if files are hardlinked shared with other snapshots.
	// Current architecture implies self-contained (hardlinked) dir.
	// Unlinking here affects this snapshot dir only.
	oldEntries := append([]SnapshotEntry{manifest.Base}, manifest.Incrementals...)
	for _, entry := range oldEntries {
		for _, part := range entry.Parts {
			os.Remove(filepath.Join(chunksDir, part.Name))
		}
	}

	// Update Manifest
	manifest.Base = SnapshotEntry{
		Height: 0,
		Since:  0,
		Parts:  parts,
	}
	manifest.Incrementals = []SnapshotEntry{}
	manifest.LastVersion = lastVersion
	manifest.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	return sm.writeManifest(chainDir, &manifest)
}

// ... existing helpers ...
func (sm *SnapshotManager) GetLatestManifest(network string, chainID uint64) (*SnapshotManifest, error) {
	snapshotRoot := filepath.Join(sm.baseDir, "snapshots")
	entries, err := os.ReadDir(snapshotRoot)
	if err != nil {
		return nil, err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(snapshotRoot, entry.Name(), network, fmt.Sprintf("chain_%d", chainID), "manifest.json")
		if _, err := os.Stat(manifestPath); err == nil {
			data, err := os.ReadFile(manifestPath)
			if err == nil {
				var m SnapshotManifest
				if err := json.Unmarshal(data, &m); err == nil {
					return &m, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no manifest found")
}

func (sm *SnapshotManager) GetLatestSnapshotDir(network string, chainID uint64) (string, error) {
	snapshotRoot := filepath.Join(sm.baseDir, "snapshots")
	entries, err := os.ReadDir(snapshotRoot)
	if err != nil {
		return "", err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(snapshotRoot, entry.Name(), network, fmt.Sprintf("chain_%d", chainID))
		if _, err := os.Stat(filepath.Join(path, "manifest.json")); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no snapshot found")
}

func (sm *SnapshotManager) writeManifest(dir string, manifest *SnapshotManifest) error {
	manifestFile := filepath.Join(dir, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestFile, manifestData, 0o644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// RestoreSnapshot restores a full snapshot (all networks/nodes)
// Handles both main DB (chain_*) and chainData (chaindata_*) directories
func (sm *SnapshotManager) RestoreSnapshot(snapshotName string) error {
	ux.Logger.PrintToUser("Restoring snapshot '%s'...", snapshotName)
	snapshotRoot := filepath.Join(sm.baseDir, "snapshots", snapshotName)
	if _, err := os.Stat(snapshotRoot); os.IsNotExist(err) {
		return fmt.Errorf("snapshot not found: %s", snapshotName)
	}
	netEntries, err := os.ReadDir(snapshotRoot)
	if err != nil {
		return err
	}
	for _, netEntry := range netEntries {
		if !netEntry.IsDir() {
			continue
		}
		networkName := netEntry.Name()
		netDir := filepath.Join(snapshotRoot, networkName)

		// Find current run directory (shared by all restores)
		runsDir := filepath.Join(sm.baseDir, "runs", networkName)
		currentLink := filepath.Join(runsDir, "current")
		runDir := ""
		if target, err := os.Readlink(currentLink); err == nil {
			runDir = filepath.Join(runsDir, target)
		} else {
			runEntries, _ := os.ReadDir(runsDir)
			for _, re := range runEntries {
				if re.IsDir() && strings.HasPrefix(re.Name(), "run_") {
					runDir = filepath.Join(runsDir, re.Name())
				}
			}
		}
		if runDir == "" {
			ux.Logger.PrintToUser("Skipping %s: no run directory found", networkName)
			continue
		}

		entries, _ := os.ReadDir(netDir)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			entryName := entry.Name()

			manifestPath := filepath.Join(netDir, entryName, "manifest.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
			var manifest SnapshotManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				continue
			}

			// === Restore Main DB (chain_<nodeID>) ===
			if strings.HasPrefix(entryName, "chain_") {
				nodeIDStr := strings.TrimPrefix(entryName, "chain_")
				nodeID, _ := strconv.ParseUint(nodeIDStr, 10, 64)

				targetNodeDir := filepath.Join(runDir, fmt.Sprintf("node%d", nodeID))
				targetDBPath := filepath.Join(targetNodeDir, "db", networkName, "db")
				dbPattern := filepath.Join(targetNodeDir, "db", "*", "db")
				matches, _ := filepath.Glob(dbPattern)
				if len(matches) > 0 {
					targetDBPath = matches[0]
				}

				if err := sm.RestoreChainSnapshot(networkName, nodeID, &manifest, targetDBPath, snapshotName); err != nil {
					return fmt.Errorf("failed to restore %s/node%d main DB: %w", networkName, nodeID, err)
				}
				ux.Logger.PrintToUser("✓ Restored %s/node%d main DB", networkName, nodeID)
			}

			// === Restore ChainData (chaindata_<nodeID>_<chainID>) ===
			if strings.HasPrefix(entryName, "chaindata_") && manifest.ChainDataID != "" {
				nodeID := manifest.NodeID
				chainDataID := manifest.ChainDataID

				// Target: runs/<net>/run_*/node<N>/chainData/network-<N>/<chainID>/db/badgerdb
				targetNodeDir := filepath.Join(runDir, fmt.Sprintf("node%d", nodeID))

				// Find network-* subdirectory
				chainDataBase := filepath.Join(targetNodeDir, "chainData")
				networkDirs, _ := filepath.Glob(filepath.Join(chainDataBase, "network-*"))
				if len(networkDirs) == 0 {
					ux.Logger.PrintToUser("Skipping chaindata %s: no network-* dir", chainDataID[:8])
					continue
				}

				// Use first network dir (should only be one)
				networkDir := networkDirs[0]
				targetDBPath := filepath.Join(networkDir, chainDataID, "db", "badgerdb")

				if err := sm.RestoreChainDataSnapshot(&manifest, targetDBPath, snapshotName, entryName); err != nil {
					return fmt.Errorf("failed to restore chaindata %s: %w", chainDataID[:8], err)
				}
				ux.Logger.PrintToUser("✓ Restored %s/node%d chain %s", networkName, nodeID, chainDataID[:8])
			}
		}
	}
	return nil
}

// RestoreChainDataSnapshot restores a chainData snapshot
func (sm *SnapshotManager) RestoreChainDataSnapshot(
	manifest *SnapshotManifest,
	dbDir string,
	snapshotID string,
	entryName string,
) error {
	// Clear existing database
	if _, err := os.Stat(dbDir); err == nil {
		if err := os.RemoveAll(dbDir); err != nil {
			return fmt.Errorf("failed to clear existing db: %w", err)
		}
	}

	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := badgerdb.New(dbDir, nil, "", nil)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}
	defer db.Close()

	chainDir := filepath.Join(sm.baseDir, "snapshots", snapshotID, manifest.Network, entryName)
	chunksDir := filepath.Join(chainDir, "chunks")

	// Restore base
	if err := sm.loadFromParts(db, chunksDir, manifest.Base.Parts); err != nil {
		return fmt.Errorf("failed to restore base: %w", err)
	}

	// Restore incrementals
	for _, inc := range manifest.Incrementals {
		if err := sm.loadFromParts(db, chunksDir, inc.Parts); err != nil {
			return fmt.Errorf("failed to restore incremental: %w", err)
		}
	}

	return nil
}

// SnapshotInfo contains metadata about a snapshot
type SnapshotInfo struct {
	Name        string
	Path        string
	Size        int64
	Incremental bool
	Created     time.Time
}

// GetSnapshotInfo returns information about a specific snapshot
func (sm *SnapshotManager) GetSnapshotInfo(snapshotName string) (*SnapshotInfo, error) {
	snapshotRoot := filepath.Join(sm.baseDir, "snapshots", snapshotName)
	if _, err := os.Stat(snapshotRoot); os.IsNotExist(err) {
		// Try lux-snapshot- prefix
		snapshotRoot = filepath.Join(sm.baseDir, "snapshots", "lux-snapshot-"+snapshotName)
		if _, err := os.Stat(snapshotRoot); os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot not found: %s", snapshotName)
		}
	}

	info := &SnapshotInfo{
		Name: snapshotName,
		Path: snapshotRoot,
	}

	// Calculate total size
	filepath.WalkDir(snapshotRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err == nil {
			info.Size += fi.Size()
		}
		return nil
	})

	// Get creation time from directory
	fi, err := os.Stat(snapshotRoot)
	if err == nil {
		info.Created = fi.ModTime()
	}

	// Check if incremental by looking for manifest
	manifestPath := filepath.Join(snapshotRoot, "manifest.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest SnapshotManifest
		if json.Unmarshal(data, &manifest) == nil {
			info.Incremental = len(manifest.Incrementals) > 0
		}
	}

	return info, nil
}

// ListSnapshots returns a list of all available snapshots
func (sm *SnapshotManager) ListSnapshots() ([]*SnapshotInfo, error) {
	snapshotsDir := filepath.Join(sm.baseDir, "snapshots")
	if _, err := os.Stat(snapshotsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return nil, err
	}

	var snapshots []*SnapshotInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip internal directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		// Strip lux-snapshot- prefix for display
		displayName := strings.TrimPrefix(name, "lux-snapshot-")

		info, err := sm.GetSnapshotInfo(displayName)
		if err == nil {
			info.Name = displayName
			snapshots = append(snapshots, info)
		}
	}

	return snapshots, nil
}
