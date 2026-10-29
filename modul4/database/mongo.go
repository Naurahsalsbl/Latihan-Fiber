
package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"latihan-fiber/modul4/config"
)

func NewMongoClient() (*mongo.Client, error) {
	uri := config.GetEnv("MONGO_URI", "mongodb://localhost:27017")

	opts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second).
		SetMaxPoolSize(uint64(config.GetEnvInt("MONGO_MAX_POOL", 20)))

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat koneksi MongoDB: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("gagal menghubungi MongoDB: %w", err)
	}

	return client, nil
}
