package user

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ErrNotFound is returned when a requested user document does not exist.
var ErrNotFound = errors.New("user not found")


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

// PublicUser is the sanitized DTO returned by public endpoints, strictly omitting
// email, password hashes, verification state, and internal timestamps.
type PublicUser struct {
	ID               primitive.ObjectID `json:"id"`
	Username         string             `json:"username"`
	Avatar           string             `json:"avatar"`
	Bio              string             `json:"bio"`
	Country          string             `json:"country"`
	ChessComUsername string             `json:"chessComUsername,omitempty"`
	LichessUsername  string             `json:"lichessUsername,omitempty"`
	CreatedAt        time.Time          `json:"createdAt"`
}

func (u *User) ToPublic() PublicUser {
	return PublicUser{
		ID:               u.ID,
		Username:         u.Username,
		Avatar:           u.Avatar,
		Bio:              u.Bio,
		Country:          u.Country,
		ChessComUsername: u.ChessComUsername,
		LichessUsername:  u.LichessUsername,
		CreatedAt:        u.CreatedAt,
	}
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]*User, error)
	DeleteAllUsers(ctx context.Context) error
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

func (r *MongoRepository) SearchUsers(ctx context.Context, query string, limit int) ([]*User, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	filter := bson.M{}
	if query != "" {
		filter = bson.M{
			"username": bson.M{
				"$regex": primitive.Regex{Pattern: regexp.QuoteMeta(query), Options: "i"},
			},
		}
	}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "username", Value: 1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var users []*User
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}
	if users == nil {
		users = []*User{}
	}
	return users, nil
}

func (r *MongoRepository) DeleteAllUsers(ctx context.Context) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{})
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
