package account

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

var ErrNotFound = errors.New("account not found")

type Store struct {
	q *db.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: db.New(pool)}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]db.Account, error) {
	rows, err := s.q.GetAllAccounts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get all accounts: %w", err)
	}
	return rows, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (db.Account, error) {
	a, err := s.q.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Account{}, ErrNotFound
		}
		return db.Account{}, fmt.Errorf("get account: %w", err)
	}
	return a, nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateAccountRequest) (db.Account, error) {
	id, err := newID()
	if err != nil {
		return db.Account{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	a, err := s.q.CreateAccount(ctx, db.CreateAccountParams{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Balance:     req.Balance,
		Owner:       ownerOrDefault(req.Owner),
		Status:      "active",
		Type:        typeOrDefault(req.Type),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return db.Account{}, fmt.Errorf("create account: %w", err)
	}
	return a, nil
}

func (s *Store) Update(ctx context.Context, id uuid.UUID, req UpdateAccountRequest) (db.Account, error) {
	a, err := s.q.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Balance:     req.Balance,
		Owner:       ownerOrDefault(req.Owner),
		Status:      req.Status,
		Type:        typeOrDefault(req.Type),
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Account{}, ErrNotFound
		}
		return db.Account{}, fmt.Errorf("update account: %w", err)
	}
	return a, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteAccount(ctx, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}
