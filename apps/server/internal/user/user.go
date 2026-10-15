package user

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username         string             `bson:"username" json:"username"`
	Email            string             `bson:"email" json:"email"`
	PasswordHash     string             `bson:"passwordHash" json:"-"`
	Avatar           string             `bson:"avatar" json:"avatar"`
	Bio              string             `bson:"bio" json:"bio"`
	Country          string             `bson:"country" json:"country"`
	ChessComUsername string             `bson:"chessComUsername" json:"chessComUsername"`
	LichessUsername  string             `bson:"lichessUsername" json:"lichessUsername"`
	EmailVerified    bool               `bson:"emailVerified" json:"emailVerified"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	UpdateProfile(ctx context.Context, id primitive.ObjectID, bio, country string) error
	UpdateAvatar(ctx context.Context, id primitive.ObjectID, avatarURL string) error
	UpdateLinkedAccounts(ctx context.Context, id primitive.ObjectID, chessComUsername, lichessUsername string) error
	EnsureIndexes(ctx context.Context) error
}

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: db.Collection("users"),
	}
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (r *MongoRepository) Create(ctx context.Context, u *User) error {
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	res, err := r.collection.InsertOne(ctx, u)
	if err != nil {
		return err
	}
	u.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MongoRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*User, error) {
	var u User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *MongoRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *MongoRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *MongoRepository) UpdateProfile(ctx context.Context, id primitive.ObjectID, bio, country string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"bio":       bio,
				"country":   country,
				"updatedAt": time.Now().UTC(),
			},
		},
	)
	return err
}

func (r *MongoRepository) UpdateAvatar(ctx context.Context, id primitive.ObjectID, avatarURL string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"avatar":    avatarURL,
				"updatedAt": time.Now().UTC(),
			},
		},
	)
	return err
}

func (r *MongoRepository) UpdateLinkedAccounts(ctx context.Context, id primitive.ObjectID, chessComUsername, lichessUsername string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"chessComUsername": chessComUsername,
				"lichessUsername":  lichessUsername,
				"updatedAt":        time.Now().UTC(),
			},
		},
	)
	return err
}
