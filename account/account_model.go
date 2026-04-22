package account

import "github.com/google/uuid"

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
