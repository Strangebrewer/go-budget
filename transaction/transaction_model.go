package transaction

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/Strangebrewer/go-budget/db/generated"
)

type Transaction struct {
	ID            string  `json:"id"`
	UserID        string  `json:"userId"`
	SourceID      *string `json:"sourceId"`
	DestinationID *string `json:"destinationId"`
	BillID        *string `json:"billId"`
	CategoryID    *string `json:"categoryId"`
	Amount        int32   `json:"amount"`
	BillMonth     *string `json:"billMonth"`
	Date          string  `json:"date"`
	Description   string  `json:"description"`
	Income        bool    `json:"income"`
	Owner         string  `json:"owner"`
	Shared        bool    `json:"shared"`
	Type          string  `json:"type"`
}

type CreateTransactionRequest struct {
	SourceID      string `json:"sourceId"`
	DestinationID string `json:"destinationId"`
	BillID        string `json:"billId"`
	CategoryID    string `json:"categoryId"`
	Amount        int32  `json:"amount"`
	BillMonth     string `json:"billMonth"`
	Date          string `json:"date"`
	Description   string `json:"description"`
	Income        bool   `json:"income"`
	Owner         string `json:"owner"`
	Shared        bool   `json:"shared"`
	Type          string `json:"type"`
}

type UpdateTransactionRequest struct {
	SourceID      string `json:"sourceId"`
	DestinationID string `json:"destinationId"`
	BillID        string `json:"billId"`
	CategoryID    string `json:"categoryId"`
	Amount        int32  `json:"amount"`
	BillMonth     string `json:"billMonth"`
	Date          string `json:"date"`
	Description   string `json:"description"`
	Income        bool   `json:"income"`
	Owner         string `json:"owner"`
	Shared        bool   `json:"shared"`
	Type          string `json:"type"`
}


func ToTransaction(t db.Transaction) Transaction {
	return Transaction{
		ID:            t.ID.String(),
		UserID:        t.UserID.String(),
		SourceID:      uuidPtr(t.SourceID),
		DestinationID: uuidPtr(t.DestinationID),
		BillID:        uuidPtr(t.BillID),
		CategoryID:    uuidPtr(t.CategoryID),
		Amount:        t.Amount,
		BillMonth:     textPtr(t.BillMonth),
		Date:          formatDate(t.Date),
		Description:   t.Description,
		Income:        t.Income,
		Owner:         t.Owner,
		Shared:        t.Shared,
		Type:          t.Type,
	}
}

func ToTransactions(rows []db.Transaction) []Transaction {
	out := make([]Transaction, len(rows))
	for i, t := range rows {
		out[i] = ToTransaction(t)
	}
	return out
}

func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func formatDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
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

func parsePgtypeDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD: %w", s, err)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func pgtypeText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func ownerOrDefault(s string) string {
	if s == "" {
		return "mine"
	}
	return s
}

func typeOrDefault(s string) string {
	if s == "" {
		return "expense"
	}
	return s
}

func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}
