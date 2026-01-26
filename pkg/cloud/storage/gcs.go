// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSStorage implements Storage for Google Cloud Storage.
type GCSStorage struct {
	client *storage.Client
	bucket *storage.BucketHandle
	cfg    *Config
}

// NewGCSStorage creates a new GCS storage backend.
func NewGCSStorage(ctx context.Context, cfg *Config) (*GCSStorage, error) {
	var opts []option.ClientOption

	if cfg.GCSCredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.GCSCredentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSStorage{
		client: client,
		bucket: client.Bucket(cfg.Bucket),
		cfg:    cfg,
	}, nil
}

// Upload uploads data from a reader to GCS.
func (g *GCSStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, opts *UploadOptions) error {
	obj := g.bucket.Object(key)
	w := obj.NewWriter(ctx)

	if opts != nil {
		if opts.ContentType != "" {
			w.ContentType = opts.ContentType
		}
		if len(opts.Metadata) > 0 {
			w.Metadata = opts.Metadata
		}
		if opts.StorageClass != "" {
			w.StorageClass = opts.StorageClass
		}
	}

	if _, err := io.Copy(w, reader); err != nil {
		w.Close()
		return err
	}

	return w.Close()
}

// UploadFile uploads a local file to GCS.
func (g *GCSStorage) UploadFile(ctx context.Context, key string, localPath string, opts *UploadOptions) error {
	f, err := openFileForUpload(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, _ := f.Stat()
	return g.Upload(ctx, key, f, info.Size(), opts)
}

// Download downloads data from GCS to a writer.
func (g *GCSStorage) Download(ctx context.Context, key string, writer io.Writer, opts *DownloadOptions) error {
	obj := g.bucket.Object(key)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(writer, r)
	return err
}

// DownloadFile downloads from GCS to a local file.
func (g *GCSStorage) DownloadFile(ctx context.Context, key string, localPath string, opts *DownloadOptions) error {
	f, err := createFileForDownload(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return g.Download(ctx, key, f, opts)
}

// Delete removes an object from GCS.
func (g *GCSStorage) Delete(ctx context.Context, key string) error {
	return g.bucket.Object(key).Delete(ctx)
}

// DeleteMany removes multiple objects from GCS.
func (g *GCSStorage) DeleteMany(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := g.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Exists checks if an object exists in GCS.
func (g *GCSStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := g.bucket.Object(key).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetInfo retrieves object metadata from GCS.
func (g *GCSStorage) GetInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	attrs, err := g.bucket.Object(key).Attrs(ctx)
	if err != nil {
		return nil, err
	}

	return &ObjectInfo{
		Key:          key,
		Size:         attrs.Size,
		LastModified: attrs.Updated,
		ETag:         attrs.Etag,
		ContentType:  attrs.ContentType,
		Metadata:     attrs.Metadata,
		StorageClass: attrs.StorageClass,
	}, nil
}

// List lists objects in GCS.
func (g *GCSStorage) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	query := &storage.Query{}

	if opts != nil {
		if opts.Prefix != "" {
			query.Prefix = opts.Prefix
		}
		if opts.Delimiter != "" {
			query.Delimiter = opts.Delimiter
		}
	}

	result := &ListResult{
		Objects: make([]ObjectInfo, 0),
	}

	it := g.bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err != nil {
			break
		}

		result.Objects = append(result.Objects, ObjectInfo{
			Key:          attrs.Name,
			Size:         attrs.Size,
			LastModified: attrs.Updated,
			ETag:         attrs.Etag,
			ContentType:  attrs.ContentType,
			StorageClass: attrs.StorageClass,
		})

		if opts != nil && opts.MaxKeys > 0 && len(result.Objects) >= opts.MaxKeys {
			break
		}
	}

	return result, nil
}

// GetSignedURL generates a pre-signed URL for GCS.
func (g *GCSStorage) GetSignedURL(ctx context.Context, key string, expiry time.Duration, forUpload bool) (string, error) {
	method := "GET"
	if forUpload {
		method = "PUT"
	}

	return g.bucket.SignedURL(key, &storage.SignedURLOptions{
		Method:  method,
		Expires: time.Now().Add(expiry),
	})
}

// Copy copies an object within GCS.
func (g *GCSStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	src := g.bucket.Object(srcKey)
	dst := g.bucket.Object(dstKey)
	_, err := dst.CopierFrom(src).Run(ctx)
	return err
}

// Provider returns the storage provider type.
func (g *GCSStorage) Provider() Provider {
	return ProviderGCS
}

// Bucket returns the bucket name.
func (g *GCSStorage) Bucket() string {
	return g.cfg.Bucket
}

// Close releases any resources.
func (g *GCSStorage) Close() error {
	return g.client.Close()
}

// Helper functions
func openFileForUpload(path string) (*fileWrapper, error) {
	f, err := openFile(path)
	return f, err
}

func createFileForDownload(path string) (*fileWrapper, error) {
	f, err := createFile(path)
	return f, err
}

type fileWrapper struct {
	*os.File
}

func openFile(path string) (*fileWrapper, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{f}, nil
}

func createFile(path string) (*fileWrapper, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{f}, nil
}
