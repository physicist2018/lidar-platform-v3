package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// PreparedProfileItem is a single prepared profile in the list response.
type PreparedProfileItem struct {
	ID             uuid.UUID `json:"id"`
	Wavelength     float32   `json:"wavelength"`
	Polarization   string    `json:"polarization"`
	DeviceID       string    `json:"device_id"`
	BinWidth       float32   `json:"bin_width"`
	Data           []float32 `json:"data"`
	BackgroundType string    `json:"background_type"`
	BackgroundFrom float32   `json:"background_from"`
	TrimFrom       float32   `json:"trim_from"`
	CreatedAt      string    `json:"created_at"`
}

// ListPreparedProfilesResponse is the response for listing prepared profiles.
type ListPreparedProfilesResponse struct {
	Profiles []PreparedProfileItem `json:"profiles"`
	Count    int                   `json:"count"`
}

// PreparedFiltersResponse contains available filter values for the dropdowns.
type PreparedFiltersResponse struct {
	Wavelengths   []float32 `json:"wavelengths"`
	Polarizations []string  `json:"polarizations"`
	DeviceIDs     []string  `json:"device_ids"`
}

// ListPreparedProfilesUseCase retrieves prepared profiles and filter metadata.
type ListPreparedProfilesUseCase struct {
	repo ports.PreparedProfileRepository
}

// NewListPreparedProfilesUseCase creates a new ListPreparedProfilesUseCase.
func NewListPreparedProfilesUseCase(repo ports.PreparedProfileRepository) *ListPreparedProfilesUseCase {
	return &ListPreparedProfilesUseCase{repo: repo}
}

// Execute retrieves prepared profiles with optional filters.
func (uc *ListPreparedProfilesUseCase) Execute(
	ctx context.Context,
	experimentID uuid.UUID,
	wavelength *float32,
	polarization, deviceID *string,
) (*ListPreparedProfilesResponse, error) {
	if experimentID == uuid.Nil {
		return nil, fmt.Errorf("experiment_id is required")
	}

	views, err := uc.repo.FindByExperiment(ctx, experimentID, wavelength, polarization, deviceID)
	if err != nil {
		return nil, fmt.Errorf("find prepared profiles: %w", err)
	}

	items := make([]PreparedProfileItem, len(views))
	for i, v := range views {
		items[i] = mapPreparedProfileView(v)
	}

	return &ListPreparedProfilesResponse{
		Profiles: items,
		Count:    len(items),
	}, nil
}

// ListExperiments returns experiment IDs that have prepared data.
func (uc *ListPreparedProfilesUseCase) ListExperiments(ctx context.Context) ([]uuid.UUID, error) {
	return uc.repo.FindExperiments(ctx)
}

// ListFilters returns available filter values for the given experiment.
// wavelength, polarization, deviceID are optional filters for narrowing down.
func (uc *ListPreparedProfilesUseCase) ListFilters(
	ctx context.Context,
	experimentID uuid.UUID,
	wavelength *float32,
	polarization *string,
) (*PreparedFiltersResponse, error) {
	if experimentID == uuid.Nil {
		return nil, fmt.Errorf("experiment_id is required")
	}

	wavelengths, err := uc.repo.FindWavelengths(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("find wavelengths: %w", err)
	}

	polarizations, err := uc.repo.FindPolarizations(ctx, experimentID, wavelength)
	if err != nil {
		return nil, fmt.Errorf("find polarizations: %w", err)
	}

	deviceIDs, err := uc.repo.FindDeviceIDs(ctx, experimentID, wavelength, polarization)
	if err != nil {
		return nil, fmt.Errorf("find device ids: %w", err)
	}

	return &PreparedFiltersResponse{
		Wavelengths:   wavelengths,
		Polarizations: polarizations,
		DeviceIDs:     deviceIDs,
	}, nil
}

func mapPreparedProfileView(v domain.PreparedProfileView) PreparedProfileItem {
	return PreparedProfileItem{
		ID:             v.ID,
		Wavelength:     v.Wavelength,
		Polarization:   v.Polarization,
		DeviceID:       v.DeviceID,
		BinWidth:       v.BinWidth,
		Data:           v.Data,
		BackgroundType: string(v.BackgroundType),
		BackgroundFrom: v.BackgroundFrom,
		TrimFrom:       v.TrimFrom,
		CreatedAt:      v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
