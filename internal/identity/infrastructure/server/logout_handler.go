package server

import (
	"encoding/json"
	"net/http"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
)

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutHandler handles POST /logout.
type LogoutHandler struct {
	uc *application.LogoutUseCase
}

// NewLogoutHandler creates a new LogoutHandler.
func NewLogoutHandler(uc *application.LogoutUseCase) *LogoutHandler {
	return &LogoutHandler{uc: uc}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		RespondWithError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	if err := h.uc.Execute(r.Context(), req.RefreshToken); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}
