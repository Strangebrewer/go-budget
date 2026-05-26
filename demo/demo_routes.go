package demo

import (
	"github.com/go-chi/chi/v5"

	"github.com/Strangebrewer/go-budget/account"
	"github.com/Strangebrewer/go-budget/bill"
	"github.com/Strangebrewer/go-budget/category"
	"github.com/Strangebrewer/go-budget/middleware"
	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/Strangebrewer/go-budget/transaction"
)

func Routes(
	accountStore *account.Store,
	billStore *bill.Store,
	categoryStore *category.Store,
	transactionStore *transaction.Store,
	tc *tracer.Client,
	audience string,
) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(accountStore, billStore, categoryStore, transactionStore, tc)
	r.With(middleware.RequirePubSubOIDC(audience)).Post("/demo-registered", h.HandleDemoRegistered)
	return r
}
