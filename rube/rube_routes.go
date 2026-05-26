package rube

import (
	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/go-chi/chi/v5"
)

func Routes(nextURL string, tc *tracer.Client) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(nextURL, tc)
	r.Post("/", h.Chain)
	return r
}
