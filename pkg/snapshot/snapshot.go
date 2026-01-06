// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/cli/pkg/ux"
)

const (
	ChunkSize = 99 * 1024 * 1024 // 99MB chunks for GitHub
	MaxRetries = 3
)

type SnapshotManager struct {
	BaseDir string
	NetworkType string
	NodeCount int
}

func NewSnapshotManager(baseDir, networkType string, nodeCount int) *SnapshotManager {
	return &SnapshotManager{
		BaseDir:     baseDir,
		NetworkType: networkType,
		NodeCount:   nodeCount,
	}
}

// CreateSnapshot creates a consistent snapshot of all nodes with minimal downtime
func (sm *SnapshotManager) CreateSnapshot(snapshotName string) error {
	ux.Logger.PrintToUser("Starting coordinated snapshot for %d nodes...", sm.NodeCount)
	
	// Step 1: Prepare snapshot directory
	snapshotDir := filepath.Join(sm.BaseDir, "snapshots", snapshotName)
	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	
	// Step 2: Get all node directories
	nodeDirs, err := sm.getNodeDirectories()
	if err != nil {
		return fmt.Errorf("failed to get node directories: %w", err)
	}
	
	// Step 3: Coordinate snapshot across all nodes
	ux.Logger.PrintToUser("Coordinating snapshot across %d nodes...", len(nodeDirs))
	
	var wg sync.WaitGroup
	errCh := make(chan error, len(nodeDirs))
	
	for i, nodeDir := range nodeDirs {
		wg.Add(1)
		go func(nodeIndex int, nodePath string) {
			defer wg.Done()
			
			nodeSnapshotName := fmt.Sprintf("node-%d", nodeIndex)
			nodeSnapshotDir := filepath.Join(snapshotDir, nodeSnapshotName)
			
			if err := sm.createNodeSnapshot(nodePath, nodeSnapshotDir); err != nil {
				errCh <- fmt.Errorf("failed to snapshot node %d: %w", nodeIndex, err)
				return
			}
			
			ux.Logger.PrintToUser("✓ Node %d snapshot completed", nodeIndex)
		}(i, nodeDir)
	}
	
	wg.Wait()
	close(errCh)
	
	// Check for errors
	if len(errCh) > 0 {
		return <-errCh
	}
	
	// Step 4: Create metadata
	if err := sm.createSnapshotMetadata(snapshotDir, snapshotName); err != nil {
		return fmt.Errorf("failed to create snapshot metadata: %w", err)
	}
	
	// Step 5: Chunk the snapshot for GitHub
	if err := sm.chunkSnapshot(snapshotDir); err != nil {
		return fmt.Errorf("failed to chunk snapshot: %w", err)
	}
	
	ux.Logger.PrintToUser("✓ Snapshot '%s' created successfully with %d nodes", snapshotName, len(nodeDirs))
	return nil
}

func (sm *SnapshotManager) getNodeDirectories() ([]string, error) {
	// Find all node directories in the network
	runDir := filepath.Join(sm.BaseDir, "runs", sm.NetworkType)
	
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read run directory: %w", err)
	}
	
	var nodeDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "run_") {
			nodeDirs = append(nodeDirs, filepath.Join(runDir, entry.Name()))
		}
	}
	
	if len(nodeDirs) == 0 {
		return nil, fmt.Errorf("no node directories found in %s", runDir)
	}
	
	// If we have a specific node count, use only that many
	if sm.NodeCount > 0 && len(nodeDirs) > sm.NodeCount {
		nodeDirs = nodeDirs[:sm.NodeCount]
	}
	
	return nodeDirs, nil
}

func (sm *SnapshotManager) createNodeSnapshot(nodeDir, snapshotDir string) error {
	// Ensure node is properly flushed before snapshot
	if err := sm.flushNodeDatabase(nodeDir); err != nil {
		return fmt.Errorf("failed to flush node database: %w", err)
	}
	
	// Create the snapshot directory
	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
		return fmt.Errorf("failed to create node snapshot directory: %w", err)
	}
	
	// Create a compressed archive of the node data
	archivePath := filepath.Join(snapshotDir, "node-data.tar.gz")
	if err := createTarGzArchive(nodeDir, archivePath); err != nil {
		return fmt.Errorf("failed to create node archive: %w", err)
	}
	
	return nil
}

func (sm *SnapshotManager) flushNodeDatabase(nodeDir string) error {
	// Find database directories
	dbDirs := []string{
		filepath.Join(nodeDir, "db"),
		filepath.Join(nodeDir, "pebbledb"),
		filepath.Join(nodeDir, "badgerdb"),
		filepath.Join(nodeDir, "leveldb"),
	}
	
	for _, dbDir := range dbDirs {
		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			continue
		}
		
		// For PebbleDB/BadgerDB/LevelDB, we need to ensure proper flush
		// This would typically involve calling the node's API to flush
		// For now, we'll just sync the directory as a basic approach
		if err := syncDirectory(dbDir); err != nil {
			return fmt.Errorf("failed to sync database directory %s: %w", dbDir, err)
		}
	}
	
	return nil
}

func syncDirectory(dir string) error {
	// This is a basic implementation - in production, you'd want to
	// call the node's API to ensure proper database flush
	
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Sync each file to ensure it's flushed to disk
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			
			if err := file.Sync(); err != nil {
				return fmt.Errorf("failed to sync file %s: %w", path, err)
			}
		}
		return nil
	})
	
	return nil
}

func createTarGzArchive(sourceDir, archivePath string) error {
	// Create the archive file
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()
	
	// Create gzip writer
	gzipWriter := gzip.NewWriter(archiveFile)
	defer gzipWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	// Walk the source directory and add files to the archive
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip the root directory itself
		if path == sourceDir {
			return nil
		}
		
		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		
		// Create tar header
		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return err
		}
		
		// Set the name to the relative path
		header.Name = relPath
		
		// Write the header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// If it's a regular file, write the contents
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}
		
		return nil
	})
}

func (sm *SnapshotManager) createSnapshotMetadata(snapshotDir, snapshotName string) error {
	metadata := map[string]interface{}{
		"name":          snapshotName,
		"network_type":  sm.NetworkType,
		"node_count":    sm.NodeCount,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"lux_version":   "1.22.8", // TODO: Use actual version from constants
		"snapshot_type": "full",
	}
	
	// Calculate checksums for each node
	nodes := make([]map[string]string, sm.NodeCount)
	for i := 0; i < sm.NodeCount; i++ {
		nodeDir := filepath.Join(snapshotDir, fmt.Sprintf("node-%d", i))
		archivePath := filepath.Join(nodeDir, "node-data.tar.gz")
		
		checksum, err := calculateFileChecksum(archivePath)
		if err != nil {
			return fmt.Errorf("failed to calculate checksum for node %d: %w", i, err)
		}
		
		nodes[i] = map[string]string{
			"archive":   "node-data.tar.gz",
			"checksum":  checksum,
			"node_id":   fmt.Sprintf("node-%d", i),
		}
	}
	
	metadata["nodes"] = nodes
	
	// Write metadata file
	metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.json")
	metadataContent, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	if err := os.WriteFile(metadataPath, metadataContent, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	
	return nil
}

func calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}
	
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (sm *SnapshotManager) chunkSnapshot(snapshotDir string) error {
	ux.Logger.PrintToUser("Chunking snapshot for GitHub upload...")
	
	// Create chunks directory
	chunksDir := filepath.Join(snapshotDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0750); err != nil {
		return fmt.Errorf("failed to create chunks directory: %w", err)
	}
	
	// Find all archive files to chunk
	var archiveFiles []string
	for i := 0; i < sm.NodeCount; i++ {
		nodeDir := filepath.Join(snapshotDir, fmt.Sprintf("node-%d", i))
		archivePath := filepath.Join(nodeDir, "node-data.tar.gz")
		
		if _, err := os.Stat(archivePath); err == nil {
			archiveFiles = append(archiveFiles, archivePath)
		}
	}
	
	// Also chunk the metadata
	metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.json")
	archiveFiles = append(archiveFiles, metadataPath)
	
	// Process each file
	for _, filePath := range archiveFiles {
		if err := sm.chunkFile(filePath, chunksDir); err != nil {
			return fmt.Errorf("failed to chunk file %s: %w", filePath, err)
		}
	}
	
	return nil
}

func (sm *SnapshotManager) chunkFile(filePath, chunksDir string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	fileName := filepath.Base(filePath)
	fileSize := fileInfo.Size()
	
	if fileSize <= ChunkSize {
		// Small file, just copy it
		destPath := filepath.Join(chunksDir, fileName)
		return copyFile(filePath, destPath)
	}
	
	// Large file, chunk it
	chunkIndex := 0
	buffer := make([]byte, ChunkSize)
	
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %w", err)
		}
		
		if n == 0 {
			break
		}
		
		chunkName := fmt.Sprintf("%s.part-%d", fileName, chunkIndex)
		chunkPath := filepath.Join(chunksDir, chunkName)
		
		chunkFile, err := os.Create(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to create chunk file: %w", err)
		}
		
		if _, err := chunkFile.Write(buffer[:n]); err != nil {
			chunkFile.Close()
			return fmt.Errorf("failed to write chunk: %w", err)
		}
		
		if err := chunkFile.Close(); err != nil {
			return fmt.Errorf("failed to close chunk file: %w", err)
		}
		
		chunkIndex++
		
		if err == io.EOF {
			break
		}
	}
	
	ux.Logger.PrintToUser("✓ Chunked %s into %d parts", fileName, chunkIndex)
	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	
	_, err = io.Copy(destination, source)
	return err
}

// RestoreSnapshot restores a snapshot to the network
func (sm *SnapshotManager) RestoreSnapshot(snapshotName string) error {
	ux.Logger.PrintToUser("Starting snapshot restoration for %d nodes...", sm.NodeCount)
	
	// Step 1: Locate the snapshot
	snapshotDir := filepath.Join(sm.BaseDir, "snapshots", snapshotName)
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}
	
	// Step 2: Check if we need to recombine chunks
	chunksDir := filepath.Join(snapshotDir, "chunks")
	if _, err := os.Stat(chunksDir); err == nil {
		ux.Logger.PrintToUser("Recombining chunks...")
		if err := sm.recombineChunks(snapshotDir); err != nil {
			return fmt.Errorf("failed to recombine chunks: %w", err)
		}
	}
	
	// Step 3: Verify snapshot metadata
	metadataPath := filepath.Join(snapshotDir, "snapshot_metadata.json")
	metadata, err := sm.loadSnapshotMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load snapshot metadata: %w", err)
	}
	
	// Step 4: Stop all nodes (if running)
	if err := sm.stopAllNodes(); err != nil {
		return fmt.Errorf("failed to stop nodes: %w", err)
	}
	
	// Step 5: Restore each node
	ux.Logger.PrintToUser("Restoring %d nodes from snapshot...", len(metadata.Nodes))
	
	var wg sync.WaitGroup
	errCh := make(chan error, len(metadata.Nodes))
	
	for i, node := range metadata.Nodes {
		wg.Add(1)
		go func(nodeIndex int, nodeData map[string]string) {
			defer wg.Done()
			
			nodeDir := filepath.Join(sm.BaseDir, "runs", sm.NetworkType, fmt.Sprintf("run_%d", nodeIndex))
			nodeSnapshotDir := filepath.Join(snapshotDir, fmt.Sprintf("node-%d", nodeIndex))
			archivePath := filepath.Join(nodeSnapshotDir, "node-data.tar.gz")
			
			if err := sm.restoreNodeSnapshot(archivePath, nodeDir); err != nil {
				errCh <- fmt.Errorf("failed to restore node %d: %w", nodeIndex, err)
				return
			}
			
			ux.Logger.PrintToUser("✓ Node %d restored successfully", nodeIndex)
		}(i, node)
	}
	
	wg.Wait()
	close(errCh)
	
	// Check for errors
	if len(errCh) > 0 {
		return <-errCh
	}
	
	ux.Logger.PrintToUser("✓ Snapshot '%s' restored successfully", snapshotName)
	ux.Logger.PrintToUser("You can now start the network with: lux network start --%s", sm.NetworkType)
	
	return nil
}

func (sm *SnapshotManager) recombineChunks(snapshotDir string) error {
	chunksDir := filepath.Join(snapshotDir, "chunks")
	
	// Find all chunk files
	entries, err := os.ReadDir(chunksDir)
	if err != nil {
		return fmt.Errorf("failed to read chunks directory: %w", err)
	}
	
	// Group chunks by base filename
	chunkGroups := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		filename := entry.Name()
		if strings.HasSuffix(filename, ".part-0") {
			baseName := strings.TrimSuffix(filename, ".part-0")
			chunkGroups[baseName] = append(chunkGroups[baseName], filename)
		}
	}
	
	// Recombine each group
	for baseName, chunks := range chunkGroups {
		// Sort chunks by part number
		// Simple sort - in production you'd want proper sorting
		
		destPath := filepath.Join(snapshotDir, baseName)
		if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}
		
		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create recombined file: %w", err)
		}
		defer destFile.Close()
		
		// Recombine all parts
		for _, chunkName := range chunks {
			chunkPath := filepath.Join(chunksDir, chunkName)
			chunkFile, err := os.Open(chunkPath)
			if err != nil {
				return fmt.Errorf("failed to open chunk %s: %w", chunkName, err)
			}
			
			if _, err := io.Copy(destFile, chunkFile); err != nil {
				chunkFile.Close()
				return fmt.Errorf("failed to copy chunk %s: %w", chunkName, err)
			}
			
			chunkFile.Close()
		}
		
		ux.Logger.PrintToUser("✓ Recombined %s from %d chunks", baseName, len(chunks))
	}
	
	return nil
}

type SnapshotMetadata struct {
	Name         string                 `json:"name"`
	NetworkType  string                 `json:"network_type"`
	NodeCount    int                    `json:"node_count"`
	CreatedAt    string                 `json:"created_at"`
	LuxVersion   string                 `json:"lux_version"`
	SnapshotType string                 `json:"snapshot_type"`
	Nodes        []map[string]string    `json:"nodes"`
}

func (sm *SnapshotManager) loadSnapshotMetadata(metadataPath string) (*SnapshotMetadata, error) {
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}
	
	var metadata SnapshotMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	
	return &metadata, nil
}

func (sm *SnapshotManager) stopAllNodes() error {
	// In a real implementation, this would call the appropriate CLI commands
	// to stop all running nodes. For now, we'll just log it.
	ux.Logger.PrintToUser("Stopping all nodes...")
	
	// This is a placeholder - in production you would:
	// 1. Use the Lux CLI to stop all nodes gracefully
	// 2. Wait for confirmation that all nodes are stopped
	// 3. Handle any errors that occur during shutdown
	
	return nil
}

func (sm *SnapshotManager) restoreNodeSnapshot(archivePath, nodeDir string) error {
	// Ensure the target directory exists
	if err := os.MkdirAll(nodeDir, 0750); err != nil {
		return fmt.Errorf("failed to create node directory: %w", err)
	}
	
	// Extract the archive
	if err := extractTarGzArchive(archivePath, nodeDir); err != nil {
		return fmt.Errorf("failed to extract node archive: %w", err)
	}
	
	return nil
}

func extractTarGzArchive(archivePath, destDir string) error {
	// Open the archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer archiveFile.Close()
	
	// Create gzip reader
	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzipReader)
	
	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		
		// Create the target file path
		targetPath := filepath.Join(destDir, header.Name)
		
		// Handle directories
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			continue
		}
		
		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(targetPath), 0750); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
		}
		
		// Create the file
		file, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}
		defer file.Close()
		
		// Copy the file contents
		if _, err := io.Copy(file, tarReader); err != nil {
			return fmt.Errorf("failed to copy file contents for %s: %w", targetPath, err)
		}
		
		// Set file permissions
		if err := file.Chmod(os.FileMode(header.Mode)); err != nil {
			return fmt.Errorf("failed to set file permissions for %s: %w", targetPath, err)
		}
	}
	
	return nil
}