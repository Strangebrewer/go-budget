package category

import (
	"github.com/google/uuid"
	db "github.com/Strangebrewer/go-budget/db/generated"
)

type Category struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toCategory(c db.Category) Category {
	return Category{
		ID:          c.ID.String(),
		UserID:      c.UserID.String(),
		Name:        c.Name,
		Description: c.Description,
	}
}

func toCategories(rows []db.Category) []Category {
	out := make([]Category, len(rows))
	for i, c := range rows {
		out[i] = toCategory(c)
	}
	return out
}

func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}
