// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"fmt"
	"io"
	"time"
)

// SFTPStorage implements Storage for SFTP servers.
type SFTPStorage struct {
	cfg *Config
}

// NewSFTPStorage creates a new SFTP storage backend.
func NewSFTPStorage(cfg *Config) (*SFTPStorage, error) {
	// TODO: Implement SFTP client using golang.org/x/crypto/ssh
	return nil, fmt.Errorf("SFTP storage not yet implemented")
}

// Upload uploads data from a reader to SFTP.
func (s *SFTPStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, opts *UploadOptions) error {
	return fmt.Errorf("SFTP storage not yet implemented")
}

// UploadFile uploads a local file to SFTP.
func (s *SFTPStorage) UploadFile(ctx context.Context, key string, localPath string, opts *UploadOptions) error {
	return fmt.Errorf("SFTP storage not yet implemented")
}

// Download downloads data from SFTP to a writer.
func (s *SFTPStorage) Download(ctx context.Context, key string, writer io.Writer, opts *DownloadOptions) error {
	return fmt.Errorf("SFTP storage not yet implemented")
}

// DownloadFile downloads from SFTP to a local file.
func (s *SFTPStorage) DownloadFile(ctx context.Context, key string, localPath string, opts *DownloadOptions) error {
	return fmt.Errorf("SFTP storage not yet implemented")
}

// Delete removes a file from SFTP.
func (s *SFTPStorage) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("SFTP storage not yet implemented")
}

// DeleteMany removes multiple files from SFTP.
func (s *SFTPStorage) DeleteMany(ctx context.Context, keys []string) error {
	return fmt.Errorf("SFTP storage not yet implemented")
}

// Exists checks if a file exists on SFTP.
func (s *SFTPStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, fmt.Errorf("SFTP storage not yet implemented")
}

// GetInfo retrieves file metadata from SFTP.
func (s *SFTPStorage) GetInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	return nil, fmt.Errorf("SFTP storage not yet implemented")
}

// List lists files on SFTP.
func (s *SFTPStorage) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	return nil, fmt.Errorf("SFTP storage not yet implemented")
}

// GetSignedURL is not supported for SFTP.
func (s *SFTPStorage) GetSignedURL(ctx context.Context, key string, expiry time.Duration, forUpload bool) (string, error) {
	return "", fmt.Errorf("signed URLs not supported for SFTP")
}

// Copy copies a file on SFTP.
func (s *SFTPStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	return fmt.Errorf("SFTP storage not yet implemented")
}

// Provider returns the storage provider type.
func (s *SFTPStorage) Provider() Provider {
	return ProviderSFTP
}

// Bucket returns the base path.
func (s *SFTPStorage) Bucket() string {
	return s.cfg.Bucket
}

// Close releases any resources.
func (s *SFTPStorage) Close() error {
	return nil
}
