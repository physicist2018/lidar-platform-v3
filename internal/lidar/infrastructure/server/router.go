package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the chi router for the lidar service.
func NewRouter(expHandler *ExperimentHandler, taskHandler *TaskHandler, jwtSecret string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", healthHandler)

	// All /api/v1 routes require JWT authentication.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(JWTAuthMiddleware(jwtSecret))

		r.Post("/experiments/create", func(w http.ResponseWriter, r *http.Request) {
			if expHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "experiment creation not available")
				return
			}
			expHandler.HandleCreateExperiment(w, r)
		})

		r.Post("/experiments/task", func(w http.ResponseWriter, r *http.Request) {
			if taskHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "task creation not available")
				return
			}
			taskHandler.HandleCreateTask(w, r)
		})

		r.Get("/tasks/{taskID}", func(w http.ResponseWriter, r *http.Request) {
			if taskHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "task status not available")
				return
			}
			taskHandler.HandleGetTaskStatus(w, r)
		})
	})

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
