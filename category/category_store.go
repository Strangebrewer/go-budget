package category

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/Strangebrewer/go-budget/db/generated"
)

var ErrNotFound = errors.New("category not found")

type Store struct {
	q *db.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: db.New(pool)}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]db.Category, error) {
	rows, err := s.q.GetAllCategories(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get all categories: %w", err)
	}
	return rows, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (db.Category, error) {
	c, err := s.q.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Category{}, ErrNotFound
		}
		return db.Category{}, fmt.Errorf("get category: %w", err)
	}
	return c, nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateCategoryRequest) (db.Category, error) {
	id, err := newID()
	if err != nil {
		return db.Category{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	c, err := s.q.CreateCategory(ctx, db.CreateCategoryParams{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return db.Category{}, fmt.Errorf("create category: %w", err)
	}
	return c, nil
}

func (s *Store) Update(ctx context.Context, id uuid.UUID, req UpdateCategoryRequest) (db.Category, error) {
	c, err := s.q.UpdateCategory(ctx, db.UpdateCategoryParams{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Category{}, ErrNotFound
		}
		return db.Category{}, fmt.Errorf("update category: %w", err)
	}
	return c, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteCategory(ctx, id); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}
