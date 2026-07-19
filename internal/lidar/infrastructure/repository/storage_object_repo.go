package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresStorageObjectRepository implements ports.StorageObjectRepository backed by sqlc.
type PostgresStorageObjectRepository struct {
	q *db.Queries
}

// NewPostgresStorageObjectRepository creates a new PostgresStorageObjectRepository.
func NewPostgresStorageObjectRepository(dbtx db.DBTX) *PostgresStorageObjectRepository {
	return &PostgresStorageObjectRepository{q: db.New(dbtx)}
}

// Create persists a new storage object and returns it with the DB-generated ID.
func (r *PostgresStorageObjectRepository) Create(ctx context.Context, obj *domain.StorageObject) (*domain.StorageObject, error) {
	var metadata []byte
	if obj.Metadata != nil {
		metadata, _ = json.Marshal(obj.Metadata)
	}

	u, err := r.q.CreateStorageObject(ctx, db.CreateStorageObjectParams{
		Bucket:      obj.Path.Bucket,
		Path:        obj.Path.Path,
		SizeBytes:   toNullInt64(obj.Size),
		Etag:        toNullString(obj.ETag),
		ContentType: toNullString(obj.ContentType),
		Metadata:    pqtype.NullRawMessage{RawMessage: metadata, Valid: metadata != nil},
	})
	if err != nil {
		return nil, err
	}
	return mapStorageObject(u), nil
}

// FindByID looks up a storage object by ID.
func (r *PostgresStorageObjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.StorageObject, error) {
	u, err := r.q.GetStorageObjectByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapStorageObject(u), nil
}

// FindByBucketPath looks up a storage object by bucket and path.
func (r *PostgresStorageObjectRepository) FindByBucketPath(ctx context.Context, bucket, path string) (*domain.StorageObject, error) {
	u, err := r.q.GetStorageObjectByBucketPath(ctx, db.GetStorageObjectByBucketPathParams{
		Bucket: bucket,
		Path:   path,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapStorageObject(u), nil
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func mapStorageObject(u db.LidarStorageObject) *domain.StorageObject {
	obj := &domain.StorageObject{
		ID:          u.ID,
		Path:        domain.ObjectPath{Bucket: u.Bucket, Path: u.Path},
		Size:        u.SizeBytes.Int64,
		ETag:        u.Etag.String,
		ContentType: u.ContentType.String,
		CreatedAt:   u.CreatedAt,
	}
	if u.Metadata.Valid {
		var meta map[string]any
		if err := json.Unmarshal(u.Metadata.RawMessage, &meta); err == nil {
			obj.Metadata = meta
		}
	}
	return obj
}

func toNullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v != 0}
}
