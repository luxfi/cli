// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// S3Storage implements Storage for AWS S3 and S3-compatible stores.
type S3Storage struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	bucket     string
	cfg        *Config
}

// NewS3Storage creates a new S3 storage backend.
func NewS3Storage(ctx context.Context, cfg *Config) (*S3Storage, error) {
	var awsCfg aws.Config
	var err error

	// Build options
	var opts []func(*config.LoadOptions) error

	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	// Explicit credentials take precedence
	if cfg.AWSAccessKey != "" && cfg.AWSSecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AWSAccessKey,
				cfg.AWSSecretKey,
				cfg.AWSSessionToken,
			),
		))
	} else if cfg.AWSProfile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.AWSProfile))
	}

	if cfg.MaxRetries > 0 {
		opts = append(opts, config.WithRetryMaxAttempts(cfg.MaxRetries))
	}

	awsCfg, err = config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Assume role if specified
	if cfg.AWSAssumeRoleARN != "" {
		stsClient := sts.NewFromConfig(awsCfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, cfg.AWSAssumeRoleARN)
		awsCfg.Credentials = aws.NewCredentialsCache(creds)
	}

	// S3 client options
	var s3Opts []func(*s3.Options)

	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	if cfg.PathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	// Default part size: 64MB
	partSize := int64(64 * 1024 * 1024)

	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = partSize
		u.Concurrency = 5
	})

	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = partSize
		d.Concurrency = 5
	})

	return &S3Storage{
		client:     client,
		uploader:   uploader,
		downloader: downloader,
		bucket:     cfg.Bucket,
		cfg:        cfg,
	}, nil
}

// Upload uploads data from a reader to S3.
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, opts *UploadOptions) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	}

	if opts != nil {
		if opts.ContentType != "" {
			input.ContentType = aws.String(opts.ContentType)
		}
		if opts.ServerSideEncryption != "" {
			input.ServerSideEncryption = types.ServerSideEncryption(opts.ServerSideEncryption)
		}
		if opts.KMSKeyID != "" {
			input.SSEKMSKeyId = aws.String(opts.KMSKeyID)
		}
		if opts.StorageClass != "" {
			input.StorageClass = types.StorageClass(opts.StorageClass)
		}
		if opts.ACL != "" {
			input.ACL = types.ObjectCannedACL(opts.ACL)
		}
		if len(opts.Metadata) > 0 {
			input.Metadata = opts.Metadata
		}
	}

	// Use multipart upload for large files
	if size > 64*1024*1024 {
		uploadInput := &s3.PutObjectInput{
			Bucket: input.Bucket,
			Key:    input.Key,
			Body:   reader,
		}
		if input.ContentType != nil {
			uploadInput.ContentType = input.ContentType
		}
		if input.ServerSideEncryption != "" {
			uploadInput.ServerSideEncryption = input.ServerSideEncryption
		}
		if input.SSEKMSKeyId != nil {
			uploadInput.SSEKMSKeyId = input.SSEKMSKeyId
		}
		if input.StorageClass != "" {
			uploadInput.StorageClass = input.StorageClass
		}
		if input.ACL != "" {
			uploadInput.ACL = input.ACL
		}
		if input.Metadata != nil {
			uploadInput.Metadata = input.Metadata
		}

		_, err := s.uploader.Upload(ctx, uploadInput)
		return err
	}

	_, err := s.client.PutObject(ctx, input)
	return err
}

// UploadFile uploads a local file to S3.
func (s *S3Storage) UploadFile(ctx context.Context, key string, localPath string, opts *UploadOptions) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	return s.Upload(ctx, key, f, info.Size(), opts)
}

// Download downloads data from S3 to a writer.
func (s *S3Storage) Download(ctx context.Context, key string, writer io.Writer, opts *DownloadOptions) error {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if opts != nil {
		if opts.Range != "" {
			input.Range = aws.String(opts.Range)
		}
		if opts.VersionID != "" {
			input.VersionId = aws.String(opts.VersionID)
		}
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return err
	}
	defer result.Body.Close()

	_, err = io.Copy(writer, result.Body)
	return err
}

// DownloadFile downloads from S3 to a local file.
func (s *S3Storage) DownloadFile(ctx context.Context, key string, localPath string, opts *DownloadOptions) error {
	// Create directory if needed
	dir := localPath[:len(localPath)-len(key)]
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if opts != nil && opts.VersionID != "" {
		input.VersionId = aws.String(opts.VersionID)
	}

	_, err = s.downloader.Download(ctx, f, input)
	return err
}

// Delete removes an object from S3.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// DeleteMany removes multiple objects from S3.
func (s *S3Storage) DeleteMany(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{Key: aws.String(key)}
	}

	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	return err
}

// Exists checks if an object exists in S3.
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a not found error
		return false, nil
	}
	return true, nil
}

// GetInfo retrieves object metadata from S3.
func (s *S3Storage) GetInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	info := &ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(result.ContentLength),
		LastModified: aws.ToTime(result.LastModified),
		ContentType:  aws.ToString(result.ContentType),
		Metadata:     result.Metadata,
		StorageClass: string(result.StorageClass),
	}

	if result.ETag != nil {
		info.ETag = *result.ETag
	}
	if result.VersionId != nil {
		info.VersionID = *result.VersionId
	}

	return info, nil
}

// List lists objects in S3.
func (s *S3Storage) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	}

	if opts != nil {
		if opts.Prefix != "" {
			input.Prefix = aws.String(opts.Prefix)
		}
		if opts.Delimiter != "" {
			input.Delimiter = aws.String(opts.Delimiter)
		}
		if opts.MaxKeys > 0 {
			input.MaxKeys = aws.Int32(int32(opts.MaxKeys))
		}
		if opts.StartAfter != "" {
			input.StartAfter = aws.String(opts.StartAfter)
		}
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}

	listResult := &ListResult{
		Objects:     make([]ObjectInfo, 0, len(result.Contents)),
		IsTruncated: aws.ToBool(result.IsTruncated),
	}

	if result.NextContinuationToken != nil {
		listResult.NextMarker = *result.NextContinuationToken
	}

	for _, obj := range result.Contents {
		listResult.Objects = append(listResult.Objects, ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
		})
	}

	for _, prefix := range result.CommonPrefixes {
		listResult.CommonPrefixes = append(listResult.CommonPrefixes, aws.ToString(prefix.Prefix))
	}

	return listResult, nil
}

// GetSignedURL generates a pre-signed URL for S3.
func (s *S3Storage) GetSignedURL(ctx context.Context, key string, expiry time.Duration, forUpload bool) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	if forUpload {
		req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(expiry))
		if err != nil {
			return "", err
		}
		return req.URL, nil
	}

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// Copy copies an object within S3.
func (s *S3Storage) Copy(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", s.bucket, srcKey)),
		Key:        aws.String(dstKey),
	})
	return err
}

// Provider returns the storage provider type.
func (s *S3Storage) Provider() Provider {
	return ProviderS3
}

// Bucket returns the bucket name.
func (s *S3Storage) Bucket() string {
	return s.bucket
}

// Close releases any resources.
func (s *S3Storage) Close() error {
	return nil
}
