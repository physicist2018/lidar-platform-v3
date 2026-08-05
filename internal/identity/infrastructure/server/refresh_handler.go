package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
)

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshHandler handles POST /refresh.
type RefreshHandler struct {
	uc *application.RefreshUseCase
}

// NewRefreshHandler creates a new RefreshHandler.
func NewRefreshHandler(uc *application.RefreshUseCase) *RefreshHandler {
	return &RefreshHandler{uc: uc}
}

func (h *RefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		RespondWithError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	result, err := h.uc.Execute(r.Context(), req.RefreshToken, r.UserAgent(), clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidRefreshToken):
			RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		default:
			RespondWithError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	RespondWithJSON(w, http.StatusOK, result)
}

// clientIP extracts the client IP from the request, stripping the port.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
