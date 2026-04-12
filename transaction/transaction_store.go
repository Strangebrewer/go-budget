package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/Strangebrewer/go-budget/db/generated"
)

var ErrNotFound = errors.New("transaction not found")

type Store struct {
	q *db.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: db.New(pool)}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]db.Transaction, error) {
	rows, err := s.q.GetAllTransactions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get all transactions: %w", err)
	}
	return rows, nil
}

// GetByBillMonths fetches transactions for the given month plus the two preceding months.
// Used by the bill handler to attach recent transactions to each bill.
func (s *Store) GetByBillMonths(ctx context.Context, userID uuid.UUID, month string) ([]db.Transaction, error) {
	months := threeMonthWindow(month)
	rows, err := s.q.GetTransactionsByBillMonths(ctx, db.GetTransactionsByBillMonthsParams{
		UserID:  userID,
		Column2: months,
	})
	if err != nil {
		return nil, fmt.Errorf("get transactions by bill months: %w", err)
	}
	return rows, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (db.Transaction, error) {
	t, err := s.q.GetTransactionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Transaction{}, ErrNotFound
		}
		return db.Transaction{}, fmt.Errorf("get transaction: %w", err)
	}
	return t, nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateTransactionRequest) (db.Transaction, error) {
	return s.create(ctx, userID, req)
}

// CreateFromBill is called by the bill handler's PayBill endpoint.
func (s *Store) CreateFromBill(ctx context.Context, userID uuid.UUID, req CreateTransactionRequest) (db.Transaction, error) {
	return s.create(ctx, userID, req)
}

func (s *Store) create(ctx context.Context, userID uuid.UUID, req CreateTransactionRequest) (db.Transaction, error) {
	id, err := newID()
	if err != nil {
		return db.Transaction{}, fmt.Errorf("generate id: %w", err)
	}

	sourceID, err := parsePgtypeUUID(req.SourceID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("source_id: %w", err)
	}
	destinationID, err := parsePgtypeUUID(req.DestinationID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("destination_id: %w", err)
	}
	billID, err := parsePgtypeUUID(req.BillID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("bill_id: %w", err)
	}
	categoryID, err := parsePgtypeUUID(req.CategoryID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("category_id: %w", err)
	}
	date, err := parsePgtypeDate(req.Date)
	if err != nil {
		return db.Transaction{}, err
	}

	now := time.Now().UTC()
	t, err := s.q.CreateTransaction(ctx, db.CreateTransactionParams{
		ID:            id,
		UserID:        userID,
		SourceID:      sourceID,
		DestinationID: destinationID,
		BillID:        billID,
		CategoryID:    categoryID,
		Amount:        req.Amount,
		BillMonth:     pgtypeText(req.BillMonth),
		Date:          date,
		Description:   req.Description,
		Income:        req.Income,
		Owner:         ownerOrDefault(req.Owner),
		Shared:        req.Shared,
		Type:          typeOrDefault(req.Type),
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return db.Transaction{}, fmt.Errorf("create transaction: %w", err)
	}
	return t, nil
}

func (s *Store) Update(ctx context.Context, id uuid.UUID, req UpdateTransactionRequest) (db.Transaction, error) {
	sourceID, err := parsePgtypeUUID(req.SourceID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("source_id: %w", err)
	}
	destinationID, err := parsePgtypeUUID(req.DestinationID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("destination_id: %w", err)
	}
	billID, err := parsePgtypeUUID(req.BillID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("bill_id: %w", err)
	}
	categoryID, err := parsePgtypeUUID(req.CategoryID)
	if err != nil {
		return db.Transaction{}, fmt.Errorf("category_id: %w", err)
	}
	date, err := parsePgtypeDate(req.Date)
	if err != nil {
		return db.Transaction{}, err
	}

	t, err := s.q.UpdateTransaction(ctx, db.UpdateTransactionParams{
		ID:            id,
		SourceID:      sourceID,
		DestinationID: destinationID,
		BillID:        billID,
		CategoryID:    categoryID,
		Amount:        req.Amount,
		BillMonth:     pgtypeText(req.BillMonth),
		Date:          date,
		Description:   req.Description,
		Income:        req.Income,
		Owner:         ownerOrDefault(req.Owner),
		Shared:        req.Shared,
		Type:          typeOrDefault(req.Type),
		UpdatedAt:     time.Now().UTC(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Transaction{}, ErrNotFound
		}
		return db.Transaction{}, fmt.Errorf("update transaction: %w", err)
	}
	return t, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteTransaction(ctx, id); err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	return nil
}

// threeMonthWindow returns the given month and the two months preceding it.
func threeMonthWindow(month string) []string {
	var y, m int
	fmt.Sscanf(month, "%d-%d", &y, &m)

	months := make([]string, 3)
	for i := 0; i < 3; i++ {
		months[i] = fmt.Sprintf("%d-%02d", y, m)
		m--
		if m == 0 {
			m = 12
			y--
		}
	}
	return months
}

// UUIDtoPgtypeUUID converts a uuid.UUID to pgtype.UUID for use by the bill handler.
func UUIDtoPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
