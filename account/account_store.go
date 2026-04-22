package account

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

var ErrNotFound = errors.New("account not found")

type accountDoc struct {
	ID          string    `bson:"_id"`
	UserID      string    `bson:"userId"`
	Name        string    `bson:"name"`
	Description string    `bson:"description"`
	Balance     int32     `bson:"balance"`
	Owner       string    `bson:"owner"`
	Status      string    `bson:"status"`
	Type        string    `bson:"type"`
	CreatedAt   time.Time `bson:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt"`
}

func (d accountDoc) toDomain() Account {
	return Account{
		ID:          d.ID,
		UserID:      d.UserID,
		Name:        d.Name,
		Description: d.Description,
		Balance:     d.Balance,
		Owner:       d.Owner,
		Status:      d.Status,
		Type:        AccountType(d.Type),
	}
}

type Store struct {
	col          *mongo.Collection
	bills        *mongo.Collection
	transactions *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{
		col:          db.Collection("accounts"),
		bills:        db.Collection("bills"),
		transactions: db.Collection("transactions"),
	}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]Account, error) {
	cursor, err := s.col.Find(ctx,
		bson.D{{Key: "userId", Value: userID.String()}},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get all accounts: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []accountDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode accounts: %w", err)
	}

	accounts := make([]Account, len(docs))
	for i, d := range docs {
		accounts[i] = d.toDomain()
	}
	return accounts, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (Account, error) {
	var doc accountDoc
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: id.String()}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Account{}, ErrNotFound
		}
		return Account{}, fmt.Errorf("get account: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateAccountRequest) (Account, error) {
	id, err := newID()
	if err != nil {
		return Account{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	doc := accountDoc{
		ID:          id.String(),
		UserID:      userID.String(),
		Name:        req.Name,
		Description: req.Description,
		Balance:     req.Balance,
		Owner:       ownerOrDefault(req.Owner),
		Status:      "active",
		Type:        typeOrDefault(req.Type),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		return Account{}, fmt.Errorf("create account: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Update(ctx context.Context, id uuid.UUID, req UpdateAccountRequest) (Account, error) {
	filter := bson.D{{Key: "_id", Value: id.String()}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: req.Name},
		{Key: "description", Value: req.Description},
		{Key: "balance", Value: req.Balance},
		{Key: "owner", Value: ownerOrDefault(req.Owner)},
		{Key: "status", Value: req.Status},
		{Key: "type", Value: typeOrDefault(req.Type)},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}}

	var doc accountDoc
	err := s.col.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Account{}, ErrNotFound
		}
		return Account{}, fmt.Errorf("update account: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	idStr := id.String()

	// Find bills sourced from this account so we can cascade their transactions.
	cursor, err := s.bills.Find(ctx, bson.D{{Key: "sourceId", Value: idStr}})
	if err != nil {
		return fmt.Errorf("find bills for cascade: %w", err)
	}
	var billDocs []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &billDocs); err != nil {
		return fmt.Errorf("decode bills for cascade: %w", err)
	}
	cursor.Close(ctx)

	if len(billDocs) > 0 {
		billIDs := make([]string, len(billDocs))
		for i, b := range billDocs {
			billIDs[i] = b.ID
		}
		_, err = s.transactions.UpdateMany(ctx,
			bson.D{{Key: "billId", Value: bson.D{{Key: "$in", Value: billIDs}}}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "billId", Value: nil}}}},
		)
		if err != nil {
			return fmt.Errorf("null billId on transactions: %w", err)
		}
		_, err = s.bills.DeleteMany(ctx, bson.D{{Key: "sourceId", Value: idStr}})
		if err != nil {
			return fmt.Errorf("delete bills for account: %w", err)
		}
	}

	_, err = s.transactions.UpdateMany(ctx,
		bson.D{{Key: "sourceId", Value: idStr}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "sourceId", Value: nil}}}},
	)
	if err != nil {
		return fmt.Errorf("null sourceId on transactions: %w", err)
	}

	_, err = s.transactions.UpdateMany(ctx,
		bson.D{{Key: "destinationId", Value: idStr}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "destinationId", Value: nil}}}},
	)
	if err != nil {
		return fmt.Errorf("null destinationId on transactions: %w", err)
	}

	result, err := s.col.DeleteOne(ctx, bson.D{{Key: "_id", Value: idStr}})
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
