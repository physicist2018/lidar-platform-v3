package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// MinIOFileStorage implements ports.FileStorage backed by an S3-compatible server.
type MinIOFileStorage struct {
	client *minio.Client
	cfg    Config
}

// NewMinIOFileStorage creates and returns a new MinIOFileStorage.
func NewMinIOFileStorage(cfg Config) (*MinIOFileStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: create client: %w", err)
	}
	return &MinIOFileStorage{client: client, cfg: cfg}, nil
}

// CreateBucket creates a bucket if it does not already exist.
func (s *MinIOFileStorage) CreateBucket(ctx context.Context, bucket string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("minio: check bucket %q: %w", bucket, err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("minio: create bucket %q: %w", bucket, err)
		}
	}
	return nil
}

// Upload streams data from a reader into the object store.
func (s *MinIOFileStorage) Upload(ctx context.Context, bucket, path string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, path, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio: upload %s/%s: %w", bucket, path, err)
	}
	return nil
}

// UploadBytes uploads a byte slice as an object.
func (s *MinIOFileStorage) UploadBytes(ctx context.Context, bucket, path string, data []byte, contentType string) error {
	return s.Upload(ctx, bucket, path, bytes.NewReader(data), int64(len(data)), contentType)
}

// Download streams the object content to the writer.
func (s *MinIOFileStorage) Download(ctx context.Context, bucket, path string, writer io.Writer) error {
	obj, err := s.client.GetObject(ctx, bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("minio: get object %s/%s: %w", bucket, path, err)
	}
	defer obj.Close()

	if _, err := io.Copy(writer, obj); err != nil {
		return fmt.Errorf("minio: download %s/%s: %w", bucket, path, err)
	}
	return nil
}

// Delete removes an object from the store.
func (s *MinIOFileStorage) Delete(ctx context.Context, bucket, path string) error {
	if err := s.client.RemoveObject(ctx, bucket, path, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio: delete %s/%s: %w", bucket, path, err)
	}
	return nil
}

// Exists checks whether an object exists in the store.
func (s *MinIOFileStorage) Exists(ctx context.Context, bucket, path string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucket, path, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "NotFound" {
			return false, nil
		}
		return false, fmt.Errorf("minio: stat %s/%s: %w", bucket, path, err)
	}
	return true, nil
}

// GetInfo returns metadata about an object.
func (s *MinIOFileStorage) GetInfo(ctx context.Context, bucket, path string) (*ports.ObjectInfo, error) {
	obj, err := s.client.StatObject(ctx, bucket, path, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio: stat %s/%s: %w", bucket, path, err)
	}
	return &ports.ObjectInfo{
		Bucket:       bucket,
		Path:         obj.Key,
		Size:         obj.Size,
		ETag:         obj.ETag,
		ContentType:  obj.ContentType,
		LastModified: obj.LastModified,
	}, nil
}

// PresignedGetURL generates a temporary URL for direct GET access.
func (s *MinIOFileStorage) PresignedGetURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, bucket, path, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("minio: presigned get %s/%s: %w", bucket, path, err)
	}
	return u.String(), nil
}
