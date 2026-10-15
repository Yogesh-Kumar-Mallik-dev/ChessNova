package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := "mongodb://127.0.0.1:27017"
	if len(os.Args) > 1 && os.Args[1] != "" {
		mongoURI = os.Args[1]
	} else if env := os.Getenv("MONGODB_URI"); env != "" {
		mongoURI = env
	}

	dbName := "chess"
	if env := os.Getenv("MONGODB_DATABASE"); env != "" {
		dbName = env
	}

	log.Printf("Connecting to MongoDB at %s (db: %s)...", mongoURI, dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}

	db := client.Database(dbName)
	usersCol := db.Collection("users")

	countBefore, err := usersCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to count users: %v", err)
	}

	res, err := usersCol.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to delete users: %v", err)
	}

	fmt.Printf("Successfully purged 'users' collection! (Found: %d, Deleted: %d)\n", countBefore, res.DeletedCount)
}
