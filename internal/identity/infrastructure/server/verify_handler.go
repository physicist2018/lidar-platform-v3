package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
)

type verifyRequest struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// VerifyHandler handles POST /verify (JSON API for frontend).
type VerifyHandler struct {
	uc *application.VerifyUseCase
}

// NewVerifyHandler creates a new VerifyHandler.
func NewVerifyHandler(uc *application.VerifyUseCase) *VerifyHandler {
	return &VerifyHandler{uc: uc}
}

func (h *VerifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" || req.Email == "" {
		RespondWithError(w, http.StatusBadRequest, "token and email are required")
		return
	}

	if err := h.uc.Execute(r.Context(), req.Token, req.Email); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidToken),
			errors.Is(err, domain.ErrTokenExpired),
			errors.Is(err, domain.ErrEmailMismatch):
			RespondWithError(w, http.StatusBadRequest, "invalid or expired verification token")
		case errors.Is(err, domain.ErrAlreadyVerified):
			RespondWithError(w, http.StatusConflict, "user already verified")
		default:
			RespondWithError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "email verified successfully",
	})
}

// VerifyLinkHandler handles GET /verify?token=...&email=... (link from email).
// It verifies the token and redirects the browser to the frontend.
type VerifyLinkHandler struct {
	uc          *application.VerifyUseCase
	frontendURL string // e.g. "https://frontend.example.com"
}

// NewVerifyLinkHandler creates a new VerifyLinkHandler.
func NewVerifyLinkHandler(uc *application.VerifyUseCase, frontendURL string) *VerifyLinkHandler {
	return &VerifyLinkHandler{uc: uc, frontendURL: frontendURL}
}

func (h *VerifyLinkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	email := r.URL.Query().Get("email")

	if token == "" || email == "" {
		if h.frontendURL != "" {
			http.Redirect(w, r, h.frontendURL+"/verified?status=error&reason=missing_params", http.StatusFound)
		} else {
			RespondWithError(w, http.StatusBadRequest, "token and email are required")
		}
		return
	}

	if err := h.uc.Execute(r.Context(), token, email); err != nil {
		var reason string
		switch {
		case errors.Is(err, domain.ErrAlreadyVerified):
			reason = "already_verified"
		case errors.Is(err, domain.ErrInvalidToken),
			errors.Is(err, domain.ErrTokenExpired),
			errors.Is(err, domain.ErrEmailMismatch):
			reason = "invalid_token"
		default:
			reason = "server_error"
		}

		if h.frontendURL != "" {
			http.Redirect(w, r, h.frontendURL+"/verified?status=error&reason="+reason, http.StatusFound)
		} else {
			RespondWithError(w, http.StatusBadRequest, reason)
		}
		return
	}

	if h.frontendURL != "" {
		http.Redirect(w, r, h.frontendURL+"/verified?status=ok", http.StatusFound)
	} else {
		RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "email verified successfully",
		})
	}
}
