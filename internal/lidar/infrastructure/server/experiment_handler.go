package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
)

// ExperimentHandler handles HTTP requests for experiments.
type ExperimentHandler struct {
	createUC *application.CreateExperimentUseCase
}

// NewExperimentHandler creates a new ExperimentHandler.
func NewExperimentHandler(createUC *application.CreateExperimentUseCase) *ExperimentHandler {
	return &ExperimentHandler{createUC: createUC}
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
