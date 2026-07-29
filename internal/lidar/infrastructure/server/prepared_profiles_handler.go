package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
)

// ListPreparedProfilesUseCase is the interface for listing prepared profiles.
type ListPreparedProfilesUseCase interface {
	Execute(ctx context.Context, experimentID uuid.UUID, wavelength *float32, polarization, deviceID *string) (*application.ListPreparedProfilesResponse, error)
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

	// Required: experiment_id
	expIDStr := q.Get("experiment_id")
	if expIDStr == "" {
		RespondWithError(w, http.StatusBadRequest, "experiment_id is required")
		return
	}
	expID, err := uuid.Parse(expIDStr)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid experiment_id")
		return
	}

	// Optional filters
	var wavelength *float32
	if v := q.Get("wavelength"); v != "" {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid wavelength")
			return
		}
		f32 := float32(f)
		wavelength = &f32
	}

	var polarization *string
	if v := q.Get("polarization"); v != "" {
		polarization = &v
	}

	var deviceID *string
	if v := q.Get("device_id"); v != "" {
		deviceID = &v
	}

	result, err := h.listUC.Execute(r.Context(), expID, wavelength, polarization, deviceID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusOK, result)
}
