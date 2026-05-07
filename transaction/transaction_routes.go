package transaction

import (
	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/go-chi/chi/v5"
)

func Routes(store *Store, tc *tracer.Client) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(store, tc)

	r.Get("/", h.GetAll)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}
