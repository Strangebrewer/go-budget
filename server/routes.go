package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Strangebrewer/go-budget/account"
	"github.com/Strangebrewer/go-budget/app"
	"github.com/Strangebrewer/go-budget/bill"
	"github.com/Strangebrewer/go-budget/category"
	"github.com/Strangebrewer/go-budget/health"
	"github.com/Strangebrewer/go-budget/transaction"
)

func registerRoutes(r chi.Router, application *app.Application, authMiddleware func(http.Handler) http.Handler) {
	r.Get("/health", health.Handler)

	r.With(authMiddleware).Group(func(r chi.Router) {
		r.Mount("/accounts", account.Routes(application.AccountStore, application.Tracer))
		r.Mount("/categories", category.Routes(application.CategoryStore, application.Tracer))
		r.Mount("/bills", bill.Routes(application.BillStore, application.TransactionStore, application.Tracer))
		r.Mount("/transactions", transaction.Routes(application.TransactionStore, application.Tracer))
	})
}
