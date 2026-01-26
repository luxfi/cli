// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage implements Storage for local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local filesystem storage backend.
func NewLocalStorage(cfg *Config) (*LocalStorage, error) {
	if cfg.LocalBasePath == "" {
		return nil, fmt.Errorf("local base path is required")
	}

	// Ensure base path exists
	if err := os.MkdirAll(cfg.LocalBasePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &LocalStorage{
		basePath: cfg.LocalBasePath,
	}, nil
}

func (l *LocalStorage) fullPath(key string) string {
	return filepath.Join(l.basePath, key)
}

// Upload uploads data from a reader to local filesystem.
func (l *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, opts *UploadOptions) error {
	path := l.fullPath(key)

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if opts != nil && opts.ProgressFunc != nil {
		// Wrap reader with progress tracking
		reader = &progressReader{
			reader:       reader,
			total:        size,
			progressFunc: opts.ProgressFunc,
		}
	}

	_, err = io.Copy(f, reader)
	return err
}

// UploadFile uploads a local file (copy).
func (l *LocalStorage) UploadFile(ctx context.Context, key string, localPath string, opts *UploadOptions) error {
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	return l.Upload(ctx, key, src, info.Size(), opts)
}

// Download downloads data to a writer.
func (l *LocalStorage) Download(ctx context.Context, key string, writer io.Writer, opts *DownloadOptions) error {
	path := l.fullPath(key)

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(writer, f)
	return err
}

// DownloadFile downloads to a local file (copy).
func (l *LocalStorage) DownloadFile(ctx context.Context, key string, localPath string, opts *DownloadOptions) error {
	// Create directory if needed
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dst, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	return l.Download(ctx, key, dst, opts)
}

// Delete removes a file.
func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	path := l.fullPath(key)
	return os.Remove(path)
}

// DeleteMany removes multiple files.
func (l *LocalStorage) DeleteMany(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := l.Delete(ctx, key); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Exists checks if a file exists.
func (l *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	path := l.fullPath(key)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetInfo retrieves file metadata.
func (l *LocalStorage) GetInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	path := l.fullPath(key)

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return &ObjectInfo{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// List lists files with the given options.
func (l *LocalStorage) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	result := &ListResult{
		Objects:        make([]ObjectInfo, 0),
		CommonPrefixes: make([]string, 0),
	}

	searchPath := l.basePath
	prefix := ""
	if opts != nil && opts.Prefix != "" {
		prefix = opts.Prefix
		searchPath = filepath.Join(l.basePath, opts.Prefix)
	}

	// Track seen directories for CommonPrefixes when delimiter is set
	seenDirs := make(map[string]bool)

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // Path doesn't exist, return empty result
			}
			return err
		}

		relPath, err := filepath.Rel(l.basePath, path)
		if err != nil {
			return err
		}

		// Handle delimiter for S3-like CommonPrefixes behavior
		if opts != nil && opts.Delimiter != "" && info.IsDir() && relPath != prefix && relPath != "." {
			// Strip prefix from relative path
			subPath := relPath
			if prefix != "" && strings.HasPrefix(relPath, prefix) {
				subPath = strings.TrimPrefix(relPath, prefix)
				subPath = strings.TrimPrefix(subPath, string(filepath.Separator))
			}

			// Check if this is a direct child directory
			if !strings.Contains(subPath, string(filepath.Separator)) && subPath != "" {
				dirPrefix := relPath + "/"
				if !seenDirs[dirPrefix] {
					seenDirs[dirPrefix] = true
					result.CommonPrefixes = append(result.CommonPrefixes, dirPrefix)
				}
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		result.Objects = append(result.Objects, ObjectInfo{
			Key:          relPath,
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})

		if opts != nil && opts.MaxKeys > 0 && len(result.Objects) >= opts.MaxKeys {
			return filepath.SkipAll
		}

		return nil
	})

	return result, err
}

// GetSignedURL is not supported for local storage.
func (l *LocalStorage) GetSignedURL(ctx context.Context, key string, expiry time.Duration, forUpload bool) (string, error) {
	return "file://" + l.fullPath(key), nil
}

// Copy copies a file.
func (l *LocalStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	srcPath := l.fullPath(srcKey)
	dstPath := l.fullPath(dstKey)

	// Create directory if needed
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

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

	_, err = io.Copy(dst, src)
	return err
}

// Provider returns the storage provider type.
func (l *LocalStorage) Provider() Provider {
	return ProviderLocal
}

// Bucket returns the base path.
func (l *LocalStorage) Bucket() string {
	return l.basePath
}

// Close releases any resources.
func (l *LocalStorage) Close() error {
	return nil
}

// progressReader wraps a reader to report progress.
type progressReader struct {
	reader       io.Reader
	total        int64
	read         int64
	progressFunc func(bytesRead, totalBytes int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.progressFunc != nil {
		pr.progressFunc(pr.read, pr.total)
	}
	return n, err
}
