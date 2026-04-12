package account

import (
	"github.com/google/uuid"
	db "github.com/Strangebrewer/go-budget/db/generated"
)

type Account struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Balance     int32  `json:"balance"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	Type        string `json:"type"`
}

type CreateAccountRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Balance     int32  `json:"balance"`
	Owner       string `json:"owner"`
	Type        string `json:"type"`
}

type UpdateAccountRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Balance     int32  `json:"balance"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	Type        string `json:"type"`
}

func toAccount(a db.Account) Account {
	return Account{
		ID:          a.ID.String(),
		UserID:      a.UserID.String(),
		Name:        a.Name,
		Description: a.Description,
		Balance:     a.Balance,
		Owner:       a.Owner,
		Status:      a.Status,
		Type:        a.Type,
	}
}

func toAccounts(rows []db.Account) []Account {
	out := make([]Account, len(rows))
	for i, a := range rows {
		out[i] = toAccount(a)
	}
	return out
}

func ownerOrDefault(s string) string {
	if s == "" {
		return "mine"
	}
	return s
}

func typeOrDefault(s string) string {
	if s == "" {
		return "debt"
	}
	return s
}

func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}
