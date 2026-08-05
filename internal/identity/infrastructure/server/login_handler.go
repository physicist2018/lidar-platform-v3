package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginHandler handles POST /login.
type LoginHandler struct {
	uc *application.LoginUseCase
}

// NewLoginHandler creates a new LoginHandler.
func NewLoginHandler(uc *application.LoginUseCase) *LoginHandler {
	return &LoginHandler{uc: uc}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	result, err := h.uc.Execute(r.Context(), req.Email, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidEmail):
			RespondWithError(w, http.StatusBadRequest, "invalid email format")
		case errors.Is(err, domain.ErrInvalidCredentials):
			RespondWithError(w, http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, domain.ErrAccountNotVerified):
			RespondWithError(w, http.StatusForbidden, "account not verified")
		default:
			RespondWithError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	RespondWithJSON(w, http.StatusOK, result)
}
