package account

import (
	"github.com/google/uuid"
	db "github.com/Strangebrewer/go-budget/db/generated"
)

type AccountType string

const (
	AccountTypeAsset AccountType = "asset"
	AccountTypeDebt  AccountType = "debt"
)

func (t AccountType) Valid() bool {
	return t == AccountTypeAsset || t == AccountTypeDebt
}

type Account struct {
	ID          string      `json:"id"`
	UserID      string      `json:"userId"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Balance     int32       `json:"balance"`
	Owner       string      `json:"owner"`
	Status      string      `json:"status"`
	Type        AccountType `json:"type"`
}

type CreateAccountRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Balance     int32       `json:"balance"`
	Owner       string      `json:"owner"`
	Type        AccountType `json:"type"`
}

type UpdateAccountRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Balance     int32       `json:"balance"`
	Owner       string      `json:"owner"`
	Status      string      `json:"status"`
	Type        AccountType `json:"type"`
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
		Type:        AccountType(a.Type),
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

func typeOrDefault(t AccountType) string {
	if t == "" {
		return string(AccountTypeDebt)
	}
	return string(t)
}

func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}
