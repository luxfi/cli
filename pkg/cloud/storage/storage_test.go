// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseURI(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		wantProvider Provider
		wantBucket   string
		wantBasePath string
		wantKey      string
		wantErr      bool
	}{
		{
			name:         "s3 uri",
			uri:          "s3://my-bucket/path/to/backup",
			wantProvider: ProviderS3,
			wantBucket:   "my-bucket",
			wantKey:      "path/to/backup",
			wantErr:      false,
		},
		{
			name:         "s3 uri bucket only",
			uri:          "s3://my-bucket",
			wantProvider: ProviderS3,
			wantBucket:   "my-bucket",
			wantKey:      "",
			wantErr:      false,
		},
		{
			name:         "gs uri",
			uri:          "gs://my-gcs-bucket/backups",
			wantProvider: ProviderGCS,
			wantBucket:   "my-gcs-bucket",
			wantKey:      "backups",
			wantErr:      false,
		},
		{
			name:         "azure uri",
			uri:          "azure://container/blob/path",
			wantProvider: ProviderAzure,
			wantBucket:   "container",
			wantKey:      "blob/path",
			wantErr:      false,
		},
		{
			name:         "file uri",
			uri:          "file:///var/backups/mpc",
			wantProvider: ProviderLocal,
			wantBasePath: "/var/backups",
			wantKey:      "mpc",
			wantErr:      false,
		},
		{
			name:         "file uri with deeper path",
			uri:          "file:///home/user/data/backups",
			wantProvider: ProviderLocal,
			wantBasePath: "/home/user/data",
			wantKey:      "backups",
			wantErr:      false,
		},
		{
			name:    "sftp uri not supported",
			uri:     "sftp://user@host:22/backups",
			wantErr: true,
		},
		{
			name:    "invalid uri",
			uri:     "ftp://invalid/path",
			wantErr: true,
		},
		{
			name:    "empty uri",
			uri:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, key, err := ParseURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if cfg.Provider != tt.wantProvider {
				t.Errorf("ParseURI() provider = %v, want %v", cfg.Provider, tt.wantProvider)
			}
			if tt.wantBucket != "" && cfg.Bucket != tt.wantBucket {
				t.Errorf("ParseURI() bucket = %v, want %v", cfg.Bucket, tt.wantBucket)
			}
			if tt.wantBasePath != "" && cfg.LocalBasePath != tt.wantBasePath {
				t.Errorf("ParseURI() basePath = %v, want %v", cfg.LocalBasePath, tt.wantBasePath)
			}
			if key != tt.wantKey {
				t.Errorf("ParseURI() key = %v, want %v", key, tt.wantKey)
			}
		})
	}
}

func TestLocalStorage(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		Provider:      ProviderLocal,
		LocalBasePath: tmpDir,
	}

	ctx := context.Background()

	store, err := NewLocalStorage(cfg)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	defer store.Close()

	t.Run("Provider", func(t *testing.T) {
		if got := store.Provider(); got != ProviderLocal {
			t.Errorf("Provider() = %v, want %v", got, ProviderLocal)
		}
	})

	t.Run("Bucket", func(t *testing.T) {
		if got := store.Bucket(); got != tmpDir {
			t.Errorf("Bucket() = %v, want %v", got, tmpDir)
		}
	})

	t.Run("Upload and Download", func(t *testing.T) {
		testData := "Hello, World!"
		key := "test/file.txt"

		// Upload
		reader := strings.NewReader(testData)
		err := store.Upload(ctx, key, reader, int64(len(testData)), nil)
		if err != nil {
			t.Fatalf("Upload() error = %v", err)
		}

		// Verify file exists
		exists, err := store.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() = false, want true")
		}

		// Download
		var buf strings.Builder
		err = store.Download(ctx, key, &buf, nil)
		if err != nil {
			t.Fatalf("Download() error = %v", err)
		}
		if buf.String() != testData {
			t.Errorf("Download() content = %v, want %v", buf.String(), testData)
		}
	})

	t.Run("GetInfo", func(t *testing.T) {
		testData := "Test content for info"
		key := "info-test.txt"

		reader := strings.NewReader(testData)
		err := store.Upload(ctx, key, reader, int64(len(testData)), nil)
		if err != nil {
			t.Fatalf("Upload() error = %v", err)
		}

		info, err := store.GetInfo(ctx, key)
		if err != nil {
			t.Fatalf("GetInfo() error = %v", err)
		}
		if info.Key != key {
			t.Errorf("GetInfo() key = %v, want %v", info.Key, key)
		}
		if info.Size != int64(len(testData)) {
			t.Errorf("GetInfo() size = %v, want %v", info.Size, len(testData))
		}
	})

	t.Run("List", func(t *testing.T) {
		// Create some test files
		for i := 0; i < 3; i++ {
			key := filepath.Join("list-test", "file"+string(rune('0'+i))+".txt")
			reader := strings.NewReader("content")
			store.Upload(ctx, key, reader, 7, nil)
		}

		result, err := store.List(ctx, &ListOptions{Prefix: "list-test/"})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(result.Objects) != 3 {
			t.Errorf("List() count = %v, want 3", len(result.Objects))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-test.txt"
		reader := strings.NewReader("to be deleted")
		store.Upload(ctx, key, reader, 13, nil)

		// Verify exists
		exists, _ := store.Exists(ctx, key)
		if !exists {
			t.Fatal("File should exist before delete")
		}

		// Delete
		err := store.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deleted
		exists, _ = store.Exists(ctx, key)
		if exists {
			t.Error("File should not exist after delete")
		}
	})

	t.Run("Copy", func(t *testing.T) {
		srcKey := "copy-src.txt"
		dstKey := "copy-dst.txt"
		testData := "copy test data"

		reader := strings.NewReader(testData)
		store.Upload(ctx, srcKey, reader, int64(len(testData)), nil)

		err := store.Copy(ctx, srcKey, dstKey)
		if err != nil {
			t.Fatalf("Copy() error = %v", err)
		}

		// Verify destination exists
		exists, _ := store.Exists(ctx, dstKey)
		if !exists {
			t.Error("Destination file should exist after copy")
		}

		// Verify content
		var buf strings.Builder
		store.Download(ctx, dstKey, &buf, nil)
		if buf.String() != testData {
			t.Errorf("Copy() content = %v, want %v", buf.String(), testData)
		}
	})

	t.Run("UploadFile and DownloadFile", func(t *testing.T) {
		// Create a temp file to upload
		srcFile, err := os.CreateTemp("", "upload-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(srcFile.Name())

		testData := "File upload test data"
		srcFile.WriteString(testData)
		srcFile.Close()

		key := "uploaded-file.txt"

		// Upload file
		err = store.UploadFile(ctx, key, srcFile.Name(), nil)
		if err != nil {
			t.Fatalf("UploadFile() error = %v", err)
		}

		// Download to new file
		dstFile, err := os.CreateTemp("", "download-test-*")
		if err != nil {
			t.Fatal(err)
		}
		dstFile.Close()
		defer os.Remove(dstFile.Name())

		err = store.DownloadFile(ctx, key, dstFile.Name(), nil)
		if err != nil {
			t.Fatalf("DownloadFile() error = %v", err)
		}

		// Verify content
		content, _ := os.ReadFile(dstFile.Name())
		if string(content) != testData {
			t.Errorf("DownloadFile() content = %v, want %v", string(content), testData)
		}
	})

	t.Run("DeleteMany", func(t *testing.T) {
		keys := []string{"delete-many-1.txt", "delete-many-2.txt", "delete-many-3.txt"}
		for _, key := range keys {
			reader := strings.NewReader("content")
			store.Upload(ctx, key, reader, 7, nil)
		}

		// Verify all exist
		for _, key := range keys {
			exists, _ := store.Exists(ctx, key)
			if !exists {
				t.Fatalf("File %s should exist before delete", key)
			}
		}

		// Delete all
		err := store.DeleteMany(ctx, keys)
		if err != nil {
			t.Fatalf("DeleteMany() error = %v", err)
		}

		// Verify all deleted
		for _, key := range keys {
			exists, _ := store.Exists(ctx, key)
			if exists {
				t.Errorf("File %s should not exist after delete", key)
			}
		}
	})
}

func TestNew(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "new-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	t.Run("Local provider", func(t *testing.T) {
		cfg := &Config{
			Provider:      ProviderLocal,
			LocalBasePath: tmpDir,
		}
		store, err := New(ctx, cfg)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		store.Close()
	})

	t.Run("Unknown provider", func(t *testing.T) {
		cfg := &Config{
			Provider: Provider("unknown"),
		}
		_, err := New(ctx, cfg)
		if err == nil {
			t.Error("New() should error for unknown provider")
		}
	})
}

func TestComputeChecksum(t *testing.T) {
	// Create temp file with known content
	tmpFile, err := os.CreateTemp("", "checksum-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	testData := "Hello, World!"
	tmpFile.WriteString(testData)
	tmpFile.Close()

	checksum, err := ComputeChecksum(tmpFile.Name())
	if err != nil {
		t.Fatalf("ComputeChecksum() error = %v", err)
	}

	// SHA256 of "Hello, World!" is known
	expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	if checksum != expected {
		t.Errorf("ComputeChecksum() = %v, want %v", checksum, expected)
	}
}
