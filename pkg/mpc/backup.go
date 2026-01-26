// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpc

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/luxfi/cli/pkg/cloud/storage"
)

// BackupManifest contains metadata about an MPC backup.
type BackupManifest struct {
	Version       string            `json:"version"`
	NodeID        string            `json:"nodeId"`
	NodeName      string            `json:"nodeName"`
	Network       string            `json:"network"`
	Timestamp     time.Time         `json:"timestamp"`
	Checksums     map[string]string `json:"checksums"`
	DatabaseType  string            `json:"databaseType"` // badgerdb
	Incremental   bool              `json:"incremental"`
	BaseVersion   uint64            `json:"baseVersion,omitempty"`
	LatestVersion uint64            `json:"latestVersion"`
	WalletCount   int               `json:"walletCount"`
	KeyCount      int               `json:"keyCount"`
	Encryption    *EncryptionInfo   `json:"encryption,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// EncryptionInfo describes how the backup is encrypted.
type EncryptionInfo struct {
	Algorithm string `json:"algorithm"` // age, gpg, aes-256-gcm
	KeyID     string `json:"keyId,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
}

// BackupOptions configures backup behavior.
type BackupOptions struct {
	// Incremental creates a delta backup from the last version
	Incremental bool
	// BaseVersion for incremental backups
	BaseVersion uint64
	// Compression algorithm (zstd, gzip, none)
	Compression string
	// CompressionLevel (1-19 for zstd, 1-9 for gzip)
	CompressionLevel int
	// Encryption settings
	Encryption *EncryptionInfo
	// AgeRecipients for age encryption
	AgeRecipients []string
	// ChunkSize for splitting large backups (default 99MB for GitHub)
	ChunkSize int64
	// ProgressFunc reports backup progress
	ProgressFunc func(stage string, current, total int64)
	// Metadata to include in manifest
	Metadata map[string]string
}

// RestoreOptions configures restore behavior.
type RestoreOptions struct {
	// TargetPath to restore to (default: original location)
	TargetPath string
	// AgeIdentities for decryption
	AgeIdentities []string
	// VerifyOnly checks integrity without restoring
	VerifyOnly bool
	// ProgressFunc reports restore progress
	ProgressFunc func(stage string, current, total int64)
}

// BackupManager handles MPC node backups.
type BackupManager struct {
	storage   storage.Storage
	basePath  string
	nodeID    string
	nodeName  string
	network   string
}

// NewBackupManager creates a new backup manager.
func NewBackupManager(store storage.Storage, basePath, nodeID, nodeName, network string) *BackupManager {
	return &BackupManager{
		storage:  store,
		basePath: basePath,
		nodeID:   nodeID,
		nodeName: nodeName,
		network:  network,
	}
}

// CreateBackup creates a backup of the MPC node's BadgerDB.
func (bm *BackupManager) CreateBackup(ctx context.Context, dbPath string, opts *BackupOptions) (*BackupManifest, error) {
	if opts == nil {
		opts = &BackupOptions{
			Compression:      "zstd",
			CompressionLevel: 3,
			ChunkSize:        99 * 1024 * 1024, // 99MB
		}
	}

	timestamp := time.Now().UTC()
	backupName := fmt.Sprintf("%s_%s_%s", bm.nodeID, bm.network, timestamp.Format("20060102-150405"))

	manifest := &BackupManifest{
		Version:      "1.0.0",
		NodeID:       bm.nodeID,
		NodeName:     bm.nodeName,
		Network:      bm.network,
		Timestamp:    timestamp,
		Checksums:    make(map[string]string),
		DatabaseType: "badgerdb",
		Incremental:  opts.Incremental,
		BaseVersion:  opts.BaseVersion,
		Encryption:   opts.Encryption,
		Metadata:     opts.Metadata,
	}

	// Create temp directory for backup staging
	tempDir, err := os.MkdirTemp("", "mpc-backup-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Stage 1: Create tar archive of database
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("archiving", 0, 0)
	}

	tarPath := filepath.Join(tempDir, "data.tar")
	if err := bm.createTarArchive(dbPath, tarPath); err != nil {
		return nil, fmt.Errorf("failed to create tar archive: %w", err)
	}

	// Stage 2: Compress the archive
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("compressing", 0, 0)
	}

	compressedPath := tarPath
	switch opts.Compression {
	case "zstd":
		compressedPath = tarPath + ".zst"
		if err := bm.compressZstd(tarPath, compressedPath, opts.CompressionLevel); err != nil {
			return nil, fmt.Errorf("failed to compress with zstd: %w", err)
		}
	case "gzip":
		compressedPath = tarPath + ".gz"
		if err := bm.compressGzip(tarPath, compressedPath, opts.CompressionLevel); err != nil {
			return nil, fmt.Errorf("failed to compress with gzip: %w", err)
		}
	}

	// Stage 3: Encrypt if requested
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("encrypting", 0, 0)
	}

	finalPath := compressedPath
	if opts.Encryption != nil && len(opts.AgeRecipients) > 0 {
		finalPath = compressedPath + ".age"
		if err := bm.encryptWithAge(compressedPath, finalPath, opts.AgeRecipients); err != nil {
			return nil, fmt.Errorf("failed to encrypt: %w", err)
		}
	}

	// Compute checksum
	checksum, err := storage.ComputeChecksum(finalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute checksum: %w", err)
	}
	manifest.Checksums["data"] = checksum

	// Stage 4: Upload to storage
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("uploading", 0, 0)
	}

	dataKey := fmt.Sprintf("mpc/%s/%s/data%s", bm.network, backupName, filepath.Ext(finalPath))
	if err := bm.storage.UploadFile(ctx, dataKey, finalPath, &storage.UploadOptions{
		ContentType:  "application/octet-stream",
		StorageClass: "STANDARD",
		Metadata: map[string]string{
			"mpc-node-id":   bm.nodeID,
			"mpc-network":   bm.network,
			"mpc-timestamp": timestamp.Format(time.RFC3339),
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to upload backup data: %w", err)
	}

	// Stage 5: Upload manifest
	manifestKey := fmt.Sprintf("mpc/%s/%s/manifest.json", bm.network, backupName)
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(tempDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	if err := bm.storage.UploadFile(ctx, manifestKey, manifestPath, &storage.UploadOptions{
		ContentType: "application/json",
	}); err != nil {
		return nil, fmt.Errorf("failed to upload manifest: %w", err)
	}

	return manifest, nil
}

// ListBackups lists available backups.
func (bm *BackupManager) ListBackups(ctx context.Context) ([]BackupManifest, error) {
	prefix := fmt.Sprintf("mpc/%s/", bm.network)

	result, err := bm.storage.List(ctx, &storage.ListOptions{
		Prefix:    prefix,
		Delimiter: "/",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	var manifests []BackupManifest

	for _, dir := range result.CommonPrefixes {
		manifestKey := dir + "manifest.json"
		exists, err := bm.storage.Exists(ctx, manifestKey)
		if err != nil || !exists {
			continue
		}

		// Download and parse manifest
		var buf []byte
		if err := bm.storage.Download(ctx, manifestKey, &bytesWriter{buf: &buf}, nil); err != nil {
			continue
		}

		var manifest BackupManifest
		if err := json.Unmarshal(buf, &manifest); err != nil {
			continue
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// RestoreBackup restores from a backup.
func (bm *BackupManager) RestoreBackup(ctx context.Context, backupName string, opts *RestoreOptions) error {
	if opts == nil {
		opts = &RestoreOptions{}
	}

	// Download manifest
	manifestKey := fmt.Sprintf("mpc/%s/%s/manifest.json", bm.network, backupName)
	var manifestBuf []byte
	if err := bm.storage.Download(ctx, manifestKey, &bytesWriter{buf: &manifestBuf}, nil); err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(manifestBuf, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "mpc-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download backup data
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("downloading", 0, 0)
	}

	// Find the data file - check common compression extensions
	var dataKey string
	for _, ext := range []string{".zst", ".gz", ""} {
		key := fmt.Sprintf("mpc/%s/%s/data%s", bm.network, backupName, ext)
		exists, err := bm.storage.Exists(ctx, key)
		if err == nil && exists {
			dataKey = key
			break
		}
	}
	if dataKey == "" {
		return fmt.Errorf("backup data not found")
	}

	dataPath := filepath.Join(tempDir, "data"+filepath.Ext(dataKey))

	if err := bm.storage.DownloadFile(ctx, dataKey, dataPath, nil); err != nil {
		return fmt.Errorf("failed to download backup data: %w", err)
	}

	// Verify checksum
	checksum, err := storage.ComputeChecksum(dataPath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	if manifest.Checksums["data"] != "" && manifest.Checksums["data"] != checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", manifest.Checksums["data"], checksum)
	}

	if opts.VerifyOnly {
		return nil
	}

	// Decrypt if needed
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("decrypting", 0, 0)
	}

	currentPath := dataPath
	if filepath.Ext(dataPath) == ".age" {
		decryptedPath := dataPath[:len(dataPath)-4] // Remove .age
		if err := bm.decryptWithAge(dataPath, decryptedPath, opts.AgeIdentities); err != nil {
			return fmt.Errorf("failed to decrypt: %w", err)
		}
		currentPath = decryptedPath
	}

	// Decompress
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("decompressing", 0, 0)
	}

	tarPath := currentPath
	switch filepath.Ext(currentPath) {
	case ".zst":
		tarPath = currentPath[:len(currentPath)-4]
		if err := bm.decompressZstd(currentPath, tarPath); err != nil {
			return fmt.Errorf("failed to decompress zstd: %w", err)
		}
	case ".gz":
		tarPath = currentPath[:len(currentPath)-3]
		if err := bm.decompressGzip(currentPath, tarPath); err != nil {
			return fmt.Errorf("failed to decompress gzip: %w", err)
		}
	}

	// Extract tar archive
	if opts.ProgressFunc != nil {
		opts.ProgressFunc("extracting", 0, 0)
	}

	targetPath := opts.TargetPath
	if targetPath == "" {
		targetPath = bm.basePath
	}

	if err := bm.extractTarArchive(tarPath, targetPath); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	return nil
}

// DeleteBackup removes a backup.
func (bm *BackupManager) DeleteBackup(ctx context.Context, backupName string) error {
	prefix := fmt.Sprintf("mpc/%s/%s/", bm.network, backupName)

	result, err := bm.storage.List(ctx, &storage.ListOptions{
		Prefix: prefix,
	})
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	var keys []string
	for _, obj := range result.Objects {
		keys = append(keys, obj.Key)
	}

	if len(keys) == 0 {
		return fmt.Errorf("backup not found: %s", backupName)
	}

	return bm.storage.DeleteMany(ctx, keys)
}

// Helper methods

func (bm *BackupManager) createTarArchive(srcPath, dstPath string) error {
	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})
}

func (bm *BackupManager) extractTarArchive(srcPath, dstPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	tr := tar.NewReader(f)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dstPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			dir := filepath.Dir(targetPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bm *BackupManager) compressZstd(srcPath, dstPath string, level int) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	enc, err := zstd.NewWriter(dst, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return err
	}
	defer enc.Close()

	_, err = io.Copy(enc, src)
	return err
}

func (bm *BackupManager) decompressZstd(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dec, err := zstd.NewReader(src)
	if err != nil {
		return err
	}
	defer dec.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, dec)
	return err
}

func (bm *BackupManager) compressGzip(srcPath, dstPath string, level int) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	gz, err := gzip.NewWriterLevel(dst, level)
	if err != nil {
		return err
	}
	defer gz.Close()

	_, err = io.Copy(gz, src)
	return err
}

func (bm *BackupManager) decompressGzip(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	gz, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	defer gz.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, gz)
	return err
}

func (bm *BackupManager) encryptWithAge(srcPath, dstPath string, recipients []string) error {
	// Age encryption implementation
	// Uses filippo.io/age library
	return fmt.Errorf("age encryption: use hanzo/mpc/pkg/kms.BackupKeys() for encrypted backups")
}

func (bm *BackupManager) decryptWithAge(srcPath, dstPath string, identities []string) error {
	// Age decryption implementation
	return fmt.Errorf("age decryption: use hanzo/mpc/pkg/kms.RestoreKeys() for encrypted restores")
}

// bytesWriter is a simple io.Writer that appends to a byte slice.
type bytesWriter struct {
	buf *[]byte
}

func (w *bytesWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
