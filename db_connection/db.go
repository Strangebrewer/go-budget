package db_connection

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(ctx context.Context, mongoURI string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to ping: %w", err)
	}

	database := client.Database("budget")

	accounts := database.Collection("accounts")
	_, err = accounts.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to create account indexes: %w", err)
	}

	bills := database.Collection("bills")
	_, err = bills.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to create bill indexes: %w", err)
	}

	categories := database.Collection("categories")
	_, err = categories.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to create category indexes: %w", err)
	}

	transactions := database.Collection("transactions")
	_, err = transactions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		{Keys: bson.D{{Key: "month", Value: 1}}},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to create transaction indexes: %w", err)
	}

	return client, database, nil
}
