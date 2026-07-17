package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the chi router for the identity service.
func NewRouter(
	registerHandler *RegisterHandler,
	verifyHandler *VerifyHandler,
	verifyLinkHandler *VerifyLinkHandler,
) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/", func(r chi.Router) {
		r.Post("/register", registerHandler.ServeHTTP)
		r.Post("/verify", verifyHandler.ServeHTTP)
		r.Get("/verify", verifyLinkHandler.ServeHTTP)
	})

	return r
}
