package category

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

var ErrNotFound = errors.New("category not found")

type categoryDoc struct {
	ID          string     `bson:"_id"`
	UserID      string     `bson:"userId"`
	Name        string     `bson:"name"`
	Description string     `bson:"description"`
	ExpiresAt   *time.Time `bson:"expiresAt,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt"`
}

func (d categoryDoc) toDomain() Category {
	return Category{
		ID:          d.ID,
		UserID:      d.UserID,
		Name:        d.Name,
		Description: d.Description,
		ExpiresAt:   d.ExpiresAt,
	}
}

type Store struct {
	col *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{col: db.Collection("categories")}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]Category, error) {
	cursor, err := s.col.Find(ctx, bson.D{{Key: "userId", Value: userID.String()}})
	if err != nil {
		return nil, fmt.Errorf("get all categories: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []categoryDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode categories: %w", err)
	}

	categories := make([]Category, len(docs))
	for i, d := range docs {
		categories[i] = d.toDomain()
	}
	return categories, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (Category, error) {
	var doc categoryDoc
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: id.String()}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Category{}, ErrNotFound
		}
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateCategoryRequest, expiresAt *time.Time) (Category, error) {
	id, err := newID()
	if err != nil {
		return Category{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	doc := categoryDoc{
		ID:          id.String(),
		UserID:      userID.String(),
		Name:        req.Name,
		Description: req.Description,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.col.CountDocuments(ctx, bson.D{{Key: "userId", Value: userID.String()}})
	if err != nil {
		return 0, fmt.Errorf("count categories: %w", err)
	}
	return count, nil
}

func (s *Store) Update(ctx context.Context, id, userID uuid.UUID, req UpdateCategoryRequest) (Category, error) {
	filter := bson.D{{Key: "_id", Value: id.String()}, {Key: "userId", Value: userID.String()}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: req.Name},
		{Key: "description", Value: req.Description},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}}

	var doc categoryDoc
	err := s.col.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Category{}, ErrNotFound
		}
		return Category{}, fmt.Errorf("update category: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result, err := s.col.DeleteOne(ctx, bson.D{{Key: "_id", Value: id.String()}, {Key: "userId", Value: userID.String()}})
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
