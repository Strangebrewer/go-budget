package bill

import (
	"github.com/Strangebrewer/go-budget/transaction"
	"github.com/google/uuid"
)

type Bill struct {
	ID          string  `json:"id"`
	UserID      string  `json:"userId"`
	SourceID    string  `json:"sourceId"`
	CategoryID  *string `json:"categoryId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	DueDay      int32   `json:"dueDay"`
	Owner       string  `json:"owner"`
	Shared      bool    `json:"shared"`
	Status      string  `json:"status"`
}

type BillResponse struct {
	Bill
	Transactions []transaction.Transaction `json:"transactions"`
}

type CreateBillRequest struct {
	SourceID    string `json:"sourceId"`
	CategoryID  string `json:"categoryId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DueDay      int32  `json:"dueDay"`
	Owner       string `json:"owner"`
}

type UpdateBillRequest struct {
	SourceID    string `json:"sourceId"`
	CategoryID  string `json:"categoryId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DueDay      int32  `json:"dueDay"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
}

type PayBillRequest struct {
	SourceID    string `json:"sourceId"`
	Amount      int32  `json:"amount"`
	Month       string `json:"month"`
	Description string `json:"description"`
}

func ownerOrDefault(s string) string {
	if s == "" {
		return "mine"
	}
	return s
}

func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}
