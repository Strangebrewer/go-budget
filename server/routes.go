package server

import (
	"net/http"

	"github.com/Strangebrewer/go-budget/app"
	"github.com/Strangebrewer/go-budget/example"
	"github.com/Strangebrewer/go-budget/health"
	"github.com/go-chi/chi/v5"
)

func registerRoutes(r chi.Router, application *app.Application, authMiddleware func(http.Handler) http.Handler) {
	r.Get("/health", health.Handler)

	r.With(authMiddleware).Mount("/examples", example.Routes(application.ExampleStore))
}
