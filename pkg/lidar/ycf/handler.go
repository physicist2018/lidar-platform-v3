// Package ycf provides a Yandex Cloud Function HTTP handler that computes the
// pure molecular (Rayleigh) backscatter signal with pkg/lidar/molecular.
//
// The handler accepts a POST request with a molecular.Input JSON body and
// returns the molecular.Result as JSON. CORS headers are set so the function
// can also be called from browsers.
//
// Yandex Cloud Functions requires a main package for deployment; the thin
// cmd/ycf-molecular wrapper exposes this Handler as the entry point.
package ycf

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/physcist2018/lidar-platform-v3/pkg/lidar/molecular"
)

// Handler is the Yandex Cloud Function entry point (HTTP trigger).
//
//	POST /
//	{"range": [...], "zenith_angle": 0, "wavelength": 532, "atmosphere": {...}}
//
//	200
//	{"backscatter": [...], "extinction": [...], "transmission": [...], "signal": [...]}
//
// Validation errors return 400, unexpected errors return 500, non-POST
// methods return 405.
func Handler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed, use POST")
		return
	}

	var in molecular.Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	res, err := molecular.Compute(r.Context(), in)
	if err != nil {
		status := http.StatusBadRequest
		if !isValidationError(err) {
			status = http.StatusInternalServerError
		}
		respondError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, res)
}

// isValidationError reports whether err is one of the molecular input
// validation errors (user-caused, mapped to 400).
func isValidationError(err error) bool {
	for _, e := range []error{
		molecular.ErrNilInput,
		molecular.ErrEmptyInput,
		molecular.ErrTooFewPoints,
		molecular.ErrLengthMismatch,
		molecular.ErrRangeNotIncreasing,
		molecular.ErrNonFinite,
		molecular.ErrInvalidParam,
		molecular.ErrInvalidModel,
	} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// enableCORS sets permissive CORS headers for browser calls.
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("ycf: encode response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
