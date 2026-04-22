package transaction

import (
	"fmt"
	"time"

	"github.com/google/uuid"
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

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func validateDate(s string) error {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("invalid date %q, expected YYYY-MM-DD: %w", s, err)
	}
	return nil
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
