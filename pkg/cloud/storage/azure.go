// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"fmt"
	"io"
	"time"
)

// AzureStorage implements Storage for Azure Blob Storage.
type AzureStorage struct {
	cfg *Config
}

// NewAzureStorage creates a new Azure Blob storage backend.
func NewAzureStorage(ctx context.Context, cfg *Config) (*AzureStorage, error) {
	// TODO: Implement Azure Blob Storage client
	return nil, fmt.Errorf("Azure storage not yet implemented")
}

// Upload uploads data from a reader to Azure.
func (a *AzureStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, opts *UploadOptions) error {
	return fmt.Errorf("Azure storage not yet implemented")
}

// UploadFile uploads a local file to Azure.
func (a *AzureStorage) UploadFile(ctx context.Context, key string, localPath string, opts *UploadOptions) error {
	return fmt.Errorf("Azure storage not yet implemented")
}

// Download downloads data from Azure to a writer.
func (a *AzureStorage) Download(ctx context.Context, key string, writer io.Writer, opts *DownloadOptions) error {
	return fmt.Errorf("Azure storage not yet implemented")
}

// DownloadFile downloads from Azure to a local file.
func (a *AzureStorage) DownloadFile(ctx context.Context, key string, localPath string, opts *DownloadOptions) error {
	return fmt.Errorf("Azure storage not yet implemented")
}

// Delete removes an object from Azure.
func (a *AzureStorage) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("Azure storage not yet implemented")
}

// DeleteMany removes multiple objects from Azure.
func (a *AzureStorage) DeleteMany(ctx context.Context, keys []string) error {
	return fmt.Errorf("Azure storage not yet implemented")
}

// Exists checks if an object exists in Azure.
func (a *AzureStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, fmt.Errorf("Azure storage not yet implemented")
}

// GetInfo retrieves object metadata from Azure.
func (a *AzureStorage) GetInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	return nil, fmt.Errorf("Azure storage not yet implemented")
}

// List lists objects in Azure.
func (a *AzureStorage) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	return nil, fmt.Errorf("Azure storage not yet implemented")
}

// GetSignedURL generates a SAS URL for Azure.
func (a *AzureStorage) GetSignedURL(ctx context.Context, key string, expiry time.Duration, forUpload bool) (string, error) {
	return "", fmt.Errorf("Azure storage not yet implemented")
}

// Copy copies an object within Azure.
func (a *AzureStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	return fmt.Errorf("Azure storage not yet implemented")
}

// Provider returns the storage provider type.
func (a *AzureStorage) Provider() Provider {
	return ProviderAzure
}

// Bucket returns the container name.
func (a *AzureStorage) Bucket() string {
	return a.cfg.Bucket
}

// Close releases any resources.
func (a *AzureStorage) Close() error {
	return nil
}
