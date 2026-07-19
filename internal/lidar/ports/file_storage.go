package ports

import (
	"context"
	"io"
	"time"
)

// ObjectInfo contains metadata about an object in storage.
type ObjectInfo struct {
	Bucket       string
	Path         string
	Size         int64
	ETag         string
	ContentType  string
	LastModified time.Time
}

// FileStorage defines the contract for S3-compatible object storage (MinIO, S3, etc.).
type FileStorage interface {
	// CreateBucket creates a bucket if it does not already exist.
	CreateBucket(ctx context.Context, bucket string) error

	// Upload streams data from a reader into the object store and returns object metadata.
	Upload(ctx context.Context, bucket, path string, reader io.Reader, size int64, contentType string) (*ObjectInfo, error)

	// UploadBytes uploads a byte slice as an object.
	UploadBytes(ctx context.Context, bucket, path string, data []byte, contentType string) (*ObjectInfo, error)

	// Download streams the object content to the writer.
	Download(ctx context.Context, bucket, path string, writer io.Writer) error

	// Delete removes an object from the store.
	Delete(ctx context.Context, bucket, path string) error

	// Exists checks whether an object exists in the store.
	Exists(ctx context.Context, bucket, path string) (bool, error)

	// GetInfo returns metadata about an object.
	GetInfo(ctx context.Context, bucket, path string) (*ObjectInfo, error)

	// PresignedGetURL generates a temporary URL for direct GET access.
	PresignedGetURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error)
}
