package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("transaction not found")

type transactionDoc struct {
	ID            string    `bson:"_id"`
	UserID        string    `bson:"userId"`
	SourceID      *string   `bson:"sourceId"`
	DestinationID *string   `bson:"destinationId"`
	BillID        *string   `bson:"billId"`
	CategoryID    *string   `bson:"categoryId"`
	Amount        int32     `bson:"amount"`
	BillMonth     *string   `bson:"billMonth"`
	Date          string    `bson:"date"`
	Description   string    `bson:"description"`
	Income        bool      `bson:"income"`
	Owner         string    `bson:"owner"`
	Shared        bool      `bson:"shared"`
	Type          string    `bson:"type"`
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
}

func (d transactionDoc) toDomain() Transaction {
	return Transaction{
		ID:            d.ID,
		UserID:        d.UserID,
		SourceID:      d.SourceID,
		DestinationID: d.DestinationID,
		BillID:        d.BillID,
		CategoryID:    d.CategoryID,
		Amount:        d.Amount,
		BillMonth:     d.BillMonth,
		Date:          d.Date,
		Description:   d.Description,
		Income:        d.Income,
		Owner:         d.Owner,
		Shared:        d.Shared,
		Type:          d.Type,
	}
}

type Store struct {
	col *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{col: db.Collection("transactions")}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]Transaction, error) {
	cursor, err := s.col.Find(ctx,
		bson.D{{Key: "userId", Value: userID.String()}},
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get all transactions: %w", err)
	}
	return decodeCursor(ctx, cursor)
}

// GetByBillMonths fetches transactions for the given month plus the two preceding months.
func (s *Store) GetByBillMonths(ctx context.Context, userID uuid.UUID, month string) ([]Transaction, error) {
	months := threeMonthWindow(month)
	cursor, err := s.col.Find(ctx,
		bson.D{
			{Key: "userId", Value: userID.String()},
			{Key: "billMonth", Value: bson.D{{Key: "$in", Value: months}}},
		},
		options.Find().SetSort(bson.D{{Key: "billMonth", Value: 1}, {Key: "date", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get transactions by bill months: %w", err)
	}
	return decodeCursor(ctx, cursor)
}

func (s *Store) GetByCategories(ctx context.Context, userID uuid.UUID, categoryIDs []uuid.UUID) ([]Transaction, error) {
	cursor, err := s.col.Find(ctx,
		bson.D{
			{Key: "userId", Value: userID.String()},
			{Key: "categoryId", Value: bson.D{{Key: "$in", Value: uuidStrings(categoryIDs)}}},
		},
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get transactions by categories: %w", err)
	}
	return decodeCursor(ctx, cursor)
}

func (s *Store) GetByMonthAndCategories(ctx context.Context, userID uuid.UUID, month string, categoryIDs []uuid.UUID) ([]Transaction, error) {
	start, end := monthDateRange(month)
	cursor, err := s.col.Find(ctx,
		bson.D{
			{Key: "userId", Value: userID.String()},
			{Key: "date", Value: bson.D{{Key: "$gte", Value: start}, {Key: "$lt", Value: end}}},
			{Key: "categoryId", Value: bson.D{{Key: "$in", Value: uuidStrings(categoryIDs)}}},
		},
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get transactions by month and categories: %w", err)
	}
	return decodeCursor(ctx, cursor)
}

func (s *Store) GetIncome(ctx context.Context, userID uuid.UUID) ([]Transaction, error) {
	cursor, err := s.col.Find(ctx,
		bson.D{
			{Key: "userId", Value: userID.String()},
			{Key: "income", Value: true},
		},
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get income transactions: %w", err)
	}
	return decodeCursor(ctx, cursor)
}

func (s *Store) GetIncomeByMonth(ctx context.Context, userID uuid.UUID, month string) ([]Transaction, error) {
	start, end := monthDateRange(month)
	cursor, err := s.col.Find(ctx,
		bson.D{
			{Key: "userId", Value: userID.String()},
			{Key: "income", Value: true},
			{Key: "date", Value: bson.D{{Key: "$gte", Value: start}, {Key: "$lt", Value: end}}},
		},
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get income transactions by month: %w", err)
	}
	return decodeCursor(ctx, cursor)
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (Transaction, error) {
	var doc transactionDoc
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: id.String()}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Transaction{}, ErrNotFound
		}
		return Transaction{}, fmt.Errorf("get transaction: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateTransactionRequest) (Transaction, error) {
	return s.create(ctx, userID, req)
}

// CreateFromBill is called by the bill handler's PayBill endpoint.
func (s *Store) CreateFromBill(ctx context.Context, userID uuid.UUID, req CreateTransactionRequest) (Transaction, error) {
	return s.create(ctx, userID, req)
}

func (s *Store) create(ctx context.Context, userID uuid.UUID, req CreateTransactionRequest) (Transaction, error) {
	if err := validateDate(req.Date); err != nil {
		return Transaction{}, err
	}

	id, err := newID()
	if err != nil {
		return Transaction{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	doc := transactionDoc{
		ID:            id.String(),
		UserID:        userID.String(),
		SourceID:      nilIfEmpty(req.SourceID),
		DestinationID: nilIfEmpty(req.DestinationID),
		BillID:        nilIfEmpty(req.BillID),
		CategoryID:    nilIfEmpty(req.CategoryID),
		Amount:        req.Amount,
		BillMonth:     nilIfEmpty(req.BillMonth),
		Date:          req.Date,
		Description:   req.Description,
		Income:        req.Income,
		Owner:         ownerOrDefault(req.Owner),
		Shared:        req.Shared,
		Type:          typeOrDefault(req.Type),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		return Transaction{}, fmt.Errorf("create transaction: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Update(ctx context.Context, id uuid.UUID, req UpdateTransactionRequest) (Transaction, error) {
	if err := validateDate(req.Date); err != nil {
		return Transaction{}, err
	}

	filter := bson.D{{Key: "_id", Value: id.String()}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "sourceId", Value: nilIfEmpty(req.SourceID)},
		{Key: "destinationId", Value: nilIfEmpty(req.DestinationID)},
		{Key: "billId", Value: nilIfEmpty(req.BillID)},
		{Key: "categoryId", Value: nilIfEmpty(req.CategoryID)},
		{Key: "amount", Value: req.Amount},
		{Key: "billMonth", Value: nilIfEmpty(req.BillMonth)},
		{Key: "date", Value: req.Date},
		{Key: "description", Value: req.Description},
		{Key: "income", Value: req.Income},
		{Key: "owner", Value: ownerOrDefault(req.Owner)},
		{Key: "shared", Value: req.Shared},
		{Key: "type", Value: typeOrDefault(req.Type)},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}}

	var doc transactionDoc
	err := s.col.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Transaction{}, ErrNotFound
		}
		return Transaction{}, fmt.Errorf("update transaction: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := s.col.DeleteOne(ctx, bson.D{{Key: "_id", Value: id.String()}})
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
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

// monthDateRange returns YYYY-MM-DD strings for the start of (month-2) and start of (month+1).
// Dates are stored as YYYY-MM-DD strings, so string comparison is correct for ISO dates.
func monthDateRange(month string) (start, end string) {
	var y, m int
	fmt.Sscanf(month, "%d-%d", &y, &m)
	startTime := time.Date(y, time.Month(m-2), 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(y, time.Month(m+1), 1, 0, 0, 0, 0, time.UTC)
	return startTime.Format("2006-01-02"), endTime.Format("2006-01-02")
}

func uuidStrings(ids []uuid.UUID) []string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return strs
}

func decodeCursor(ctx context.Context, cursor *mongo.Cursor) ([]Transaction, error) {
	defer cursor.Close(ctx)
	var docs []transactionDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode transactions: %w", err)
	}
	txns := make([]Transaction, len(docs))
	for i, d := range docs {
		txns[i] = d.toDomain()
	}
	return txns, nil
}
