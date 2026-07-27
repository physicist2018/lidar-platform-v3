package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// ExperimentItem is a single experiment in the list response.
type ExperimentItem struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Comments    string    `json:"comments"`
	ZenithAngle float32   `json:"zenith_angle"`
	StartTime   time.Time `json:"experiment_start"`
	EndTime     time.Time `json:"experiment_end"`
	Latitude    float32   `json:"latitude"`
	Longitude   float32   `json:"longitude"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListExperimentsResponse is the response for listing experiments.
type ListExperimentsResponse struct {
	Experiments []ExperimentItem `json:"experiments"`
	Count       int              `json:"count"`
}

// ListExperimentsUseCase retrieves experiments, optionally filtered by time range.
type ListExperimentsUseCase struct {
	repo ports.ExperimentRepository
}

// NewListExperimentsUseCase creates a new ListExperimentsUseCase.
func NewListExperimentsUseCase(repo ports.ExperimentRepository) *ListExperimentsUseCase {
	return &ListExperimentsUseCase{repo: repo}
}

// Execute returns experiments within the given time range [startTime, endTime].
// If startTime is zero, the range start is unbounded (beginning of time).
// If endTime is zero, the range end is unbounded (end of time).
func (uc *ListExperimentsUseCase) Execute(ctx context.Context, startTime, endTime time.Time, limit, offset int) (*ListExperimentsResponse, error) {
	if startTime.IsZero() {
		startTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if endTime.IsZero() {
		endTime = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	experiments, err := uc.repo.FindByTimeRange(ctx, startTime, endTime, limit, offset)
	if err != nil {
		return nil, err
	}

	items := make([]ExperimentItem, len(experiments))
	for i, exp := range experiments {
		items[i] = mapExperimentToItem(exp)
	}

	return &ListExperimentsResponse{
		Experiments: items,
		Count:       len(items),
	}, nil
}

func mapExperimentToItem(exp domain.Experiment) ExperimentItem {
	return ExperimentItem{
		ID:          exp.ID,
		Title:       exp.Title,
		Comments:    exp.Comments,
		ZenithAngle: exp.ZenithAngle,
		StartTime:   exp.TimeRange.Start,
		EndTime:     exp.TimeRange.End,
		Latitude:    exp.GeoLocation.Latitude,
		Longitude:   exp.GeoLocation.Longitude,
		CreatedAt:   exp.CreatedAt,
		UpdatedAt:   exp.UpdatedAt,
	}
}
