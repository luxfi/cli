//go:build !gcs

// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Default-build stub for the GCS storage backend. cloud.google.com/go/storage
// pulls google.golang.org/grpc + s2a-go + cloud.google.com/go/iam + the
// full Google Cloud SDK transitive chain. grpc is opt-in only across the
// Lux/Hanzo stack, so GCS follows the same discipline: enable with
// `-tags gcs`. Without the tag, NewGCSStorage returns errGCSDisabled and
// the dispatch in storage.go returns the operator a clear error pointing
// them at s3:// (hanzoai/s3, MinIO-protocol).
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

var errGCSDisabled = errors.New(
	"cli/storage: gcs provider not compiled (build with `-tags gcs` to enable Google Cloud Storage; default builds use s3 via hanzoai/s3)",
)

// GCSStorage is the disabled-build type. NewGCSStorage always returns
// errGCSDisabled so the methods are never invoked; they only need to
// exist so the Storage interface is satisfied at type-check time.
type GCSStorage struct{}

func NewGCSStorage(_ context.Context, _ *Config) (*GCSStorage, error) {
	return nil, errGCSDisabled
}

func (*GCSStorage) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ *UploadOptions) error {
	return errGCSDisabled
}
func (*GCSStorage) UploadFile(_ context.Context, _ string, _ string, _ *UploadOptions) error {
	return errGCSDisabled
}
func (*GCSStorage) Download(_ context.Context, _ string, _ io.Writer, _ *DownloadOptions) error {
	return errGCSDisabled
}
func (*GCSStorage) DownloadFile(_ context.Context, _ string, _ string, _ *DownloadOptions) error {
	return errGCSDisabled
}
func (*GCSStorage) Delete(_ context.Context, _ string) error { return errGCSDisabled }
func (*GCSStorage) DeleteMany(_ context.Context, _ []string) error {
	return errGCSDisabled
}
func (*GCSStorage) Exists(_ context.Context, _ string) (bool, error) {
	return false, errGCSDisabled
}
func (*GCSStorage) GetInfo(_ context.Context, _ string) (*ObjectInfo, error) {
	return nil, errGCSDisabled
}
func (*GCSStorage) List(_ context.Context, _ *ListOptions) (*ListResult, error) {
	return nil, errGCSDisabled
}
func (*GCSStorage) GetSignedURL(_ context.Context, _ string, _ time.Duration, _ bool) (string, error) {
	return "", errGCSDisabled
}
func (*GCSStorage) Copy(_ context.Context, _, _ string) error { return errGCSDisabled }
func (*GCSStorage) Provider() Provider                        { return ProviderGCS }
func (*GCSStorage) Bucket() string                            { return "" }
func (*GCSStorage) Close() error                              { return nil }
