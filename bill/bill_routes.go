package bill

import (
	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/Strangebrewer/go-budget/transaction"
	"github.com/go-chi/chi/v5"
)

func Routes(store *Store, transactionStore *transaction.Store, tc *tracer.Client) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(store, transactionStore, tc)

	r.Get("/", h.GetAll)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/pay", h.PayBill)

	return r
}
