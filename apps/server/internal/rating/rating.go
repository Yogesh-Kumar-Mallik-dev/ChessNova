package rating

import (
	"context"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Category string

const (
	Bullet    Category = "bullet"
	Blitz     Category = "blitz"
	Rapid     Category = "rapid"
	Classical Category = "classical"
)

func DetermineCategory(initialSeconds int) Category {
	mins := initialSeconds / 60
	switch {
	case mins < 3:
		return Bullet
	case mins < 10:
		return Blitz
	case mins < 30:
		return Rapid
	default:
		return Classical
	}
}

type UserRating struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Username  string             `bson:"username" json:"username"`
	Category  Category           `bson:"category" json:"category"`
	Rating    int                `bson:"rating" json:"rating"`
	Games     int                `bson:"games" json:"games"`
	Wins      int                `bson:"wins" json:"wins"`
	Losses    int                `bson:"losses" json:"losses"`
	Draws     int                `bson:"draws" json:"draws"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

const DefaultRating = 1200
const DefaultKFactor = 32

func CalculateElo(ratingA, ratingB int, scoreA float64, kFactor int) (newA, newB int, changeA, changeB int) {
	expA := 1.0 / (1.0 + math.Pow(10.0, float64(ratingB-ratingA)/400.0))
	expB := 1.0 - expA

	scoreB := 1.0 - scoreA

	deltaA := int(math.Round(float64(kFactor) * (scoreA - expA)))
	deltaB := int(math.Round(float64(kFactor) * (scoreB - expB)))

	return ratingA + deltaA, ratingB + deltaB, deltaA, deltaB
}

type Repository interface {
	GetRating(ctx context.Context, userID primitive.ObjectID, cat Category) (*UserRating, error)
	UpsertRating(ctx context.Context, ur *UserRating) error
	GetLeaderboard(ctx context.Context, cat Category, limit int) ([]UserRating, error)
	EnsureIndexes(ctx context.Context) error
}

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: db.Collection("ratings"),
	}
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "category", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}, {Key: "rating", Value: -1}},
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (r *MongoRepository) GetRating(ctx context.Context, userID primitive.ObjectID, cat Category) (*UserRating, error) {
	var ur UserRating
	err := r.collection.FindOne(ctx, bson.M{"userId": userID, "category": cat}).Decode(&ur)
	if err != nil {
		if errorsIsNotFound(err) {
			return &UserRating{
				UserID:   userID,
				Category: cat,
				Rating:   DefaultRating,
				Games:    0,
				Wins:     0,
				Losses:   0,
				Draws:    0,
			}, nil
		}
		return nil, err
	}
	return &ur, nil
}

func (r *MongoRepository) UpsertRating(ctx context.Context, ur *UserRating) error {
	ur.UpdatedAt = time.Now().UTC()
	filter := bson.M{"userId": ur.UserID, "category": ur.Category}
	update := bson.M{
		"$set": bson.M{
			"username":  ur.Username,
			"rating":    ur.Rating,
			"games":     ur.Games,
			"wins":      ur.Wins,
			"losses":    ur.Losses,
			"draws":     ur.Draws,
			"updatedAt": ur.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"userId":   ur.UserID,
			"category": ur.Category,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *MongoRepository) GetLeaderboard(ctx context.Context, cat Category, limit int) ([]UserRating, error) {
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().SetSort(bson.D{{Key: "rating", Value: -1}}).SetLimit(int64(limit))
	cursor, err := r.collection.Find(ctx, bson.M{"category": cat}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []UserRating
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func errorsIsNotFound(err error) bool {
	return err == mongo.ErrNoDocuments
}
