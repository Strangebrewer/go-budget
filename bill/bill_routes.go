package bill

import (
	"github.com/go-chi/chi/v5"
	"github.com/Strangebrewer/go-budget/transaction"
)

func Routes(store *Store, transactionStore *transaction.Store) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(store, transactionStore)

	r.Get("/", h.GetAll)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/pay", h.PayBill)

	return r
}
