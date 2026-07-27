package server

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
)

// CreateExperimentUseCase is the interface for creating experiments.
type CreateExperimentUseCase interface {
	Execute(ctx context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error)
}

// ListExperimentsUseCase is the interface for listing experiments.
type ListExperimentsUseCase interface {
	Execute(ctx context.Context, startTime, endTime time.Time, limit, offset int) (*application.ListExperimentsResponse, error)
}

// ExperimentHandler handles HTTP requests for experiments.
type ExperimentHandler struct {
	createUC CreateExperimentUseCase
	listUC   ListExperimentsUseCase
}

// NewExperimentHandler creates a new ExperimentHandler.
func NewExperimentHandler(createUC CreateExperimentUseCase, listUC ListExperimentsUseCase) *ExperimentHandler {
	return &ExperimentHandler{
		createUC: createUC,
		listUC:   listUC,
	}
}

// HandleCreateExperiment handles POST /api/v1/experiments/create with multipart form data.
func (h *ExperimentHandler) HandleCreateExperiment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		RespondWithError(w, http.StatusBadRequest, "failed to parse form: "+err.Error())
		return
	}
	defer r.MultipartForm.RemoveAll()

	req := &application.CreateExperimentRequest{}

	// --- Required fields ---
	req.Title = r.FormValue("title")
	if req.Title == "" {
		RespondWithError(w, http.StatusBadRequest, "title is required")
		return
	}

	zenith, err := strconv.ParseFloat(r.FormValue("zenith_angle"), 32)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "zenith_angle is required and must be a number")
		return
	}
	req.ZenithAngle = float32(zenith)

	lat, err := strconv.ParseFloat(r.FormValue("latitude"), 32)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "latitude is required and must be a number")
		return
	}
	req.Latitude = float32(lat)

	lng, err := strconv.ParseFloat(r.FormValue("longitude"), 32)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "longitude is required and must be a number")
		return
	}
	req.Longitude = float32(lng)

	req.Comments = r.FormValue("comments")

	// --- Required file ---
	file, fileHeader, err := r.FormFile("experiment_files")
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "experiment_files is required")
		return
	}
	defer file.Close()
	req.ExperimentFiles = application.FileUpload{
		Filename: fileHeader.Filename,
		Size:     fileHeader.Size,
		Reader:   file,
	}

	// --- Optional files ---
	if f, fh, err := r.FormFile("background"); err == nil {
		defer f.Close()
		req.Background = &application.FileUpload{
			Filename: fh.Filename,
			Size:     fh.Size,
			Reader:   f,
		}
	}

	if f, fh, err := r.FormFile("meteo"); err == nil {
		defer f.Close()
		req.Meteo = &application.FileUpload{
			Filename: fh.Filename,
			Size:     fh.Size,
			Reader:   f,
		}
	}

	result, err := h.createUC.Execute(r.Context(), req)
	if err != nil {
		log.Printf("create experiment error: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusCreated, result)
}

// HandleListExperiments handles GET /api/v1/experiments/list.
func (h *ExperimentHandler) HandleListExperiments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var startTime, endTime time.Time

	if st := q.Get("start_time"); st != "" {
		t, err := time.Parse(time.RFC3339, st)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid start_time format, use RFC3339 (e.g. 2026-01-01T00:00:00Z)")
			return
		}
		startTime = t
	}

	if et := q.Get("end_time"); et != "" {
		t, err := time.Parse(time.RFC3339, et)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid end_time format, use RFC3339 (e.g. 2026-01-01T00:00:00Z)")
			return
		}
		endTime = t
	}

	// Default limit/offset
	limit := 100
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	result, err := h.listUC.Execute(r.Context(), startTime, endTime, limit, offset)
	if err != nil {
		log.Printf("list experiments error: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusOK, result)
}
