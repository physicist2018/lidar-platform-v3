package domain

import (
	"time"

	"github.com/google/uuid"
)

// ObjectPath is a value object representing the location of an object in storage.
type ObjectPath struct {
	Bucket string
	Path   string
}

// NewObjectPath validates and creates an ObjectPath.
func NewObjectPath(bucket, path string) (ObjectPath, error) {
	if bucket == "" || path == "" {
		return ObjectPath{}, ErrInvalidPath
	}
	return ObjectPath{Bucket: bucket, Path: path}, nil
}

// Key returns the full key in "bucket/path" format.
func (p ObjectPath) Key() string {
	return p.Bucket + "/" + p.Path
}

// String returns a human-readable representation.
func (p ObjectPath) String() string {
	return p.Key()
}

// StorageObject is an entity representing a file or object stored in external storage (S3, MinIO, etc.).
type StorageObject struct {
	ID          uuid.UUID
	Path        ObjectPath
	Size        int64
	ETag        string
	ContentType string
	Metadata    map[string]any
	CreatedAt   time.Time
}

// StorageObjectOption is a functional option for creating a StorageObject.
type StorageObjectOption func(*StorageObject)

// WithSize sets the object size in bytes.
func WithSize(size int64) StorageObjectOption {
	return func(o *StorageObject) { o.Size = size }
}

// WithETag sets the ETag.
func WithETag(etag string) StorageObjectOption {
	return func(o *StorageObject) { o.ETag = etag }
}

// WithContentType sets the MIME content type.
func WithContentType(ct string) StorageObjectOption {
	return func(o *StorageObject) { o.ContentType = ct }
}

// WithMetadata sets the metadata map.
func WithMetadata(meta map[string]any) StorageObjectOption {
	return func(o *StorageObject) { o.Metadata = meta }
}

// NewStorageObject creates a new StorageObject with a generated ID and the current timestamp.
func NewStorageObject(path ObjectPath, opts ...StorageObjectOption) StorageObject {
	o := StorageObject{
		ID:        uuid.New(),
		Path:      path,
		CreatedAt: time.Now(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// Key returns the storage key for this object.
func (o *StorageObject) Key() string {
	return o.Path.Key()
}

// UpdateContentInfo updates the size, ETag, and content type.
func (o *StorageObject) UpdateContentInfo(size int64, etag, contentType string) {
	o.Size = size
	o.ETag = etag
	o.ContentType = contentType
}

// UpdateMetadata replaces the metadata map.
func (o *StorageObject) UpdateMetadata(meta map[string]any) {
	o.Metadata = meta
}
