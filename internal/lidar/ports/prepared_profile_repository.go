package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// PreparedProfileRepository provides read/write access to processed profiles.
type PreparedProfileRepository interface {
	// Create persists a new PreparedProfile.
	Create(ctx context.Context, profile *domain.PreparedProfile) error

	// FindByMetaID returns all prepared profiles for a given meta, ordered by created_at.
	FindByMetaID(ctx context.Context, metaID uuid.UUID) ([]domain.PreparedProfile, error)
}
