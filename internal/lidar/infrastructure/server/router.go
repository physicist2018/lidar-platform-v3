package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the chi router for the lidar service.
func NewRouter(expHandler *ExperimentHandler, taskHandler *TaskHandler, preparedHandler *PreparedProfilesHandler, jwtSecret string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", healthHandler)

	// All /api/v1 routes require JWT authentication.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(JWTAuthMiddleware(jwtSecret))

		r.Get("/experiments/list", func(w http.ResponseWriter, r *http.Request) {
			if expHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "experiment listing not available")
				return
			}
			expHandler.HandleListExperiments(w, r)
		})

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

		r.Delete("/tasks/{taskID}", func(w http.ResponseWriter, r *http.Request) {
			if taskHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "task deletion not available")
				return
			}
			taskHandler.HandleDeleteTask(w, r)
		})

		r.Get("/prepared-profiles", func(w http.ResponseWriter, r *http.Request) {
			if preparedHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "prepared profiles not available")
				return
			}
			preparedHandler.HandleListPreparedProfiles(w, r)
		})

		r.Get("/prepared-profiles/experiments", func(w http.ResponseWriter, r *http.Request) {
			if preparedHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "prepared profiles not available")
				return
			}
			preparedHandler.HandleListExperiments(w, r)
		})

		r.Get("/prepared-profiles/filters", func(w http.ResponseWriter, r *http.Request) {
			if preparedHandler == nil {
				RespondWithError(w, http.StatusNotImplemented, "prepared profiles not available")
				return
			}
			preparedHandler.HandleListFilters(w, r)
		})
	})

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
