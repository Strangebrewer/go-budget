package bill

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

var ErrNotFound = errors.New("bill not found")

type Store struct {
	q *db.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: db.New(pool)}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]db.Bill, error) {
	rows, err := s.q.GetAllBills(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get all bills: %w", err)
	}
	return rows, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (db.Bill, error) {
	b, err := s.q.GetBillByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Bill{}, ErrNotFound
		}
		return db.Bill{}, fmt.Errorf("get bill: %w", err)
	}
	return b, nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateBillRequest) (db.Bill, error) {
	id, err := newID()
	if err != nil {
		return db.Bill{}, fmt.Errorf("generate id: %w", err)
	}

	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		return db.Bill{}, fmt.Errorf("invalid source_id: %w", err)
	}

	categoryID, err := parsePgtypeUUID(req.CategoryID)
	if err != nil {
		return db.Bill{}, fmt.Errorf("category_id: %w", err)
	}

	now := time.Now().UTC()
	b, err := s.q.CreateBill(ctx, db.CreateBillParams{
		ID:          id,
		UserID:      userID,
		SourceID:    sourceID,
		CategoryID:  categoryID,
		Name:        req.Name,
		Description: req.Description,
		DueDay:      req.DueDay,
		Owner:       ownerOrDefault(req.Owner),
		Shared:      req.Shared,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return db.Bill{}, fmt.Errorf("create bill: %w", err)
	}
	return b, nil
}

func (s *Store) Update(ctx context.Context, id uuid.UUID, req UpdateBillRequest) (db.Bill, error) {
	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		return db.Bill{}, fmt.Errorf("invalid source_id: %w", err)
	}

	categoryID, err := parsePgtypeUUID(req.CategoryID)
	if err != nil {
		return db.Bill{}, fmt.Errorf("category_id: %w", err)
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	b, err := s.q.UpdateBill(ctx, db.UpdateBillParams{
		ID:          id,
		SourceID:    sourceID,
		CategoryID:  categoryID,
		Name:        req.Name,
		Description: req.Description,
		DueDay:      req.DueDay,
		Owner:       ownerOrDefault(req.Owner),
		Shared:      req.Shared,
		Status:      status,
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Bill{}, ErrNotFound
		}
		return db.Bill{}, fmt.Errorf("update bill: %w", err)
	}
	return b, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteBill(ctx, id); err != nil {
		return fmt.Errorf("delete bill: %w", err)
	}
	return nil
}
