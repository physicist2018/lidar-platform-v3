package storage

import (
	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// MapObjectInfoToStorageObject converts a ports.ObjectInfo (from MinIO/S3) to a domain.StorageObject.
// ID is generated as a new UUID since MinIO does not provide one.
func MapObjectInfoToStorageObject(info *ports.ObjectInfo) (domain.StorageObject, error) {
	path, err := domain.NewObjectPath(info.Bucket, info.Path)
	if err != nil {
		return domain.StorageObject{}, err
	}

	return domain.StorageObject{
		ID:          uuid.New(),
		Path:        path,
		Size:        info.Size,
		ETag:        info.ETag,
		ContentType: info.ContentType,
		CreatedAt:   info.LastModified,
	}, nil
}
