package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
)

// ListPreparedProfilesUseCase is the interface for listing prepared profiles.
type ListPreparedProfilesUseCase interface {
	Execute(ctx context.Context, experimentID uuid.UUID, wavelength *float32, polarization, deviceID *string) (*application.ListPreparedProfilesResponse, error)
	ListExperiments(ctx context.Context) ([]application.PreparedExperimentResponse, error)
	ListFilters(ctx context.Context, experimentID uuid.UUID, wavelength *float32, polarization *string) (*application.PreparedFiltersResponse, error)
}

// PreparedProfilesHandler handles HTTP requests for prepared profiles.
type PreparedProfilesHandler struct {
	listUC ListPreparedProfilesUseCase
}

// NewPreparedProfilesHandler creates a new PreparedProfilesHandler.
func NewPreparedProfilesHandler(listUC ListPreparedProfilesUseCase) *PreparedProfilesHandler {
	return &PreparedProfilesHandler{listUC: listUC}
}

// HandleListPreparedProfiles handles GET /api/v1/prepared-profiles.
func (h *PreparedProfilesHandler) HandleListPreparedProfiles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	expID, err := parseExperimentID(q)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	wavelength := parseOptionalFloat(q, "wavelength")
	polarization := parseOptionalString(q, "polarization")
	deviceID := parseOptionalString(q, "device_id")

	result, err := h.listUC.Execute(r.Context(), expID, wavelength, polarization, deviceID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusOK, result)
}

// HandleListExperiments handles GET /api/v1/prepared-profiles/experiments.
func (h *PreparedProfilesHandler) HandleListExperiments(w http.ResponseWriter, r *http.Request) {
	items, err := h.listUC.ListExperiments(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusOK, items)
}

// HandleListFilters handles GET /api/v1/prepared-profiles/filters?experiment_id=...
func (h *PreparedProfilesHandler) HandleListFilters(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	expID, err := parseExperimentID(q)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	wavelength := parseOptionalFloat(q, "wavelength")
	polarization := parseOptionalString(q, "polarization")

	result, err := h.listUC.ListFilters(r.Context(), expID, wavelength, polarization)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseExperimentID(q map[string][]string) (uuid.UUID, error) {
	s := getQueryParam(q, "experiment_id")
	if s == "" {
		return uuid.Nil, fmt.Errorf("experiment_id is required")
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid experiment_id")
	}
	return id, nil
}

func parseOptionalFloat(q map[string][]string, key string) *float32 {
	s := getQueryParam(q, key)
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return nil
	}
	v := float32(f)
	return &v
}

func parseOptionalString(q map[string][]string, key string) *string {
	s := getQueryParam(q, key)
	if s == "" {
		return nil
	}
	return &s
}

func getQueryParam(q map[string][]string, key string) string {
	if vals, ok := q[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}
