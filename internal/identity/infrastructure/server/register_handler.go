package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterHandler handles POST /register.
type RegisterHandler struct {
	uc *application.RegisterUseCase
}

// NewRegisterHandler creates a new RegisterHandler.
func NewRegisterHandler(uc *application.RegisterUseCase) *RegisterHandler {
	return &RegisterHandler{uc: uc}
}

func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	if err := h.uc.Execute(r.Context(), req.Email, req.Password); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidEmail):
			RespondWithError(w, http.StatusBadRequest, "invalid email format")
		case errors.Is(err, domain.ErrWeakPassword):
			RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
		case errors.Is(err, domain.ErrEmailAlreadyExists):
			RespondWithError(w, http.StatusConflict, "email already registered")
		default:
			RespondWithError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	RespondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "user registered, verification email sent",
	})
}
