// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Provider represents a cloud storage provider type.
type Provider string

const (
	ProviderS3    Provider = "s3"
	ProviderGCS   Provider = "gcs"
	ProviderAzure Provider = "azure"
	ProviderLocal Provider = "local"
	ProviderSFTP  Provider = "sftp"
)

// UploadOptions configures upload behavior.
type UploadOptions struct {
	// ContentType for the uploaded object
	ContentType string
	// Metadata key-value pairs
	Metadata map[string]string
	// ServerSideEncryption type (e.g., "AES256", "aws:kms")
	ServerSideEncryption string
	// KMSKeyID for KMS encryption
	KMSKeyID string
	// StorageClass (e.g., "STANDARD", "GLACIER", "NEARLINE")
	StorageClass string
	// ACL (e.g., "private", "public-read")
	ACL string
	// PartSize for multipart uploads (default 64MB)
	PartSize int64
	// Concurrency for parallel part uploads
	Concurrency int
	// ProgressFunc reports upload progress
	ProgressFunc func(bytesUploaded, totalBytes int64)
}

// DownloadOptions configures download behavior.
type DownloadOptions struct {
	// Range for partial downloads (e.g., "bytes=0-1023")
	Range string
	// VersionID for versioned objects
	VersionID string
	// ProgressFunc reports download progress
	ProgressFunc func(bytesDownloaded, totalBytes int64)
}

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
	Metadata     map[string]string
	VersionID    string
	StorageClass string
}

// ListOptions for listing objects.
type ListOptions struct {
	Prefix     string
	Delimiter  string
	MaxKeys    int
	StartAfter string
}

// ListResult contains listing results.
type ListResult struct {
	Objects        []ObjectInfo
	CommonPrefixes []string
	IsTruncated    bool
	NextMarker     string
}

// Storage defines the interface for cloud storage operations.
type Storage interface {
	// Upload uploads data from a reader to the storage.
	Upload(ctx context.Context, key string, reader io.Reader, size int64, opts *UploadOptions) error

	// UploadFile uploads a local file to storage.
	UploadFile(ctx context.Context, key string, localPath string, opts *UploadOptions) error

	// Download downloads data from storage to a writer.
	Download(ctx context.Context, key string, writer io.Writer, opts *DownloadOptions) error

	// DownloadFile downloads from storage to a local file.
	DownloadFile(ctx context.Context, key string, localPath string, opts *DownloadOptions) error

	// Delete removes an object from storage.
	Delete(ctx context.Context, key string) error

	// DeleteMany removes multiple objects.
	DeleteMany(ctx context.Context, keys []string) error

	// Exists checks if an object exists.
	Exists(ctx context.Context, key string) (bool, error)

	// GetInfo retrieves object metadata.
	GetInfo(ctx context.Context, key string) (*ObjectInfo, error)

	// List lists objects with the given options.
	List(ctx context.Context, opts *ListOptions) (*ListResult, error)

	// GetSignedURL generates a pre-signed URL for temporary access.
	GetSignedURL(ctx context.Context, key string, expiry time.Duration, forUpload bool) (string, error)

	// Copy copies an object within the storage.
	Copy(ctx context.Context, srcKey, dstKey string) error

	// Provider returns the storage provider type.
	Provider() Provider

	// Bucket returns the bucket/container name.
	Bucket() string

	// Close releases any resources.
	Close() error
}

// Config holds configuration for storage backends.
type Config struct {
	Provider Provider
	Bucket   string
	Region   string
	Endpoint string // Custom endpoint for S3-compatible stores (MinIO, R2, etc.)

	// AWS-specific
	AWSProfile       string
	AWSAccessKey     string
	AWSSecretKey     string
	AWSSessionToken  string
	AWSAssumeRoleARN string

	// GCS-specific
	GCSCredentialsFile string
	GCSProjectID       string

	// Azure-specific
	AzureAccountName   string
	AzureAccountKey    string
	AzureConnectionStr string

	// SFTP-specific
	SFTPHost       string
	SFTPPort       int
	SFTPUser       string
	SFTPPassword   string
	SFTPPrivateKey string

	// Local-specific
	LocalBasePath string

	// Common options
	PathStyle  bool // Use path-style URLs (for MinIO, etc.)
	DisableSSL bool
	MaxRetries int
	Timeout    time.Duration
}

// New creates a new Storage instance based on the config.
func New(ctx context.Context, cfg *Config) (Storage, error) {
	switch cfg.Provider {
	case ProviderS3:
		return NewS3Storage(ctx, cfg)
	case ProviderGCS:
		return NewGCSStorage(ctx, cfg)
	case ProviderAzure:
		return NewAzureStorage(ctx, cfg)
	case ProviderLocal:
		return NewLocalStorage(cfg)
	case ProviderSFTP:
		return NewSFTPStorage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.Provider)
	}
}

// ParseURI parses a storage URI and returns config.
// Supported formats:
//   - s3://bucket/path
//   - gs://bucket/path
//   - azure://container/path
//   - file:///local/path
//   - sftp://user@host:port/path
func ParseURI(uri string) (*Config, string, error) {
	if strings.HasPrefix(uri, "s3://") {
		parts := strings.SplitN(strings.TrimPrefix(uri, "s3://"), "/", 2)
		bucket := parts[0]
		key := ""
		if len(parts) > 1 {
			key = parts[1]
		}
		return &Config{Provider: ProviderS3, Bucket: bucket}, key, nil
	}

	if strings.HasPrefix(uri, "gs://") {
		parts := strings.SplitN(strings.TrimPrefix(uri, "gs://"), "/", 2)
		bucket := parts[0]
		key := ""
		if len(parts) > 1 {
			key = parts[1]
		}
		return &Config{Provider: ProviderGCS, Bucket: bucket}, key, nil
	}

	if strings.HasPrefix(uri, "azure://") {
		parts := strings.SplitN(strings.TrimPrefix(uri, "azure://"), "/", 2)
		container := parts[0]
		key := ""
		if len(parts) > 1 {
			key = parts[1]
		}
		return &Config{Provider: ProviderAzure, Bucket: container}, key, nil
	}

	if strings.HasPrefix(uri, "file://") {
		path := strings.TrimPrefix(uri, "file://")
		dir := filepath.Dir(path)
		key := filepath.Base(path)
		return &Config{Provider: ProviderLocal, LocalBasePath: dir}, key, nil
	}

	return nil, "", fmt.Errorf("unsupported URI scheme: %s", uri)
}

// ComputeChecksum calculates SHA256 checksum of a file.
func ComputeChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyChecksum verifies a file against expected checksum.
func VerifyChecksum(path, expected string) (bool, error) {
	actual, err := ComputeChecksum(path)
	if err != nil {
		return false, err
	}
	return actual == expected, nil
}
