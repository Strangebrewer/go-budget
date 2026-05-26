package bill

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

var ErrNotFound = errors.New("bill not found")

type billDoc struct {
	ID          string     `bson:"_id"`
	UserID      string     `bson:"userId"`
	SourceID    string     `bson:"sourceId"`
	CategoryID  *string    `bson:"categoryId"`
	Name        string     `bson:"name"`
	Description string     `bson:"description"`
	Owner       string     `bson:"owner"`
	Shared      bool       `bson:"shared"`
	Status      string     `bson:"status"`
	ExpiresAt   *time.Time `bson:"expiresAt,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt"`
}

func (d billDoc) toDomain() Bill {
	return Bill{
		ID:          d.ID,
		UserID:      d.UserID,
		SourceID:    d.SourceID,
		CategoryID:  d.CategoryID,
		Name:        d.Name,
		Description: d.Description,
		Owner:       d.Owner,
		Shared:      d.Shared,
		Status:      d.Status,
		ExpiresAt:   d.ExpiresAt,
	}
}

type Store struct {
	col          *mongo.Collection
	transactions *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{
		col:          db.Collection("bills"),
		transactions: db.Collection("transactions"),
	}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]Bill, error) {
	cursor, err := s.col.Find(ctx, bson.D{{Key: "userId", Value: userID.String()}})
	if err != nil {
		return nil, fmt.Errorf("get all bills: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []billDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode bills: %w", err)
	}

	bills := make([]Bill, len(docs))
	for i, d := range docs {
		bills[i] = d.toDomain()
	}
	return bills, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (Bill, error) {
	var doc billDoc
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: id.String()}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Bill{}, ErrNotFound
		}
		return Bill{}, fmt.Errorf("get bill: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateBillRequest, expiresAt *time.Time) (Bill, error) {
	id, err := newID()
	if err != nil {
		return Bill{}, fmt.Errorf("generate id: %w", err)
	}

	if _, err := uuid.Parse(req.SourceID); err != nil {
		return Bill{}, fmt.Errorf("invalid source_id: %w", err)
	}

	var categoryID *string
	if req.CategoryID != "" {
		if _, err := uuid.Parse(req.CategoryID); err != nil {
			return Bill{}, fmt.Errorf("invalid category_id: %w", err)
		}
		catIDStr := req.CategoryID
		categoryID = &catIDStr
	}

	now := time.Now().UTC()
	doc := billDoc{
		ID:          id.String(),
		UserID:      userID.String(),
		SourceID:    req.SourceID,
		CategoryID:  categoryID,
		Name:        req.Name,
		Description: req.Description,
		Owner:       ownerOrDefault(req.Owner),
		Shared:      true,
		Status:      "active",
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		return Bill{}, fmt.Errorf("create bill: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.col.CountDocuments(ctx, bson.D{{Key: "userId", Value: userID.String()}})
	if err != nil {
		return 0, fmt.Errorf("count bills: %w", err)
	}
	return count, nil
}

func (s *Store) Update(ctx context.Context, id, userID uuid.UUID, req UpdateBillRequest) (Bill, error) {
	if _, err := uuid.Parse(req.SourceID); err != nil {
		return Bill{}, fmt.Errorf("invalid source_id: %w", err)
	}

	var categoryID *string
	if req.CategoryID != "" {
		if _, err := uuid.Parse(req.CategoryID); err != nil {
			return Bill{}, fmt.Errorf("invalid category_id: %w", err)
		}
		catIDStr := req.CategoryID
		categoryID = &catIDStr
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	filter := bson.D{{Key: "_id", Value: id.String()}, {Key: "userId", Value: userID.String()}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "sourceId", Value: req.SourceID},
		{Key: "categoryId", Value: categoryID},
		{Key: "name", Value: req.Name},
		{Key: "description", Value: req.Description},
		{Key: "owner", Value: ownerOrDefault(req.Owner)},
		{Key: "status", Value: status},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}}

	var doc billDoc
	err := s.col.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Bill{}, ErrNotFound
		}
		return Bill{}, fmt.Errorf("update bill: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Delete(ctx context.Context, id, userID uuid.UUID) error {
	idStr := id.String()

	// Delete the bill first — cascade only if ownership is confirmed.
	result, err := s.col.DeleteOne(ctx, bson.D{
		{Key: "_id", Value: idStr},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("delete bill: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}

	// Null out billId on transactions that reference this bill.
	_, err = s.transactions.UpdateMany(ctx,
		bson.D{{Key: "billId", Value: idStr}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "billId", Value: nil}}}},
	)
	if err != nil {
		return fmt.Errorf("null billId on transactions: %w", err)
	}
	return nil
}
