package puzzle

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Puzzle struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FEN      string             `bson:"fen" json:"fen"`
	Solution []string           `bson:"solution" json:"solution"`
	Rating   int                `bson:"rating" json:"rating"`
	Themes   []string           `bson:"themes" json:"themes"`
}

type Repository interface {
	GetRandomPuzzle(ctx context.Context) (*Puzzle, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*Puzzle, error)
	SeedInitialPuzzles(ctx context.Context) error
}

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: db.Collection("puzzles"),
	}
}

func (r *MongoRepository) GetRandomPuzzle(ctx context.Context) (*Puzzle, error) {
	// Sample 1 puzzle randomly using mongo aggregation
	pipeline := mongo.Pipeline{
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []Puzzle
	if err := cursor.All(ctx, &list); err != nil || len(list) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return &list[0], nil
}

func (r *MongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*Puzzle, error) {
	var p Puzzle
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *MongoRepository) SeedInitialPuzzles(ctx context.Context) error {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err == nil && count > 0 {
		return nil
	}

	samplePuzzles := []interface{}{
		Puzzle{
			FEN:      "r1bqk2r/pppp1ppp/2n5/4p3/2B1P1n1/3P1N2/PPP2KPP/RNBQ3R w kq - 1 6",
			Solution: []string{"f2g3", "d7d6", "h2h3"},
			Rating:   1250,
			Themes:   []string{"defense", "king_safety"},
		},
		Puzzle{
			FEN:      "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 1",
			Solution: []string{"e1e8#"},
			Rating:   1000,
			Themes:   []string{"mateIn1", "backRankMate"},
		},
		Puzzle{
			FEN:      "r1b1k2r/ppppqppp/2n5/4n3/1bP2B2/5N2/PP1NPPPP/R2QKB1R w KQkq - 0 8",
			Solution: []string{"f3e5", "c6e5", "e2e3"},
			Rating:   1350,
			Themes:   []string{"opening", "skewer"},
		},
		Puzzle{
			FEN:      "r4rk1/ppp2ppp/2n5/3q4/3P4/5N2/PP1Q1PPP/R3R1K1 b - - 0 14",
			Solution: []string{"a8d8", "a1d1", "d8d6"},
			Rating:   1450,
			Themes:   []string{"middlegame", "positional"},
		},
	}

	_, err = r.collection.InsertMany(ctx, samplePuzzles, options.InsertMany())
	return err
}
