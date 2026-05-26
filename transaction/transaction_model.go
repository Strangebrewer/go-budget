package transaction

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeCredit TransactionType = "credit"
	TransactionTypeDebit  TransactionType = "debit"
)

type Transaction struct {
	ID            string          `json:"id"`
	UserID        string          `json:"userId"`
	SourceID      *string         `json:"sourceId"`
	DestinationID *string         `json:"destinationId"`
	BillID        *string         `json:"billId"`
	CategoryID    *string         `json:"categoryId"`
	Amount        int32           `json:"amount"`
	Month         string          `json:"month"`
	Description   string          `json:"description"`
	Income        bool            `json:"income"`
	Owner         string          `json:"owner"`
	Shared        bool            `json:"shared"`
	Type          TransactionType `json:"type"`
	ExpiresAt     *time.Time      `json:"expiresAt,omitempty"`
}

type CreateTransactionRequest struct {
	SourceID      string          `json:"sourceId"`
	DestinationID string          `json:"destinationId"`
	BillID        string          `json:"billId"`
	CategoryID    string          `json:"categoryId"`
	Amount        int32           `json:"amount"`
	Month         string          `json:"month"`
	Description   string          `json:"description"`
	Income        bool            `json:"income"`
	Owner         string          `json:"owner"`
	Shared        bool            `json:"shared"`
	Type          TransactionType `json:"type"`
}

type UpdateTransactionRequest struct {
	SourceID      string          `json:"sourceId"`
	DestinationID string          `json:"destinationId"`
	BillID        string          `json:"billId"`
	CategoryID    string          `json:"categoryId"`
	Amount        int32           `json:"amount"`
	Month         string          `json:"month"`
	Description   string          `json:"description"`
	Income        bool            `json:"income"`
	Owner         string          `json:"owner"`
	Shared        bool            `json:"shared"`
	Type          TransactionType `json:"type"`
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ownerOrDefault(s string) string {
	if s == "" {
		return "mine"
	}
	return s
}

func validateType(t TransactionType) error {
	if t != TransactionTypeCredit && t != TransactionTypeDebit {
		return fmt.Errorf("invalid transaction type %q, must be %q or %q", t, TransactionTypeCredit, TransactionTypeDebit)
	}
	return nil
}

func typeOrDefault(t TransactionType) TransactionType {
	if t == "" {
		return TransactionTypeDebit
	}
	return t
}

func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}
