package game

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository interface {
	Save(ctx context.Context, doc *GameDocument) error
	FindByID(ctx context.Context, id string) (*GameDocument, error)
	FindUserGames(ctx context.Context, userID string, limit, skip int) ([]GameDocument, error)
	FindRecentGames(ctx context.Context, limit int) ([]GameDocument, error)
	EnsureIndexes(ctx context.Context) error
}

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: db.Collection("games"),
	}
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "white.userId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "black.userId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "startedAt", Value: -1}},
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (r *MongoRepository) Save(ctx context.Context, doc *GameDocument) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": doc.ID}
	_, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": doc}, opts)
	return err
}

func (r *MongoRepository) FindByID(ctx context.Context, id string) (*GameDocument, error) {
	var doc GameDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *MongoRepository) FindUserGames(ctx context.Context, userID string, limit, skip int) ([]GameDocument, error) {
	if limit <= 0 {
		limit = 20
	}
	filter := bson.M{
		"$or": []bson.M{
			{"white.userId": userID},
			{"black.userId": userID},
		},
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "startedAt", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(skip))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []GameDocument
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *MongoRepository) FindRecentGames(ctx context.Context, limit int) ([]GameDocument, error) {
	if limit <= 0 {
		limit = 20
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "startedAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []GameDocument
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}
