package app

import (
	"github.com/Strangebrewer/go-budget/account"
	"github.com/Strangebrewer/go-budget/bill"
	"github.com/Strangebrewer/go-budget/category"
	"github.com/Strangebrewer/go-budget/transaction"
)

type Application struct {
	AccountStore     *account.Store
	BillStore        *bill.Store
	CategoryStore    *category.Store
	TransactionStore *transaction.Store
}
