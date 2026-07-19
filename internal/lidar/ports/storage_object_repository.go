package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// StorageObjectRepository defines the persistence contract for storage object metadata.
type StorageObjectRepository interface {
	// Create persists a new storage object and returns it with the DB-generated ID.
	Create(ctx context.Context, obj *domain.StorageObject) (*domain.StorageObject, error)

	// FindByID looks up a storage object by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.StorageObject, error)

	// FindByBucketPath looks up a storage object by bucket and path.
	FindByBucketPath(ctx context.Context, bucket, path string) (*domain.StorageObject, error)
}
