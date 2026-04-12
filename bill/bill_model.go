package bill

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/Strangebrewer/go-budget/db/generated"
	"github.com/Strangebrewer/go-budget/transaction"
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
	Shared      bool   `json:"shared"`
}

type UpdateBillRequest struct {
	SourceID    string `json:"sourceId"`
	CategoryID  string `json:"categoryId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DueDay      int32  `json:"dueDay"`
	Owner       string `json:"owner"`
	Shared      bool   `json:"shared"`
	Status      string `json:"status"`
}

type PayBillRequest struct {
	SourceID    string `json:"sourceId"`
	Amount      int32  `json:"amount"`
	BillMonth   string `json:"billMonth"`
	Date        string `json:"date"`
	Description string `json:"description"`
}

func toBill(b db.Bill) Bill {
	return Bill{
		ID:          b.ID.String(),
		UserID:      b.UserID.String(),
		SourceID:    b.SourceID.String(),
		CategoryID:  uuidPtr(b.CategoryID),
		Name:        b.Name,
		Description: b.Description,
		DueDay:      b.DueDay,
		Owner:       b.Owner,
		Shared:      b.Shared,
		Status:      b.Status,
	}
}

func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

func parsePgtypeUUID(s string) (pgtype.UUID, error) {
	if s == "" {
		return pgtype.UUID{Valid: false}, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
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
