package server

import (
	"log/slog"
	"net/http"

	"github.com/Strangebrewer/go-budget/app"
	"github.com/Strangebrewer/go-budget/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	HTTPServer *http.Server
}

func New(addr string, allowedOrigins []string, application *app.Application, authMiddleware func(http.Handler) http.Handler) *Server {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Trace-ID"},
		MaxAge:         300,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(slog.Default()))
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.Tracing(application.Tracer))

	registerRoutes(r, application, authMiddleware)

	return &Server{
		HTTPServer: &http.Server{
			Addr:    addr,
			Handler: r,
		},
	}
}
